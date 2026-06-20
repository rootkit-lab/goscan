package scripts

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"goscan/internal/paths"
)

type ScriptInfo struct {
	ID          string   `yaml:"id" json:"id"`
	Path        string   `yaml:"path" json:"path"`
	Label       string   `yaml:"label" json:"label"`
	EnvKeys     []string `yaml:"env_keys" json:"envKeys"`
	Interactive bool     `yaml:"interactive" json:"interactive"`
}

type Registry struct {
	Scripts []ScriptInfo `yaml:"scripts"`
}

type RunResult struct {
	ScriptID string
	ExitCode int
	Output   string
	Err      error
}

type EventEmitter func(event string, data any)

type Runner struct {
	repoRoot string
	registry Registry
	mu       sync.Mutex
	cancel   context.CancelFunc
	ptyFile  *os.File
	ptyCmd   *exec.Cmd
}

func NewRunner(repoRoot string) (*Runner, error) {
	regPath := filepath.Join(repoRoot, "scripts", "registry.yaml")
	data, err := os.ReadFile(regPath)
	if err != nil {
		return nil, err
	}
	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	return &Runner{repoRoot: repoRoot, registry: reg}, nil
}

func (r *Runner) List() []ScriptInfo {
	return r.registry.Scripts
}

func (r *Runner) Find(id string) (*ScriptInfo, error) {
	for i := range r.registry.Scripts {
		if r.registry.Scripts[i].ID == id {
			return &r.registry.Scripts[i], nil
		}
	}
	return nil, fmt.Errorf("script não encontrado: %s", id)
}

func LoadEnvKeys(envPath string) (map[string]string, error) {
	f, err := os.Open(envPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	keys := make(map[string]string)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		k := strings.TrimSpace(line[:idx])
		v := strings.TrimSpace(line[idx+1:])
		v = strings.Trim(v, `"'`)
		keys[k] = v
	}
	return keys, sc.Err()
}

func (r *Runner) CompatibleScripts(envPath string) ([]ScriptInfo, error) {
	keys, err := LoadEnvKeys(envPath)
	if err != nil {
		return nil, err
	}
	var out []ScriptInfo
	for _, s := range r.registry.Scripts {
		for _, ek := range s.EnvKeys {
			if keys[ek] != "" {
				out = append(out, s)
				break
			}
		}
	}
	return out, nil
}

func (r *Runner) Cancel() {
	r.killPTY()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
}

func (r *Runner) Run(ctx context.Context, scriptID, envPath string, emit EventEmitter, timeout time.Duration) RunResult {
	res := RunResult{ScriptID: scriptID}
	s, err := r.Find(scriptID)
	if err != nil {
		res.Err = err
		return res
	}

	absEnv, err := filepath.Abs(envPath)
	if err != nil {
		res.Err = err
		return res
	}

	scriptPath := filepath.Join(r.repoRoot, s.Path)
	if _, err := os.Stat(scriptPath); err != nil {
		res.Err = fmt.Errorf("script path: %w", err)
		return res
	}

	if !VenvReady(r.repoRoot) {
		res.Err = fmt.Errorf("venv em falta — corra make scripts-venv")
		res.ExitCode = 127
		if emit != nil {
			emit("script:stderr", venvMissingMessage(r.repoRoot))
			emit("script:exit", map[string]any{"scriptId": scriptID, "exitCode": 127})
		}
		return res
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	r.mu.Lock()
	r.cancel = cancel
	r.mu.Unlock()
	defer cancel()

	cmd := exec.CommandContext(runCtx, PythonExecutable(r.repoRoot), scriptPath, "--env", absEnv)
	cmd.Dir = filepath.Join(r.repoRoot, "scripts")
	cmd.Env = ScriptEnv(r.repoRoot)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	var buf strings.Builder
	pump := func(rdr io.Reader, tag string) {
		sc := bufio.NewScanner(rdr)
		for sc.Scan() {
			line := sc.Text()
			buf.WriteString(line + "\n")
			if emit != nil {
				emit(tag, line)
			}
		}
	}

	if err := cmd.Start(); err != nil {
		res.Err = err
		return res
	}
	go pump(stdout, "script:stdout")
	go pump(stderr, "script:stderr")

	err = cmd.Wait()
	res.Output = buf.String()
	if runCtx.Err() == context.DeadlineExceeded {
		res.Err = fmt.Errorf("timeout após %v", timeout)
		res.ExitCode = 124
		return res
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.Err = err
		}
	}
	if emit != nil {
		emit("script:exit", map[string]any{"scriptId": scriptID, "exitCode": res.ExitCode})
	}
	return res
}

func DefaultRunner() (*Runner, error) {
	root, err := paths.RepoRoot()
	if err != nil {
		return nil, err
	}
	return NewRunner(root)
}

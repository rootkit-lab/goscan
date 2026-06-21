package scripts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/creack/pty"
)

func (r *Runner) RunInteractive(ctx context.Context, scriptID, envPath string, emit EventEmitter) RunResult {
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

	r.Cancel()

	if !VenvReady(r.repoRoot) {
		if emit != nil {
			emit("terminal:start", map[string]string{"scriptId": scriptID, "label": s.Label})
			emit("terminal:data", venvMissingMessage(r.repoRoot))
			emit("terminal:exit", map[string]any{"scriptId": scriptID, "exitCode": 127})
		}
		res.ExitCode = 127
		res.Err = fmt.Errorf("venv em falta — corra make scripts-venv")
		return res
	}

	py := PythonExecutable(r.repoRoot)

	runCtx, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	r.cancel = cancel
	r.mu.Unlock()

	if emit != nil {
		emit("terminal:start", map[string]string{"scriptId": scriptID, "label": s.Label, "python": py})
	}

	cmd := exec.CommandContext(runCtx, py, "-u", scriptPath, "--env", absEnv)
	cmd.Dir = filepath.Join(r.repoRoot, "scripts")
	cmd.Env = ScriptEnv(r.repoRoot)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		res.Err = fmt.Errorf("pty: %w", err)
		if emit != nil {
			emit("terminal:data", fmt.Sprintf("Erro ao iniciar PTY: %v\r\n", err))
			emit("terminal:exit", map[string]any{"scriptId": scriptID, "exitCode": 1})
		}
		cancel()
		return res
	}

	r.mu.Lock()
	r.ptyFile = ptmx
	r.ptyCmd = cmd
	r.mu.Unlock()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, readErr := ptmx.Read(buf)
			if n > 0 && emit != nil {
				emit("terminal:data", string(buf[:n]))
			}
			if readErr != nil {
				break
			}
		}
	}()

	waitErr := cmd.Wait()
	_ = ptmx.Close()

	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	if waitErr != nil && runCtx.Err() != context.Canceled {
		if exitCode == 0 {
			if exitErr, ok := waitErr.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				res.Err = waitErr
			}
		}
	}

	r.mu.Lock()
	if r.ptyFile != nil {
		_ = r.ptyFile.Close()
		r.ptyFile = nil
	}
	r.ptyCmd = nil
	r.cancel = nil
	r.mu.Unlock()

	res.ExitCode = exitCode
	if emit != nil {
		emit("terminal:exit", map[string]any{"scriptId": scriptID, "exitCode": exitCode})
	}
	cancel()
	return res
}

func (r *Runner) WriteInput(data string) error {
	r.mu.Lock()
	pt := r.ptyFile
	r.mu.Unlock()
	if pt == nil {
		return fmt.Errorf("sem sessão terminal activa")
	}
	_, err := pt.Write([]byte(data))
	return err
}

func (r *Runner) ResizeTerminal(cols, rows uint16) error {
	r.mu.Lock()
	pt := r.ptyFile
	r.mu.Unlock()
	if pt == nil {
		return nil
	}
	return pty.Setsize(pt, &pty.Winsize{Cols: cols, Rows: rows})
}

func (r *Runner) killPTY() {
	r.mu.Lock()
	cmd := r.ptyCmd
	pt := r.ptyFile
	cancel := r.cancel
	r.ptyCmd = nil
	r.ptyFile = nil
	r.cancel = nil
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		time.AfterFunc(2*time.Second, func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		})
	}
	if pt != nil {
		_ = pt.Close()
	}
}

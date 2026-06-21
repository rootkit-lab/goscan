package scripts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func venvPythonPath(repoRoot string) string {
	return filepath.Join(repoRoot, "scripts", ".venv", "bin", "python")
}

func venvDir(repoRoot string) string {
	return filepath.Join(repoRoot, "scripts", ".venv")
}

// VenvReady indica se scripts/.venv/bin/python existe.
func VenvReady(repoRoot string) bool {
	py := venvPythonPath(repoRoot)
	st, err := os.Stat(py)
	return err == nil && !st.IsDir()
}

// PythonExecutable devolve o Python dos checkers (venv ou erro explícito).
func PythonExecutable(repoRoot string) string {
	py := venvPythonPath(repoRoot)
	if VenvReady(repoRoot) {
		return py
	}
	return "python3"
}

// ScriptEnv ambiente para subprocessos Python (PATH do venv, PYTHONUNBUFFERED).
func ScriptEnv(repoRoot string) []string {
	env := os.Environ()
	if VenvReady(repoRoot) {
		venv := venvDir(repoRoot)
		bin := filepath.Join(venv, "bin")
		env = append(env, "VIRTUAL_ENV="+venv)
		env = prependPath(env, "PATH", bin)
	}
	env = append(env, "PYTHONUNBUFFERED=1")
	return env
}

// BatchScriptEnv adds GOSCAN_BATCH for batch checkers.
func BatchScriptEnv(repoRoot string) []string {
	env := ScriptEnv(repoRoot)
	return append(env, "GOSCAN_BATCH=1")
}

func prependPath(env []string, key, prefix string) []string {
	out := make([]string, 0, len(env)+1)
	replaced := false
	prefixEq := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefixEq) {
			cur := strings.TrimPrefix(e, prefixEq)
			if cur != "" {
				out = append(out, key+"="+prefix+string(os.PathListSeparator)+cur)
			} else {
				out = append(out, key+"="+prefix)
			}
			replaced = true
		} else {
			out = append(out, e)
		}
	}
	if !replaced {
		out = append(out, key+"="+prefix)
	}
	return out
}

func venvMissingMessage(repoRoot string) string {
	return fmt.Sprintf(
		"\r\n\x1b[31mvenv Python em falta\x1b[0m\r\n"+
			"Execute na raiz do projeto:\r\n"+
			"  make scripts-venv\r\n"+
			"ou reinicie com:\r\n"+
			"  make dev-ui\r\n\r\n"+
			"(esperado: %s)\r\n",
		venvPythonPath(repoRoot),
	)
}

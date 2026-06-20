package paths

import (
	"os"
	"path/filepath"
)

// RepoRoot devolve a raiz do repo (go.mod + scripts/registry.yaml).
// Usa GOSCAN_REPO_ROOT se definido (make dev-ui).
func RepoRoot() (string, error) {
	if root := os.Getenv("GOSCAN_REPO_ROOT"); root != "" {
		if abs, err := filepath.Abs(root); err == nil && isRepoRoot(abs) {
			return abs, nil
		}
	}

	starts := []string{}
	if wd, err := os.Getwd(); err == nil {
		starts = append(starts, wd)
	}
	if exe, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(exe))
	}

	seen := map[string]bool{}
	for _, start := range starts {
		dir := start
		for {
			if seen[dir] {
				break
			}
			seen[dir] = true
			if isRepoRoot(dir) {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return "", os.ErrNotExist
}

func isRepoRoot(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "scripts", "registry.yaml")); err != nil {
		return false
	}
	return true
}

func DefaultDBPath(root string) string {
	return filepath.Join(root, "dominios.db")
}

func FindingsRoot(root string) string {
	return filepath.Join(root, "var", "findings")
}

func FindingsByDomain(root string) string {
	return filepath.Join(FindingsRoot(root), "by-domain")
}

func ScriptsDir(root string) string {
	return filepath.Join(root, "scripts")
}

func ArchiveDir(root string) string {
	return filepath.Join(root, "var", "archive")
}

func DevLogsDir(root string) string {
	return filepath.Join(root, "var", "dev", "logs")
}

func ConfigPath(root string) string {
	return filepath.Join(root, "config.yml")
}

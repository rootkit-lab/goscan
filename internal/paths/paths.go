package paths

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"goscan/internal/settings"
)

const (
	ModeDev  = "dev"
	ModeProd = "prod"
)

// RepoRoot devolve AppRoot (scripts/registry). Mantido por compatibilidade.
func RepoRoot() (string, error) {
	return AppRoot()
}

// Mode devolve "dev" ou "prod".
func Mode() string {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GOSCAN_MODE"))); v == ModeDev || v == ModeProd {
		return v
	}
	if _, ok := installRootFromExecutable(); ok {
		return ModeProd
	}
	if root := strings.TrimSpace(os.Getenv("GOSCAN_REPO_ROOT")); root != "" {
		return ModeDev
	}
	if _, err := findDevRepoRoot(); err == nil {
		return ModeDev
	}
	return ModeProd
}

func installRootFromExecutable() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	dir, err := filepath.EvalSymlinks(filepath.Dir(exe))
	if err != nil {
		dir = filepath.Dir(exe)
	}
	candidates := []string{
		dir,
		filepath.Dir(dir),
		filepath.Join(filepath.Dir(dir), ".."),
	}
	for _, c := range candidates {
		c = filepath.Clean(c)
		if isInstallAppRoot(c) {
			return c, true
		}
	}
	return "", false
}

// AppRoot — scripts, registry, binários instalados (dev = repo).
func AppRoot() (string, error) {
	if v := strings.TrimSpace(os.Getenv("GOSCAN_APP_ROOT")); v != "" {
		return filepath.Abs(v)
	}
	if root, ok := installRootFromExecutable(); ok {
		return root, nil
	}
	if root := strings.TrimSpace(os.Getenv("GOSCAN_REPO_ROOT")); root != "" {
		if abs, err := filepath.Abs(root); err == nil && isDevRepoRoot(abs) {
			return abs, nil
		}
	}
	if Mode() == ModeDev {
		return findDevRepoRoot()
	}
	return defaultInstallAppRoot()
}

// DataRoot — dominios.db, var/findings, var/logs.
func DataRoot() (string, error) {
	if v := strings.TrimSpace(os.Getenv("GOSCAN_DATA_DIR")); v != "" {
		return filepath.Abs(v)
	}
	if Mode() == ModeDev {
		return findDevRepoRoot()
	}
	if u, ok := validProdSettings(); ok {
		return u.DataDir, nil
	}
	return defaultInstallDataRoot()
}

// ScanInputDir — listas .txt / ficheiros de domínios para o scan.
func ScanInputDir(dataRoot string) (string, error) {
	if Mode() == ModeDev {
		return filepath.Join(dataRoot, "files"), nil
	}
	if u, ok := validProdSettings(); ok && u.ScanDir != "" {
		return u.ScanDir, nil
	}
	return filepath.Join(dataRoot, "files"), nil
}

// IsDevRepoPath reports whether path is a dev checkout or inside one.
func IsDevRepoPath(dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	if isDevRepoRoot(abs) {
		return true
	}
	d := abs
	for {
		if isDevRepoRoot(d) {
			return true
		}
		parent := filepath.Dir(d)
		if parent == d {
			return false
		}
		d = parent
	}
}

func validProdSettings() (settings.User, bool) {
	u, err := settings.Load()
	if err != nil || strings.TrimSpace(u.DataDir) == "" {
		return settings.User{}, false
	}
	absData, err := filepath.Abs(u.DataDir)
	if err != nil || IsDevRepoPath(absData) {
		return settings.User{}, false
	}
	out := settings.User{DataDir: absData}
	if strings.TrimSpace(u.ScanDir) != "" {
		absScan, err := filepath.Abs(u.ScanDir)
		if err == nil && !IsDevRepoPath(absScan) {
			out.ScanDir = absScan
		}
	}
	return out, true
}

// DefaultProdDataRoot is the canonical prod data directory.
func DefaultProdDataRoot() (string, error) {
	return defaultInstallDataRoot()
}

// ProdNeedsSetup is true when prod has invalid or empty data paths.
func ProdNeedsSetup() bool {
	if Mode() != ModeProd {
		return false
	}
	root, err := DataRoot()
	if err != nil || IsDevRepoPath(root) {
		return true
	}
	return !hasProdData(root)
}

func hasProdData(root string) bool {
	if st, err := os.Stat(DefaultDBPath(root)); err == nil && st.Size() > 4096 {
		return true
	}
	if ents, err := os.ReadDir(FindingsByDomain(root)); err == nil && len(ents) > 0 {
		return true
	}
	return false
}

func findDevRepoRoot() (string, error) {
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
			if isDevRepoRoot(dir) {
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

func isDevRepoRoot(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "scripts", "registry.yaml")); err != nil {
		return false
	}
	return true
}

func isInstallAppRoot(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, ".goscan-install")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "scripts", "registry.yaml")); err != nil {
		return false
	}
	return true
}

func defaultInstallAppRoot() (string, error) {
	base, err := userDataHome()
	if err != nil {
		return "", err
	}
	app := filepath.Join(base, "goscan", "app")
	if isInstallAppRoot(app) {
		return app, nil
	}
	// fallback mesmo se ainda não instalado (erro claro nos callers)
	return app, nil
}

func defaultInstallDataRoot() (string, error) {
	base, err := userDataHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "goscan", "data"), nil
}

func userDataHome() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Abs(xdg)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}

func DefaultDBPath(dataRoot string) string {
	return filepath.Join(dataRoot, "dominios.db")
}

func FindingsRoot(dataRoot string) string {
	return filepath.Join(dataRoot, "var", "findings")
}

func FindingsByDomain(dataRoot string) string {
	return filepath.Join(FindingsRoot(dataRoot), "by-domain")
}

func ScriptsDir(appRoot string) string {
	return filepath.Join(appRoot, "scripts")
}

func ArchiveDir(dataRoot string) string {
	return filepath.Join(dataRoot, "var", "archive")
}

func DevLogsDir(dataRoot string) string {
	return filepath.Join(dataRoot, "var", "dev", "logs")
}

func ConfigPath(dataRoot string) string {
	return filepath.Join(dataRoot, "config.yml")
}

func BatchLogsRoot(dataRoot string) string {
	return filepath.Join(dataRoot, "var", "logs", "batch")
}

func InstallVersion(appRoot string) string {
	b, err := os.ReadFile(filepath.Join(appRoot, "VERSION"))
	if err != nil {
		b, err = os.ReadFile(filepath.Join(appRoot, "assets", "VERSION"))
		if err != nil {
			return ""
		}
	}
	return strings.TrimSpace(string(b))
}

// NewBatchRunID returns a UTC timestamp id safe for directory names.
func NewBatchRunID() string {
	return time.Now().UTC().Format("20060102_150405")
}

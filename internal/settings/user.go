package settings

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// User stores workspace paths chosen in the UI.
type User struct {
	DataDir string `yaml:"data_dir"`
	ScanDir string `yaml:"scan_dir"`
}

func configPath() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "goscan")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.yaml"), nil
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

// DefaultProdUser returns canonical XDG prod paths.
func DefaultProdUser() (User, error) {
	base, err := userDataHome()
	if err != nil {
		return User{}, err
	}
	data := filepath.Join(base, "goscan", "data")
	return User{
		DataDir: data,
		ScanDir: filepath.Join(data, "files"),
	}, nil
}

// Load reads ~/.config/goscan/settings.yaml (missing file → empty settings).
func Load() (User, error) {
	path, err := configPath()
	if err != nil {
		return User{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return User{}, nil
		}
		return User{}, err
	}
	var u User
	if err := yaml.Unmarshal(data, &u); err != nil {
		return User{}, err
	}
	u.DataDir = strings.TrimSpace(u.DataDir)
	u.ScanDir = strings.TrimSpace(u.ScanDir)
	return u, nil
}

// Save writes settings.yaml.
func Save(u User) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	u.DataDir = strings.TrimSpace(u.DataDir)
	u.ScanDir = strings.TrimSpace(u.ScanDir)
	body, err := yaml.Marshal(u)
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0644)
}

// SaveProdDefaults writes prod XDG paths if missing or force is true.
func SaveProdDefaults(force bool) (User, error) {
	if !force {
		u, err := Load()
		if err != nil {
			return User{}, err
		}
		if u.DataDir != "" {
			return u, nil
		}
	}
	u, err := DefaultProdUser()
	if err != nil {
		return User{}, err
	}
	if err := os.MkdirAll(u.DataDir, 0755); err != nil {
		return User{}, err
	}
	if err := os.MkdirAll(u.ScanDir, 0755); err != nil {
		return User{}, err
	}
	return u, Save(u)
}

// Configured reports whether settings.yaml has a data_dir.
func Configured() bool {
	u, err := Load()
	return err == nil && u.DataDir != ""
}

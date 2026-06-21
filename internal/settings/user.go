package settings

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// User stores workspace paths and UI preferences chosen in the app.
type User struct {
	DataDir        string `yaml:"data_dir"`
	ScanDir        string `yaml:"scan_dir"`
	PythonPath     string `yaml:"python_path,omitempty"`
	NotifyEnvFound *bool  `yaml:"notify_env_found,omitempty"`
	NotifyScriptOk *bool  `yaml:"notify_script_ok,omitempty"`
	SoundEnvFound  *bool  `yaml:"sound_env_found,omitempty"`
	SoundScriptOk  *bool  `yaml:"sound_script_ok,omitempty"`
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
	u.PythonPath = strings.TrimSpace(u.PythonPath)
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
	u.PythonPath = strings.TrimSpace(u.PythonPath)
	body, err := yaml.Marshal(u)
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0644)
}

// NotifyEnvFoundOrDefault returns whether to notify when a new .env is found.
func (u User) NotifyEnvFoundOrDefault() bool {
	if u.NotifyEnvFound != nil {
		return *u.NotifyEnvFound
	}
	return true
}

// NotifyScriptOkOrDefault returns whether to notify when a checker succeeds.
func (u User) NotifyScriptOkOrDefault() bool {
	if u.NotifyScriptOk != nil {
		return *u.NotifyScriptOk
	}
	return true
}

// SoundEnvFoundOrDefault returns whether to play a sound when a new .env is found.
func (u User) SoundEnvFoundOrDefault() bool {
	if u.SoundEnvFound != nil {
		return *u.SoundEnvFound
	}
	return false
}

// SoundScriptOkOrDefault returns whether to play a sound when a checker succeeds.
func (u User) SoundScriptOkOrDefault() bool {
	if u.SoundScriptOk != nil {
		return *u.SoundScriptOk
	}
	return true
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

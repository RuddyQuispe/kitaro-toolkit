package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds user preferences persisted across sessions.
type Config struct {
	Theme string `json:"theme"` // "dark" or "light"
}

func defaultConfig() Config {
	return Config{Theme: "dark"}
}

// AppDirName is the per-user directory holding all persisted app state.
const AppDirName = "kitaro-toolkit"

// legacyAppDirName is the directory used before the rename to Kitaro Toolkit.
const legacyAppDirName = "kitaro-rq"

// AppDir returns the app's directory inside the user config dir, moving the
// legacy directory there first so existing config and history survive.
func AppDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return appDirIn(base), nil
}

func appDirIn(base string) string {
	dir := filepath.Join(base, AppDirName)
	legacy := filepath.Join(base, legacyAppDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Best effort: on failure the app simply starts with fresh state.
		_ = os.Rename(legacy, dir)
	}
	return dir
}

func path() (string, error) {
	dir, err := AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config file, creating it with defaults if it doesn't exist.
func Load() (Config, error) {
	p, err := path()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		cfg := defaultConfig()
		return cfg, Save(cfg)
	}
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save writes the config file, creating its parent directory if needed.
func Save(cfg Config) error {
	p, err := path()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p, data, 0o644)
}

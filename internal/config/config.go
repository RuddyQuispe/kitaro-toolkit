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

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kitaro-rq", "config.json"), nil
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

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Editor font size bounds (px), shared by every input/output/diff editor.
// The frontend (components/fontZoom.ts) uses the same range.
const (
	DefaultFontSize = 14
	MinFontSize     = 10
	MaxFontSize     = 32
)

// Config holds user preferences persisted across sessions.
type Config struct {
	Theme    string `json:"theme"`    // "dark" or "light"
	FontSize int    `json:"fontSize"` // editor font size in px
}

func defaultConfig() Config {
	return Config{Theme: "dark", FontSize: DefaultFontSize}
}

// ClampFontSize bounds size to [MinFontSize, MaxFontSize].
func ClampFontSize(size int) int {
	return max(MinFontSize, min(MaxFontSize, size))
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

	// Start from defaults so fields missing from older config files (e.g.
	// fontSize) keep their default instead of the zero value.
	cfg := defaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	cfg.FontSize = ClampFontSize(cfg.FontSize)
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

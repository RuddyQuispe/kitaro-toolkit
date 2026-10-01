package config

import (
	"os"
	"path/filepath"
	"testing"
)

// isolateConfigDir points os.UserConfigDir at a temp dir so tests never touch
// the real user config.
func isolateConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	return dir
}

func TestLoad_DefaultsWhenMissing(t *testing.T) {
	isolateConfigDir(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Theme != "dark" {
		t.Errorf("expected default theme 'dark', got %q", cfg.Theme)
	}
	if cfg.FontSize != DefaultFontSize {
		t.Errorf("expected default font size %d, got %d", DefaultFontSize, cfg.FontSize)
	}
	if DefaultFontSize != 14 {
		t.Errorf("expected DefaultFontSize 14, got %d", DefaultFontSize)
	}
}

func TestSaveAndLoad_RoundTripsFontSize(t *testing.T) {
	isolateConfigDir(t)

	if err := Save(Config{Theme: "light", FontSize: 20}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Theme != "light" || cfg.FontSize != 20 {
		t.Fatalf("expected {light 20}, got %+v", cfg)
	}
}

func TestLoad_LegacyFileWithoutFontSizeGetsDefault(t *testing.T) {
	isolateConfigDir(t)

	p, err := path()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(`{"theme":"light"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Theme != "light" || cfg.FontSize != DefaultFontSize {
		t.Fatalf("expected {light %d}, got %+v", DefaultFontSize, cfg)
	}
}

func TestClampFontSize(t *testing.T) {
	tests := []struct{ in, want int }{
		{14, 14},
		{MinFontSize - 3, MinFontSize},
		{MaxFontSize + 3, MaxFontSize},
		{MinFontSize, MinFontSize},
		{MaxFontSize, MaxFontSize},
	}
	for _, tt := range tests {
		if got := ClampFontSize(tt.in); got != tt.want {
			t.Errorf("ClampFontSize(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
	if MinFontSize != 10 || MaxFontSize != 32 {
		t.Errorf("expected font size range 10..32, got %d..%d", MinFontSize, MaxFontSize)
	}
}

func TestAppDirIn_MigratesLegacyDir(t *testing.T) {
	base := t.TempDir()
	legacy := filepath.Join(base, legacyAppDirName)
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "config.json"), []byte(`{"theme":"light"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got := appDirIn(base)

	if want := filepath.Join(base, AppDirName); got != want {
		t.Fatalf("appDirIn = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(got, "config.json")); err != nil {
		t.Fatalf("legacy config not migrated: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy dir still exists: %v", err)
	}
}

func TestAppDirIn_KeepsExistingNewDir(t *testing.T) {
	base := t.TempDir()
	for _, name := range []string{legacyAppDirName, AppDirName} {
		if err := os.MkdirAll(filepath.Join(base, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	appDirIn(base)

	if _, err := os.Stat(filepath.Join(base, legacyAppDirName)); err != nil {
		t.Fatalf("legacy dir should be left alone when new dir exists: %v", err)
	}
}

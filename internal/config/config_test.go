package config

import (
	"os"
	"path/filepath"
	"testing"
)

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

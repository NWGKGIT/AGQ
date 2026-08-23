package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReturnsDefaultsWhenMissing(t *testing.T) {
	setTestHome(t)
	t.Setenv("AGQ_PORT", "")

	cfg := Load()
	if cfg.Port != DefaultPort {
		t.Fatalf("port = %d, want %d", cfg.Port, DefaultPort)
	}
	if cfg.MaskEmails {
		t.Fatal("mask_emails should default to false")
	}
}

func TestDefaultHonorsAgqPort(t *testing.T) {
	t.Setenv("AGQ_PORT", "9999")

	if got := Default().Port; got != 9999 {
		t.Fatalf("port = %d, want 9999", got)
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	setTestHome(t)
	t.Setenv("AGQ_PORT", "")

	want := Config{ExposeAPI: true, Port: 8123, MaskEmails: true}
	if err := Save(want); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if got := Load(); got != want {
		t.Fatalf("Load = %+v, want %+v", got, want)
	}
}

func TestSaveRejectsInvalidPort(t *testing.T) {
	setTestHome(t)

	if err := Save(Config{Port: 0}); err == nil {
		t.Fatal("Save accepted port 0")
	}
	if err := Save(Config{Port: 70000}); err == nil {
		t.Fatal("Save accepted port 70000")
	}
}

func TestLoadRecoversFromCorruptFile(t *testing.T) {
	home := setTestHome(t)
	t.Setenv("AGQ_PORT", "")

	dir := filepath.Join(home, ".agq")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "desktop.json"), []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := Load(); got.Port != DefaultPort {
		t.Fatalf("port = %d, want default %d", got.Port, DefaultPort)
	}
}

func setTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	// os.UserHomeDir uses USERPROFILE on Windows and HOME elsewhere.
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

// Package config persists desktop app settings alongside the daemon's other
// runtime data under ~/.agq.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
)

// DefaultPort matches the daemon's AGQ_PORT default.
const DefaultPort = 7432

// Config holds the user-tunable desktop settings.
type Config struct {
	// Port is the daemon API port to connect to.
	Port int `json:"port"`
	// MaskEmails hides local parts of account emails in the UI.
	MaskEmails bool `json:"mask_emails"`
}

// Default returns the configuration used before the user saves anything.
// AGQ_PORT is honored so a daemon on a custom port works out of the box.
func Default() Config {
	port := DefaultPort
	if raw := os.Getenv("AGQ_PORT"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 65535 {
			port = n
		}
	}
	return Config{Port: port}
}

// Path returns the settings file location, creating the parent directory.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	dir := filepath.Join(home, ".agq")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", fmt.Errorf("secure %s: %w", dir, err)
	}
	return filepath.Join(dir, "desktop.json"), nil
}

// Load reads the saved settings, falling back to defaults when the file does
// not exist yet or cannot be parsed.
func Load() Config {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	var saved Config
	if err := json.Unmarshal(data, &saved); err != nil {
		return cfg
	}
	if saved.Port <= 0 || saved.Port > 65535 {
		saved.Port = cfg.Port
	}
	return saved
}

// Save writes the settings atomically.
func Save(cfg Config) error {
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port %d", cfg.Port)
	}
	path, err := Path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

// Exists reports whether a settings file has been saved before.
func Exists() bool {
	path, err := Path()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

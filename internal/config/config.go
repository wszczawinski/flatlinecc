// Package config loads and saves flatlinecc's settings file.
package config

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is the settings file.
type Config struct {
	DataDir string `toml:"data_dir"`
	// Sync on every connect.
	AutoSyncRides  bool `toml:"auto_sync_rides"`
	AutoSyncRoutes bool `toml:"auto_sync_routes"`
}

// Default returns the settings used when there is no file yet.
func Default() Config {
	return Config{DataDir: "./data"}
}

// Path returns ~/.config/flatlinecc/config.toml.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "flatlinecc", "config.toml"), nil
}

// Load reads the config file, returning defaults if it doesn't exist yet.
func Load() (Config, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, err
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return cfg, err
	}
	return cfg, nil
}

// Save writes the config file. It writes a temp file and renames it, so a
// failed write never leaves a truncated config behind.
func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "config-*.toml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultConfigDir returns the default ~/.rudder directory path.
func DefaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".rudder"
	}
	return filepath.Join(home, ".rudder")
}

// EnsureConfigDir creates the config directory if it does not exist.
func EnsureConfigDir(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	return nil
}

// LoadConfig reads config.yaml from dir. Returns an empty config if the file
// does not exist (first-run case).
func LoadConfig(dir string) (*RudderConfig, error) {
	path := filepath.Join(dir, "config.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &RudderConfig{Version: "1"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg RudderConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// SaveConfig writes cfg to config.yaml atomically.
func SaveConfig(dir string, cfg *RudderConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return writeAtomic(filepath.Join(dir, "config.yaml"), data)
}

// LoadState reads state.yaml from dir. Returns nil, nil if the file does not exist.
func LoadState(dir string) (*State, error) {
	path := filepath.Join(dir, "state.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}
	var state State
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return &state, nil
}

// SaveState writes state to state.yaml atomically using a rename-on-write pattern.
// This is safe under concurrent rudder use invocations.
func SaveState(dir string, state *State) error {
	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return writeAtomic(filepath.Join(dir, "state.yaml"), data)
}

// ExpandPath expands a leading ~/ and environment variables in a path.
func ExpandPath(p string) string {
	if len(p) >= 2 && p[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil {
			p = filepath.Join(home, p[2:])
		}
	}
	return os.ExpandEnv(p)
}

// writeAtomic writes data to path atomically using a unique temp file in the
// same directory followed by os.Rename. Using a unique temp name (via os.CreateTemp)
// prevents concurrent writers from clobbering each other's temp file.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".rudder-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmp := f.Name()

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Chmod(0600); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename to final: %w", err)
	}
	return nil
}

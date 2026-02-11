package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// File represents the rr config file.
type File struct {
	Customer       string   `yaml:"customer,omitempty"`
	DefaultTLDs    []string `yaml:"default_tlds,omitempty"`
	AutoRenew      *bool    `yaml:"auto_renew,omitempty"`
	KeyringBackend string   `yaml:"keyring_backend,omitempty"`
}

// ConfigExists returns true if the config file exists.
func ConfigExists() bool {
	p, err := ConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// ReadConfig reads and parses the config file.
// Returns empty config if file doesn't exist.
func ReadConfig() (*File, error) {
	p, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{}, nil // Return empty config for new users
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg File
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// WriteConfig writes the config file atomically.
func WriteConfig(cfg *File) error {
	dir, err := EnsureDir()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	p, err := ConfigPath()
	if err != nil {
		return err
	}

	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	_ = dir // used by EnsureDir above

	if err := os.Rename(tmp, p); err != nil {
		return fmt.Errorf("rename config: %w", err)
	}

	return nil
}

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const defaultCloudHost = "compasscloud.io"

type Config struct {
	Version      int                         `yaml:"version,omitempty"`
	DefaultStore string                      `yaml:"default_store,omitempty"` // "local" or hostname
	LocalEnabled bool                        `yaml:"local_enabled,omitempty"`
	Stores       map[string]CloudStoreConfig `yaml:"stores,omitempty"`   // hostname -> config
	Projects     map[string]string           `yaml:"projects,omitempty"` // projectKey -> storeName

	// Legacy fields for migration detection
	Mode           string       `yaml:"mode,omitempty"`
	Cloud          *CloudConfig `yaml:"cloud,omitempty"`
	DefaultProject string       `yaml:"default_project,omitempty"`
}

type CloudConfig struct {
	APIKey string `yaml:"api_key"`
}

type CloudStoreConfig struct {
	APIKey   string `yaml:"api_key"`
	Path     string `yaml:"path,omitempty"`     // defaults to "/api/v1"
	Protocol string `yaml:"protocol,omitempty"` // defaults to "https"
}

// URL assembles the full API base URL for a cloud store.
func (c CloudStoreConfig) URL(hostname string) string {
	proto := c.Protocol
	if proto == "" {
		proto = "https"
	}
	path := c.Path
	if path == "" {
		path = "/api/v1"
	}
	return proto + "://" + hostname + path
}

func Load(dataDir string) (*Config, error) {
	path := filepath.Join(dataDir, "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Version < 2 && needsMigration(&cfg) {
		migrated := migrateV1(&cfg)
		return migrated, nil
	}

	return &cfg, nil
}

func Save(dataDir string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	path := filepath.Join(dataDir, "config.yaml")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// needsMigration returns true if the config has v1 fields.
func needsMigration(cfg *Config) bool {
	return cfg.Mode != "" || cfg.Cloud != nil || cfg.DefaultProject != ""
}

func migrateV1(old *Config) *Config {
	cfg := &Config{
		Version:  2,
		Stores:   map[string]CloudStoreConfig{},
		Projects: map[string]string{},
	}
	if old.Cloud != nil && old.Cloud.APIKey != "" {
		cfg.Stores[defaultCloudHost] = CloudStoreConfig{APIKey: old.Cloud.APIKey}
		cfg.DefaultStore = defaultCloudHost
	}
	if old.Mode == "local" {
		cfg.LocalEnabled = true
		cfg.DefaultStore = "local"
	}
	return cfg
}

// IsEmpty returns true when no stores are configured.
func (c *Config) IsEmpty() bool {
	return !c.LocalEnabled && len(c.Stores) == 0
}

// StoreNames returns all configured store names ("local" + hostnames).
func (c *Config) StoreNames() []string {
	var names []string
	if c.LocalEnabled {
		names = append(names, "local")
	}
	for h := range c.Stores {
		names = append(names, h)
	}
	return names
}

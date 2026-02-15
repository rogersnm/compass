package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("version: 2\ndefault_store: local\nlocal_enabled: true\n"), 0644)

	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, 2, cfg.Version)
	assert.Equal(t, "local", cfg.DefaultStore)
	assert.True(t, cfg.LocalEnabled)
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load(t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, 0, cfg.Version)
}

func TestLoad_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("{{bad yaml"), 0644)

	_, err := Load(dir)
	assert.Error(t, err)
}

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Version:      2,
		DefaultStore: "local",
		LocalEnabled: true,
		Projects:     map[string]string{"AUTH": "local"},
	}

	require.NoError(t, Save(dir, cfg))

	loaded, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, 2, loaded.Version)
	assert.Equal(t, "local", loaded.DefaultStore)
	assert.True(t, loaded.LocalEnabled)
	assert.Equal(t, "local", loaded.Projects["AUTH"])
}

func TestSave_CreatesFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir")
	cfg := &Config{Version: 2}

	require.NoError(t, Save(dir, cfg))
	_, err := os.Stat(filepath.Join(dir, "config.yaml"))
	assert.NoError(t, err)
}

func TestMigrateV1_LocalMode(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("mode: local\ndefault_project: AUTH\n"), 0644)

	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, 2, cfg.Version)
	assert.True(t, cfg.LocalEnabled)
	assert.Equal(t, "local", cfg.DefaultStore)
}

func TestMigrateV1_CloudMode(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("cloud:\n  api_key: cpk_test123\n"), 0644)

	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, 2, cfg.Version)
	assert.False(t, cfg.LocalEnabled)
	assert.Equal(t, "compasscloud.io", cfg.DefaultStore)
	assert.Equal(t, "cpk_test123", cfg.Stores["compasscloud.io"].APIKey)
}

func TestMigrateV1_Empty(t *testing.T) {
	// Empty config (no v1 fields) should NOT migrate
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(""), 0644)

	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, 0, cfg.Version) // no migration needed
}

func TestCloudStoreConfig_URL(t *testing.T) {
	tests := []struct {
		name string
		cfg  CloudStoreConfig
		want string
	}{
		{"defaults", CloudStoreConfig{Hostname: "compasscloud.io", APIKey: "k"}, "https://compasscloud.io/api/v1"},
		{"custom path", CloudStoreConfig{Hostname: "example.com", APIKey: "k", Path: "/compass/api/v1"}, "https://example.com/compass/api/v1"},
		{"custom protocol", CloudStoreConfig{Hostname: "localhost:8080", APIKey: "k", Protocol: "http"}, "http://localhost:8080/api/v1"},
		{"all custom", CloudStoreConfig{Hostname: "self.host", APIKey: "k", Path: "/v2", Protocol: "http"}, "http://self.host/v2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.cfg.URL())
		})
	}
}

func TestConfig_IsEmpty(t *testing.T) {
	assert.True(t, (&Config{}).IsEmpty())
	assert.False(t, (&Config{LocalEnabled: true}).IsEmpty())
	assert.False(t, (&Config{Stores: map[string]CloudStoreConfig{"h": {}}}).IsEmpty())
}

func TestConfig_StoreNames(t *testing.T) {
	cfg := &Config{
		LocalEnabled: true,
		Stores:       map[string]CloudStoreConfig{"example.com": {}},
	}
	names := cfg.StoreNames()
	assert.Contains(t, names, "local")
	assert.Contains(t, names, "example.com")
}

func TestSave_V2WithStores(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Version:      2,
		DefaultStore: "compasscloud.io",
		LocalEnabled: true,
		Stores: map[string]CloudStoreConfig{
			"compasscloud.io": {Hostname: "compasscloud.io", APIKey: "cpk_xxx"},
			"self.example.com": {
				Hostname: "self.example.com",
				APIKey:   "cpk_yyy",
				Path:     "/compass/api/v1",
				Protocol: "http",
			},
		},
		Projects: map[string]string{
			"AUTH": "local",
			"API":  "compasscloud.io",
		},
	}

	require.NoError(t, Save(dir, cfg))

	loaded, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, 2, loaded.Version)
	assert.Equal(t, "compasscloud.io", loaded.DefaultStore)
	assert.True(t, loaded.LocalEnabled)
	assert.Len(t, loaded.Stores, 2)
	assert.Equal(t, "cpk_xxx", loaded.Stores["compasscloud.io"].APIKey)
	assert.Equal(t, "http", loaded.Stores["self.example.com"].Protocol)
	assert.Equal(t, "local", loaded.Projects["AUTH"])
	assert.Equal(t, "compasscloud.io", loaded.Projects["API"])
}

func TestLoad_BackfillHostname(t *testing.T) {
	dir := t.TempDir()
	// Old config without hostname field
	yaml := "version: 2\nstores:\n  compasscloud.io:\n    api_key: cpk_xxx\n"
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0644)

	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "compasscloud.io", cfg.Stores["compasscloud.io"].Hostname)
}

func TestLoad_BackfillHostname_Mixed(t *testing.T) {
	dir := t.TempDir()
	// Mix of old (no hostname) and new (has hostname) entries
	yaml := `version: 2
stores:
  compasscloud.io:
    api_key: cpk_old
  work:
    hostname: compasscloud.io
    api_key: cpk_work
`
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0644)

	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "compasscloud.io", cfg.Stores["compasscloud.io"].Hostname)
	assert.Equal(t, "compasscloud.io", cfg.Stores["work"].Hostname)
}

func TestSave_RoundTripNamedStores(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Version:      2,
		DefaultStore: "personal",
		Stores: map[string]CloudStoreConfig{
			"personal": {Hostname: "compasscloud.io", APIKey: "cpk_personal"},
			"work":     {Hostname: "compasscloud.io", APIKey: "cpk_work"},
		},
		Projects: map[string]string{
			"AUTH": "personal",
			"API":  "work",
		},
	}

	require.NoError(t, Save(dir, cfg))

	loaded, err := Load(dir)
	require.NoError(t, err)
	assert.Len(t, loaded.Stores, 2)
	assert.Equal(t, "compasscloud.io", loaded.Stores["personal"].Hostname)
	assert.Equal(t, "cpk_personal", loaded.Stores["personal"].APIKey)
	assert.Equal(t, "compasscloud.io", loaded.Stores["work"].Hostname)
	assert.Equal(t, "cpk_work", loaded.Stores["work"].APIKey)
	assert.Equal(t, "personal", loaded.Projects["AUTH"])
	assert.Equal(t, "work", loaded.Projects["API"])
}

func TestValidateStoreName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid hostname", "compasscloud.io", false},
		{"valid custom name", "work", false},
		{"valid with numbers", "org42", false},
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"leading space", " work", true},
		{"trailing space", "work ", true},
		{"reserved local", "local", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStoreName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

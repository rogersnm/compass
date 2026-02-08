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
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("default_project: PROJ-ABCDE\n"), 0644)

	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "PROJ-ABCDE", cfg.DefaultProject)
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load(t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, "", cfg.DefaultProject)
}

func TestLoad_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("{{bad yaml"), 0644)

	_, err := Load(dir)
	assert.Error(t, err)
}

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DefaultProject: "PROJ-ABCDE"}

	require.NoError(t, Save(dir, cfg))

	loaded, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, cfg.DefaultProject, loaded.DefaultProject)
}

func TestSave_CreatesFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir")
	cfg := &Config{DefaultProject: "PROJ-ABCDE"}

	require.NoError(t, Save(dir, cfg))
	_, err := os.Stat(filepath.Join(dir, "config.yaml"))
	assert.NoError(t, err)
}

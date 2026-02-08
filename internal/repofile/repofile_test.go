package repofile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteRead_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Write(dir, "AUTH"))

	got, err := Read(dir)
	require.NoError(t, err)
	assert.Equal(t, "AUTH", got)
}

func TestRead_Missing(t *testing.T) {
	dir := t.TempDir()
	got, err := Read(dir)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestRead_TrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, FileName), []byte("  AUTH \n\n"), 0644)

	got, err := Read(dir)
	require.NoError(t, err)
	assert.Equal(t, "AUTH", got)
}

func TestFind_CurrentDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Write(dir, "PROJ"))

	id, foundDir, err := Find(dir)
	require.NoError(t, err)
	assert.Equal(t, "PROJ", id)
	assert.Equal(t, dir, foundDir)
}

func TestFind_ParentDir(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "sub", "deep")
	require.NoError(t, os.MkdirAll(child, 0755))
	require.NoError(t, Write(parent, "PROJ"))

	id, foundDir, err := Find(child)
	require.NoError(t, err)
	assert.Equal(t, "PROJ", id)
	assert.Equal(t, parent, foundDir)
}

func TestFind_NotFound(t *testing.T) {
	dir := t.TempDir()
	id, foundDir, err := Find(dir)
	require.NoError(t, err)
	assert.Empty(t, id)
	assert.Empty(t, foundDir)
}

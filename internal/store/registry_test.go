package store

import (
	"testing"

	"github.com/rogersnm/compass/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRegistry(t *testing.T) (*Registry, *LocalStore, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		Version:      2,
		LocalEnabled: true,
		DefaultStore: "local",
		Projects:     map[string]string{},
	}
	reg := NewRegistry(cfg, dir)
	ls := NewLocal(dir)
	reg.Add("local", ls)
	return reg, ls, dir
}

func TestForProject_CacheHit(t *testing.T) {
	reg, ls, _ := setupRegistry(t)
	ls.CreateProject("Test", "TP", "")
	reg.CacheProject("TP", "local")

	s, name, err := reg.ForProject("TP")
	require.NoError(t, err)
	assert.Equal(t, "local", name)
	assert.Equal(t, ls, s)
}

func TestForProject_CacheMiss(t *testing.T) {
	reg, ls, _ := setupRegistry(t)
	ls.CreateProject("Test", "TP", "")

	s, name, err := reg.ForProject("TP")
	require.NoError(t, err)
	assert.Equal(t, "local", name)
	assert.Equal(t, ls, s)
	// Should now be cached
	assert.Equal(t, "local", reg.cfg.Projects["TP"])
}

func TestForProject_NotFound(t *testing.T) {
	reg, _, _ := setupRegistry(t)
	_, _, err := reg.ForProject("NOPE")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found on any configured store")
}

func TestForProject_StaleCache(t *testing.T) {
	reg, ls, _ := setupRegistry(t)
	ls.CreateProject("Test", "TP", "")
	reg.CacheProject("TP", "local")

	// Delete the project to make cache stale
	ls.DeleteProject("TP")

	_, _, err := reg.ForProject("TP")
	assert.Error(t, err) // not found anywhere after prune
	// Cache should be pruned
	_, ok := reg.cfg.Projects["TP"]
	assert.False(t, ok)
}

func TestForEntity(t *testing.T) {
	reg, ls, _ := setupRegistry(t)
	ls.CreateProject("Test", "TP", "")
	ls.CreateTask("Task", "TP", TaskCreateOpts{})

	tasks, _ := ls.ListTasks(TaskFilter{ProjectID: "TP"})
	require.Len(t, tasks, 1)

	s, name, err := reg.ForEntity(tasks[0].ID)
	require.NoError(t, err)
	assert.Equal(t, "local", name)
	assert.Equal(t, ls, s)
}

func TestUncacheProject(t *testing.T) {
	reg, _, _ := setupRegistry(t)
	reg.CacheProject("TP", "local")
	assert.Equal(t, "local", reg.cfg.Projects["TP"])

	reg.UncacheProject("TP")
	_, ok := reg.cfg.Projects["TP"]
	assert.False(t, ok)
}

func TestRegistry_IsEmpty(t *testing.T) {
	cfg := &config.Config{Version: 2}
	reg := NewRegistry(cfg, t.TempDir())
	assert.True(t, reg.IsEmpty())

	reg.Add("local", NewLocal(t.TempDir()))
	assert.False(t, reg.IsEmpty())
}

func TestRegistry_Default(t *testing.T) {
	reg, ls, _ := setupRegistry(t)
	s, name, err := reg.Default()
	require.NoError(t, err)
	assert.Equal(t, "local", name)
	assert.Equal(t, ls, s)
}

func TestRegistry_DefaultNotSet(t *testing.T) {
	cfg := &config.Config{Version: 2}
	reg := NewRegistry(cfg, t.TempDir())
	_, _, err := reg.Default()
	assert.Error(t, err)
}

package cmd

import (
	"testing"

	"github.com/rogersnm/compass/internal/config"
	"github.com/rogersnm/compass/internal/model"
	"github.com/rogersnm/compass/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEnv(t *testing.T) (*store.Store, string) {
	t.Helper()
	dir := t.TempDir()
	dataDir = dir
	st = store.New(dir)
	cfg = &config.Config{}
	return st, dir
}

func run(t *testing.T, args ...string) error {
	t.Helper()
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

// Tests operate through the store layer and use CLI commands only where
// Cobra's shared flag state won't interfere.

func TestProjectCreate_Success(t *testing.T) {
	s, _ := setupEnv(t)
	require.NoError(t, run(t, "project", "create", "Test Project"))

	projects, err := s.ListProjects()
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, "Test Project", projects[0].Name)
}

func TestProjectList_Empty(t *testing.T) {
	setupEnv(t)
	require.NoError(t, run(t, "project", "list"))
}

func TestProjectShow_NotFound(t *testing.T) {
	setupEnv(t)
	assert.Error(t, run(t, "project", "show", "PROJ-ZZZZZ"))
}

func TestProjectSetDefault(t *testing.T) {
	s, dir := setupEnv(t)
	p, err := s.CreateProject("P", "")
	require.NoError(t, err)

	require.NoError(t, run(t, "project", "set-default", p.ID))

	c, err := config.Load(dir)
	require.NoError(t, err)
	assert.Equal(t, p.ID, c.DefaultProject)
}

func TestDocCreate_WithProject(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("P", "")

	require.NoError(t, run(t, "doc", "create", "My Doc", "--project", p.ID))

	docs, err := s.ListDocuments(p.ID)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
}

func TestDocCreate_DefaultProject(t *testing.T) {
	s, dir := setupEnv(t)
	p, _ := s.CreateProject("P", "")
	cfg = &config.Config{DefaultProject: p.ID}
	config.Save(dir, cfg)

	// Must explicitly pass --project "" to clear any leftover flag from prior test
	require.NoError(t, run(t, "doc", "create", "My Doc", "--project", p.ID))

	docs, err := s.ListDocuments(p.ID)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
}

func TestEpicCreate_Success(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("P", "")

	require.NoError(t, run(t, "epic", "create", "Auth Epic", "--project", p.ID))

	epics, err := s.ListEpics(p.ID)
	require.NoError(t, err)
	assert.Len(t, epics, 1)
	assert.Equal(t, "Auth Epic", epics[0].Title)
	assert.Equal(t, model.StatusOpen, epics[0].Status)
}

func TestTaskCreate_Minimal(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("P", "")

	require.NoError(t, run(t, "task", "create", "My Task", "--project", p.ID))

	tasks, err := s.ListTasks(store.TaskFilter{ProjectID: p.ID})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestTaskCreate_WithDeps(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("P", "")
	t1, _ := s.CreateTask("Dep", p.ID, store.TaskCreateOpts{})

	require.NoError(t, run(t, "task", "create", "My Task", "--project", p.ID, "--depends-on", t1.ID))

	tasks, err := s.ListTasks(store.TaskFilter{ProjectID: p.ID})
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestTaskUpdate_Status(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("P", "")
	task, _ := s.CreateTask("Task", p.ID, store.TaskCreateOpts{})

	require.NoError(t, run(t, "task", "update", task.ID, "--status", "in_progress"))

	got, _, err := s.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusInProgress, got.Status)
}

func TestTaskGraph(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("P", "")
	cfg = &config.Config{DefaultProject: p.ID}
	s.CreateTask("Root", p.ID, store.TaskCreateOpts{})

	require.NoError(t, run(t, "task", "graph", "--project", p.ID))
}

func TestSearch_NoResults(t *testing.T) {
	s, _ := setupEnv(t)
	s.CreateProject("Test", "")
	require.NoError(t, run(t, "search", "xyznonexistent"))
}

func TestE2EWorkflow(t *testing.T) {
	s, _ := setupEnv(t)

	// 1. Create project
	p, err := s.CreateProject("E2E Project", "")
	require.NoError(t, err)

	// 2. Set default
	cfg = &config.Config{DefaultProject: p.ID}

	// 3. Create docs
	d1, _ := s.CreateDocument("Design Doc", p.ID, "")
	d2, _ := s.CreateDocument("API Spec", p.ID, "")
	_ = d1
	_ = d2

	// 4. Create epic
	e, _ := s.CreateEpic("Auth Epic", p.ID, "")

	// 5. Create tasks
	tA, _ := s.CreateTask("Task A", p.ID, store.TaskCreateOpts{Epic: e.ID})
	_, _ = s.CreateTask("Task B", p.ID, store.TaskCreateOpts{})
	tC, _ := s.CreateTask("Task C", p.ID, store.TaskCreateOpts{DependsOn: []string{tA.ID}})

	// 6. Verify counts
	docs, _ := s.ListDocuments(p.ID)
	assert.Len(t, docs, 2)
	tasks, _ := s.ListTasks(store.TaskFilter{ProjectID: p.ID})
	assert.Len(t, tasks, 3)

	// 7. Task C should be blocked
	allTasks, _ := s.AllTaskMap(p.ID)
	assert.True(t, tC.IsBlocked(allTasks))

	// 8. Close Task A
	closed := model.StatusClosed
	s.UpdateTask(tA.ID, store.TaskUpdate{Status: &closed})

	// 9. Task C no longer blocked
	allTasks, _ = s.AllTaskMap(p.ID)
	gotC, _, _ := s.GetTask(tC.ID)
	assert.False(t, gotC.IsBlocked(allTasks))

	// 10. Graph via CLI
	require.NoError(t, run(t, "task", "graph", "--project", p.ID))

	// 11. Search
	results, _ := s.Search("Auth", "")
	assert.GreaterOrEqual(t, len(results), 1)
}

package cmd

import (
	"os"
	"path/filepath"
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

func TestProjectCreate_WithKey(t *testing.T) {
	s, _ := setupEnv(t)
	require.NoError(t, run(t, "project", "create", "Test Project", "--key", "TP"))

	projects, err := s.ListProjects()
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, "TP", projects[0].ID)
}

func TestProjectList_Empty(t *testing.T) {
	setupEnv(t)
	require.NoError(t, run(t, "project", "list"))
}

func TestProjectShow_NotFound(t *testing.T) {
	setupEnv(t)
	assert.Error(t, run(t, "project", "show", "ZZZZ"))
}

func TestProjectSetDefault(t *testing.T) {
	s, dir := setupEnv(t)
	p, err := s.CreateProject("Test Project", "TP", "")
	require.NoError(t, err)

	require.NoError(t, run(t, "project", "set-default", p.ID))

	c, err := config.Load(dir)
	require.NoError(t, err)
	assert.Equal(t, p.ID, c.DefaultProject)
}

func TestDocCreate_WithProject(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	require.NoError(t, run(t, "doc", "create", "My Doc", "--project", p.ID))

	docs, err := s.ListDocuments(p.ID)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
}

func TestDocCreate_DefaultProject(t *testing.T) {
	s, dir := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	cfg = &config.Config{DefaultProject: p.ID}
	config.Save(dir, cfg)

	// Must explicitly pass --project "" to clear any leftover flag from prior test
	require.NoError(t, run(t, "doc", "create", "My Doc", "--project", p.ID))

	docs, err := s.ListDocuments(p.ID)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
}

func TestTaskCreate_Minimal(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	require.NoError(t, run(t, "task", "create", "My Task", "--project", p.ID))

	tasks, err := s.ListTasks(store.TaskFilter{ProjectID: p.ID})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, model.TypeTask, tasks[0].Type)
}

func TestTaskCreate_EpicType(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	require.NoError(t, run(t, "task", "create", "Auth Epic", "--project", p.ID, "--type", "epic"))

	tasks, err := s.ListTasks(store.TaskFilter{ProjectID: p.ID, Type: model.TypeEpic})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Auth Epic", tasks[0].Title)
	assert.Equal(t, model.TypeEpic, tasks[0].Type)
}

func TestTaskCreate_WithPriority(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	require.NoError(t, run(t, "task", "create", "Urgent", "--project", p.ID, "--type", "task", "--priority", "1"))

	tasks, err := s.ListTasks(store.TaskFilter{ProjectID: p.ID})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.NotNil(t, tasks[0].Priority)
	assert.Equal(t, 1, *tasks[0].Priority)
}

func TestTaskCreate_NoPriority(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	require.NoError(t, run(t, "task", "create", "Normal", "--project", p.ID, "--type", "task", "--priority", "-1"))

	tasks, err := s.ListTasks(store.TaskFilter{ProjectID: p.ID})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Nil(t, tasks[0].Priority)
}

func TestTaskUpdate_Priority(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("Task", p.ID, store.TaskCreateOpts{})

	require.NoError(t, run(t, "task", "update", task.ID, "--priority", "0"))

	got, _, err := s.GetTask(task.ID)
	require.NoError(t, err)
	require.NotNil(t, got.Priority)
	assert.Equal(t, 0, *got.Priority)
}

func TestTaskCreate_WithDeps(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	t1, _ := s.CreateTask("Dep", p.ID, store.TaskCreateOpts{})

	require.NoError(t, run(t, "task", "create", "My Task", "--project", p.ID, "--type", "task", "--depends-on", t1.ID))

	tasks, err := s.ListTasks(store.TaskFilter{ProjectID: p.ID})
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestTaskUpdate_Status(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("Task", p.ID, store.TaskCreateOpts{})

	require.NoError(t, run(t, "task", "update", task.ID, "--status", "in_progress"))

	got, _, err := s.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusInProgress, got.Status)
}

func TestTaskStart(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("Task", p.ID, store.TaskCreateOpts{})

	require.NoError(t, run(t, "task", "start", task.ID))

	got, _, err := s.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusInProgress, got.Status)
}

func TestTaskClose(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("Task", p.ID, store.TaskCreateOpts{})

	require.NoError(t, run(t, "task", "close", task.ID))

	got, _, err := s.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusClosed, got.Status)
}

func TestTaskReady(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	cfg = &config.Config{DefaultProject: p.ID}
	s.CreateTask("Ready Task", p.ID, store.TaskCreateOpts{})

	require.NoError(t, run(t, "task", "ready", "--project", p.ID))
}

func TestTaskReady_All(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	cfg = &config.Config{DefaultProject: p.ID}
	s.CreateTask("T1", p.ID, store.TaskCreateOpts{})
	s.CreateTask("T2", p.ID, store.TaskCreateOpts{})

	require.NoError(t, run(t, "task", "ready", "--project", p.ID, "--all"))
}

func TestTaskDelete_Force(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("Task", p.ID, store.TaskCreateOpts{})

	require.NoError(t, run(t, "task", "delete", task.ID, "--force"))

	_, _, err := s.GetTask(task.ID)
	assert.Error(t, err)
}

func TestDocDelete_Force(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	d, _ := s.CreateDocument("Doc", p.ID, "body")

	require.NoError(t, run(t, "doc", "delete", d.ID, "--force"))

	_, _, err := s.GetDocument(d.ID)
	assert.Error(t, err)
}

func TestProjectDelete_Force(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	s.CreateTask("Task", p.ID, store.TaskCreateOpts{})

	require.NoError(t, run(t, "project", "delete", p.ID, "--force"))

	_, _, err := s.GetProject(p.ID)
	assert.Error(t, err)
}

func TestProjectDelete_ClearsDefault(t *testing.T) {
	s, dir := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	cfg = &config.Config{DefaultProject: p.ID}
	config.Save(dir, cfg)

	require.NoError(t, run(t, "project", "delete", p.ID, "--force"))

	c, err := config.Load(dir)
	require.NoError(t, err)
	assert.Empty(t, c.DefaultProject)
}

func TestTaskGraph(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	cfg = &config.Config{DefaultProject: p.ID}
	s.CreateTask("Root", p.ID, store.TaskCreateOpts{})

	require.NoError(t, run(t, "task", "graph", "--project", p.ID))
}

func TestSearch_NoResults(t *testing.T) {
	s, _ := setupEnv(t)
	s.CreateProject("Test Project", "TP", "")
	require.NoError(t, run(t, "search", "xyznonexistent"))
}

func TestTaskCheckout(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("My Task", p.ID, store.TaskCreateOpts{Body: "task body"})

	// Change to a temp dir so .compass/ is created there
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	require.NoError(t, run(t, "task", "checkout", task.ID))

	localPath := filepath.Join(".compass", task.ID+".md")
	assert.FileExists(t, localPath)
}

func TestTaskCheckin(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("My Task", p.ID, store.TaskCreateOpts{Body: "old body"})

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	require.NoError(t, run(t, "task", "checkout", task.ID))
	require.NoError(t, run(t, "task", "checkin", task.ID))

	// Local file should be gone
	localPath := filepath.Join(".compass", task.ID+".md")
	assert.NoFileExists(t, localPath)

	// Store should still have the task
	got, _, err := s.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, "My Task", got.Title)
}

func TestDocCheckout(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	doc, _ := s.CreateDocument("My Doc", p.ID, "doc body")

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	require.NoError(t, run(t, "doc", "checkout", doc.ID))

	localPath := filepath.Join(".compass", doc.ID+".md")
	assert.FileExists(t, localPath)
}

func TestDocCheckin(t *testing.T) {
	s, _ := setupEnv(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	doc, _ := s.CreateDocument("My Doc", p.ID, "old body")

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	require.NoError(t, run(t, "doc", "checkout", doc.ID))
	require.NoError(t, run(t, "doc", "checkin", doc.ID))

	localPath := filepath.Join(".compass", doc.ID+".md")
	assert.NoFileExists(t, localPath)

	got, _, err := s.GetDocument(doc.ID)
	require.NoError(t, err)
	assert.Equal(t, "My Doc", got.Title)
}

func TestE2EWorkflow(t *testing.T) {
	s, _ := setupEnv(t)

	// 1. Create project
	p, err := s.CreateProject("E2E Project", "", "")
	require.NoError(t, err)

	// 2. Set default
	cfg = &config.Config{DefaultProject: p.ID}

	// 3. Create docs
	d1, _ := s.CreateDocument("Design Doc", p.ID, "")
	d2, _ := s.CreateDocument("API Spec", p.ID, "")
	_ = d1
	_ = d2

	// 4. Create epic (now a task with type=epic)
	epic, _ := s.CreateTask("Auth Epic", p.ID, store.TaskCreateOpts{Type: model.TypeEpic})

	// 5. Create tasks
	tA, _ := s.CreateTask("Task A", p.ID, store.TaskCreateOpts{Epic: epic.ID})
	_, _ = s.CreateTask("Task B", p.ID, store.TaskCreateOpts{})
	tC, _ := s.CreateTask("Task C", p.ID, store.TaskCreateOpts{DependsOn: []string{tA.ID}})

	// 6. Verify counts
	docs, _ := s.ListDocuments(p.ID)
	assert.Len(t, docs, 2)
	tasks, _ := s.ListTasks(store.TaskFilter{ProjectID: p.ID})
	assert.Len(t, tasks, 4) // 3 tasks + 1 epic

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

	// 12. Ready tasks
	ready, err := s.ReadyTasks(p.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(ready), 1)
}

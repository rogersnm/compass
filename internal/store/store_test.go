package store

import (
	"testing"

	"github.com/rogersnm/compass/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	return New(t.TempDir())
}

func TestEnsureProjectDirs(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.EnsureProjectDirs("PROJ-ABCDE"))

	for _, sub := range []string{"documents", "epics", "tasks"} {
		assert.DirExists(t, s.ProjectDir("PROJ-ABCDE")+"/"+sub)
	}
}

func TestWriteEntity_ReadEntity_RoundTrip(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.EnsureProjectDirs("PROJ-ABCDE"))

	task := &model.Task{
		ID:      "TASK-ABCDE",
		Title:   "Test Task",
		Project: "PROJ-ABCDE",
		Status:  model.StatusOpen,
	}
	path := s.ProjectDir("PROJ-ABCDE") + "/tasks/TASK-ABCDE.md"
	require.NoError(t, s.WriteEntity(path, task, "some body"))

	got, body, err := ReadEntity[model.Task](path)
	require.NoError(t, err)
	assert.Equal(t, "TASK-ABCDE", got.ID)
	assert.Equal(t, "Test Task", got.Title)
	assert.Equal(t, "some body", body)
}

func TestResolveEntityPath_Task(t *testing.T) {
	s := newTestStore(t)
	p, err := s.CreateProject("Test", "")
	require.NoError(t, err)

	task, err := s.CreateTask("My Task", p.ID, TaskCreateOpts{})
	require.NoError(t, err)

	path, err := s.ResolveEntityPath(task.ID)
	require.NoError(t, err)
	assert.Contains(t, path, task.ID+".md")
}

func TestResolveEntityPath_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.ResolveEntityPath("TASK-ZZZZZ")
	assert.Error(t, err)
}

func TestResolveEntityPath_AcrossProjects(t *testing.T) {
	s := newTestStore(t)
	p1, err := s.CreateProject("P1", "")
	require.NoError(t, err)
	p2, err := s.CreateProject("P2", "")
	require.NoError(t, err)

	_, err = s.CreateTask("Task1", p1.ID, TaskCreateOpts{})
	require.NoError(t, err)
	t2, err := s.CreateTask("Task2", p2.ID, TaskCreateOpts{})
	require.NoError(t, err)

	path, err := s.ResolveEntityPath(t2.ID)
	require.NoError(t, err)
	assert.Contains(t, path, t2.ID+".md")
}

// --- Project tests ---

func TestCreateProject(t *testing.T) {
	s := newTestStore(t)
	p, err := s.CreateProject("My Project", "")
	require.NoError(t, err)
	assert.NotEmpty(t, p.ID)
	assert.Equal(t, "My Project", p.Name)
	assert.NotEmpty(t, p.CreatedBy)
	assert.False(t, p.CreatedAt.IsZero())
	assert.False(t, p.UpdatedAt.IsZero())
}

func TestCreateProject_EmptyName(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CreateProject("", "")
	assert.Error(t, err)
}

func TestGetProject(t *testing.T) {
	s := newTestStore(t)
	p, err := s.CreateProject("Test", "project body")
	require.NoError(t, err)

	got, body, err := s.GetProject(p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, "Test", got.Name)
	assert.Equal(t, "project body", body)
}

func TestGetProject_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, _, err := s.GetProject("PROJ-ZZZZZ")
	assert.Error(t, err)
}

func TestListProjects_Empty(t *testing.T) {
	s := newTestStore(t)
	projects, err := s.ListProjects()
	require.NoError(t, err)
	assert.Empty(t, projects)
}

func TestListProjects_Multiple(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject("P1", "")
	s.CreateProject("P2", "")
	s.CreateProject("P3", "")

	projects, err := s.ListProjects()
	require.NoError(t, err)
	assert.Len(t, projects, 3)
}

// --- Document tests ---

func TestCreateDocument(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")

	d, err := s.CreateDocument("My Doc", p.ID, "")
	require.NoError(t, err)
	assert.NotEmpty(t, d.ID)
	assert.Equal(t, "My Doc", d.Title)
	assert.Equal(t, p.ID, d.Project)
}

func TestCreateDocument_WithBody(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")

	d, err := s.CreateDocument("Doc", p.ID, "# Hello\n\nBody content.")
	require.NoError(t, err)

	_, body, err := s.GetDocument(d.ID)
	require.NoError(t, err)
	assert.Equal(t, "# Hello\n\nBody content.", body)
}

func TestCreateDocument_InvalidProject(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CreateDocument("Doc", "PROJ-ZZZZZ", "")
	assert.Error(t, err)
}

func TestListDocuments_FilterByProject(t *testing.T) {
	s := newTestStore(t)
	p1, _ := s.CreateProject("P1", "")
	p2, _ := s.CreateProject("P2", "")
	s.CreateDocument("D1", p1.ID, "")
	s.CreateDocument("D2", p1.ID, "")
	s.CreateDocument("D3", p2.ID, "")

	docs, err := s.ListDocuments(p1.ID)
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestListDocuments_AllProjects(t *testing.T) {
	s := newTestStore(t)
	p1, _ := s.CreateProject("P1", "")
	p2, _ := s.CreateProject("P2", "")
	s.CreateDocument("D1", p1.ID, "")
	s.CreateDocument("D2", p2.ID, "")

	docs, err := s.ListDocuments("")
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestUpdateDocument(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")
	d, _ := s.CreateDocument("Original", p.ID, "old body")

	newTitle := "Updated"
	newBody := "new body"
	updated, err := s.UpdateDocument(d.ID, &newTitle, &newBody)
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Title)

	_, body, _ := s.GetDocument(d.ID)
	assert.Equal(t, "new body", body)
}

// --- Epic tests ---

func TestCreateEpic(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")

	e, err := s.CreateEpic("My Epic", p.ID, "")
	require.NoError(t, err)
	assert.NotEmpty(t, e.ID)
	assert.Equal(t, model.StatusOpen, e.Status)
}

func TestCreateEpic_WithBody(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")

	e, err := s.CreateEpic("Epic", p.ID, "Epic description")
	require.NoError(t, err)

	_, body, _ := s.GetEpic(e.ID)
	assert.Equal(t, "Epic description", body)
}

func TestListEpics_FilterByProject(t *testing.T) {
	s := newTestStore(t)
	p1, _ := s.CreateProject("P1", "")
	p2, _ := s.CreateProject("P2", "")
	s.CreateEpic("E1", p1.ID, "")
	s.CreateEpic("E2", p2.ID, "")

	epics, err := s.ListEpics(p1.ID)
	require.NoError(t, err)
	assert.Len(t, epics, 1)
}

// --- Task tests ---

func TestCreateTask_Minimal(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")

	task, err := s.CreateTask("Task", p.ID, TaskCreateOpts{})
	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)
	assert.Equal(t, model.StatusOpen, task.Status)
	assert.Empty(t, task.Epic)
	assert.Empty(t, task.DependsOn)
}

func TestCreateTask_WithEpic(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")
	e, _ := s.CreateEpic("Epic", p.ID, "")

	task, err := s.CreateTask("Task", p.ID, TaskCreateOpts{Epic: e.ID})
	require.NoError(t, err)
	assert.Equal(t, e.ID, task.Epic)
}

func TestCreateTask_WithDependencies(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")
	t1, _ := s.CreateTask("T1", p.ID, TaskCreateOpts{})

	t2, err := s.CreateTask("T2", p.ID, TaskCreateOpts{DependsOn: []string{t1.ID}})
	require.NoError(t, err)
	assert.Equal(t, []string{t1.ID}, t2.DependsOn)
}

func TestCreateTask_WithBody(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")

	task, err := s.CreateTask("Task", p.ID, TaskCreateOpts{Body: "task body"})
	require.NoError(t, err)

	_, body, _ := s.GetTask(task.ID)
	assert.Equal(t, "task body", body)
}

func TestCreateTask_InvalidEpic(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")

	_, err := s.CreateTask("Task", p.ID, TaskCreateOpts{Epic: "EPIC-ZZZZZ"})
	assert.Error(t, err)
}

func TestCreateTask_InvalidDependency(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")

	_, err := s.CreateTask("Task", p.ID, TaskCreateOpts{DependsOn: []string{"TASK-ZZZZZ"}})
	assert.Error(t, err)
}

func TestCreateTask_CyclicDep(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")
	t1, _ := s.CreateTask("T1", p.ID, TaskCreateOpts{})
	t2, _ := s.CreateTask("T2", p.ID, TaskCreateOpts{DependsOn: []string{t1.ID}})

	// T1 depends on T2 would create a cycle
	_, err := s.UpdateTask(t1.ID, TaskUpdate{DependsOn: &[]string{t2.ID}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestListTasks_FilterByStatus(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")
	s.CreateTask("T1", p.ID, TaskCreateOpts{})
	t2, _ := s.CreateTask("T2", p.ID, TaskCreateOpts{})
	inProg := model.StatusInProgress
	s.UpdateTask(t2.ID, TaskUpdate{Status: &inProg})

	tasks, err := s.ListTasks(TaskFilter{Status: model.StatusOpen})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestListTasks_FilterByEpic(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")
	e, _ := s.CreateEpic("Epic", p.ID, "")
	s.CreateTask("T1", p.ID, TaskCreateOpts{Epic: e.ID})
	s.CreateTask("T2", p.ID, TaskCreateOpts{})

	tasks, err := s.ListTasks(TaskFilter{EpicID: e.ID})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestUpdateTask_Status(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")
	task, _ := s.CreateTask("Task", p.ID, TaskCreateOpts{})

	status := model.StatusInProgress
	updated, err := s.UpdateTask(task.ID, TaskUpdate{Status: &status})
	require.NoError(t, err)
	assert.Equal(t, model.StatusInProgress, updated.Status)
}

func TestUpdateTask_UpdatesTimestamp(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")
	task, _ := s.CreateTask("Task", p.ID, TaskCreateOpts{})

	status := model.StatusInProgress
	updated, err := s.UpdateTask(task.ID, TaskUpdate{Status: &status})
	require.NoError(t, err)
	assert.True(t, !updated.UpdatedAt.Before(task.UpdatedAt))
}

func TestAllTaskMap(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("P", "")
	t1, _ := s.CreateTask("T1", p.ID, TaskCreateOpts{})
	t2, _ := s.CreateTask("T2", p.ID, TaskCreateOpts{})

	m, err := s.AllTaskMap(p.ID)
	require.NoError(t, err)
	assert.Contains(t, m, t1.ID)
	assert.Contains(t, m, t2.ID)
}

// --- Search tests ---

func TestSearch_MatchTitle(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Authentication Service", "")
	s.CreateEpic("Auth Epic", p.ID, "")
	s.CreateTask("Login Form", p.ID, TaskCreateOpts{})

	results, err := s.Search("auth", "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 2)
}

func TestSearch_MatchBody(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Project", "")
	s.CreateDocument("Doc", p.ID, "This mentions authentication details.")

	results, err := s.Search("authentication", "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestSearch_CaseInsensitive(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject("Authentication", "")

	results, err := s.Search("AUTHENTICATION", "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestSearch_NoResults(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject("Test", "")

	results, err := s.Search("nonexistent", "")
	require.NoError(t, err)
	assert.Empty(t, results)
}

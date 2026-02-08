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
	require.NoError(t, s.EnsureProjectDirs("TEST"))

	for _, sub := range []string{"documents", "tasks"} {
		assert.DirExists(t, s.ProjectDir("TEST")+"/"+sub)
	}
}

func TestWriteEntity_ReadEntity_RoundTrip(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.EnsureProjectDirs("TEST"))

	task := &model.Task{
		ID:      "TEST-TABCDE",
		Title:   "Test Task",
		Type:    model.TypeTask,
		Project: "TEST",
		Status:  model.StatusOpen,
	}
	path := s.ProjectDir("TEST") + "/tasks/TEST-TABCDE.md"
	require.NoError(t, s.WriteEntity(path, task, "some body"))

	got, body, err := ReadEntity[model.Task](path)
	require.NoError(t, err)
	assert.Equal(t, "TEST-TABCDE", got.ID)
	assert.Equal(t, "Test Task", got.Title)
	assert.Equal(t, model.TypeTask, got.Type)
	assert.Equal(t, "some body", body)
}

func TestResolveEntityPath_Task(t *testing.T) {
	s := newTestStore(t)
	p, err := s.CreateProject("Test Project", "", "")
	require.NoError(t, err)

	task, err := s.CreateTask("My Task", p.ID, TaskCreateOpts{})
	require.NoError(t, err)

	path, err := s.ResolveEntityPath(task.ID)
	require.NoError(t, err)
	assert.Contains(t, path, task.ID+".md")
}

func TestResolveEntityPath_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.ResolveEntityPath("ZZZZ-TZZZZZ")
	assert.Error(t, err)
}

func TestResolveEntityPath_AcrossProjects(t *testing.T) {
	s := newTestStore(t)
	p1, err := s.CreateProject("Project One", "", "")
	require.NoError(t, err)
	p2, err := s.CreateProject("Project Two", "", "")
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
	p, err := s.CreateProject("My Project", "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, p.ID)
	assert.Equal(t, "My Project", p.Name)
	assert.NotEmpty(t, p.CreatedBy)
	assert.False(t, p.CreatedAt.IsZero())
	assert.False(t, p.UpdatedAt.IsZero())
}

func TestCreateProject_AutoKey(t *testing.T) {
	s := newTestStore(t)
	p, err := s.CreateProject("Authentication Service", "", "")
	require.NoError(t, err)
	assert.Equal(t, "AUTH", p.ID)
}

func TestCreateProject_AutoKeyCollision(t *testing.T) {
	s := newTestStore(t)
	p1, err := s.CreateProject("Authentication", "", "")
	require.NoError(t, err)
	assert.Equal(t, "AUTH", p1.ID)

	p2, err := s.CreateProject("Authorization", "", "")
	require.NoError(t, err)
	assert.Equal(t, "AUTH2", p2.ID)
}

func TestCreateProject_ExplicitKey(t *testing.T) {
	s := newTestStore(t)
	p, err := s.CreateProject("Backend API", "API", "")
	require.NoError(t, err)
	assert.Equal(t, "API", p.ID)
}

func TestCreateProject_ExplicitKeyCollision(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CreateProject("Backend API", "API", "")
	require.NoError(t, err)

	_, err = s.CreateProject("Another API", "API", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestCreateProject_EmptyName(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CreateProject("", "", "")
	assert.Error(t, err)
}

func TestCreateProject_ShortNameNeedsKey(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CreateProject("X", "", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "need at least 2 alpha")
}

func TestGetProject(t *testing.T) {
	s := newTestStore(t)
	p, err := s.CreateProject("Test Project", "TP", "project body")
	require.NoError(t, err)

	got, body, err := s.GetProject(p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, "Test Project", got.Name)
	assert.Equal(t, "project body", body)
}

func TestGetProject_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, _, err := s.GetProject("ZZZZ")
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
	s.CreateProject("Project One", "PR", "")
	s.CreateProject("Second Proj", "SP", "")
	s.CreateProject("Third Thing", "TH", "")

	projects, err := s.ListProjects()
	require.NoError(t, err)
	assert.Len(t, projects, 3)
}

// --- Document tests ---

func TestCreateDocument(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	d, err := s.CreateDocument("My Doc", p.ID, "")
	require.NoError(t, err)
	assert.NotEmpty(t, d.ID)
	assert.Equal(t, "My Doc", d.Title)
	assert.Equal(t, p.ID, d.Project)
}

func TestCreateDocument_WithBody(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	d, err := s.CreateDocument("Doc", p.ID, "# Hello\n\nBody content.")
	require.NoError(t, err)

	_, body, err := s.GetDocument(d.ID)
	require.NoError(t, err)
	assert.Equal(t, "# Hello\n\nBody content.", body)
}

func TestCreateDocument_InvalidProject(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CreateDocument("Doc", "ZZZZ", "")
	assert.Error(t, err)
}

func TestListDocuments_FilterByProject(t *testing.T) {
	s := newTestStore(t)
	p1, _ := s.CreateProject("Project One", "PR", "")
	p2, _ := s.CreateProject("Second Proj", "SP", "")
	s.CreateDocument("D1", p1.ID, "")
	s.CreateDocument("D2", p1.ID, "")
	s.CreateDocument("D3", p2.ID, "")

	docs, err := s.ListDocuments(p1.ID)
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestListDocuments_AllProjects(t *testing.T) {
	s := newTestStore(t)
	p1, _ := s.CreateProject("Project One", "PR", "")
	p2, _ := s.CreateProject("Second Proj", "SP", "")
	s.CreateDocument("D1", p1.ID, "")
	s.CreateDocument("D2", p2.ID, "")

	docs, err := s.ListDocuments("")
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestUpdateDocument(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	d, _ := s.CreateDocument("Original", p.ID, "old body")

	newTitle := "Updated"
	newBody := "new body"
	updated, err := s.UpdateDocument(d.ID, &newTitle, &newBody)
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Title)

	_, body, _ := s.GetDocument(d.ID)
	assert.Equal(t, "new body", body)
}

// --- Task tests ---

func TestCreateTask_Minimal(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	task, err := s.CreateTask("Task", p.ID, TaskCreateOpts{})
	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)
	assert.Equal(t, model.TypeTask, task.Type)
	assert.Equal(t, model.StatusOpen, task.Status)
	assert.Empty(t, task.Epic)
	assert.Empty(t, task.DependsOn)
}

func TestCreateTask_EpicType(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	epic, err := s.CreateTask("Auth Epic", p.ID, TaskCreateOpts{Type: model.TypeEpic})
	require.NoError(t, err)
	assert.Equal(t, model.TypeEpic, epic.Type)
	assert.Equal(t, model.StatusOpen, epic.Status)
}

func TestCreateTask_WithEpic(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	epic, _ := s.CreateTask("Epic", p.ID, TaskCreateOpts{Type: model.TypeEpic})

	task, err := s.CreateTask("Task", p.ID, TaskCreateOpts{Epic: epic.ID})
	require.NoError(t, err)
	assert.Equal(t, epic.ID, task.Epic)
}

func TestCreateTask_EpicRefMustBeEpicType(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	regularTask, _ := s.CreateTask("Not Epic", p.ID, TaskCreateOpts{})

	_, err := s.CreateTask("Task", p.ID, TaskCreateOpts{Epic: regularTask.ID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not an epic-type task")
}

func TestCreateTask_WithDependencies(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	t1, _ := s.CreateTask("T1", p.ID, TaskCreateOpts{})

	t2, err := s.CreateTask("T2", p.ID, TaskCreateOpts{DependsOn: []string{t1.ID}})
	require.NoError(t, err)
	assert.Equal(t, []string{t1.ID}, t2.DependsOn)
}

func TestCreateTask_CannotDependOnEpic(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	epic, _ := s.CreateTask("Epic", p.ID, TaskCreateOpts{Type: model.TypeEpic})

	_, err := s.CreateTask("Task", p.ID, TaskCreateOpts{DependsOn: []string{epic.ID}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot depend on epic-type task")
}

func TestCreateTask_WithBody(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	task, err := s.CreateTask("Task", p.ID, TaskCreateOpts{Body: "task body"})
	require.NoError(t, err)

	_, body, _ := s.GetTask(task.ID)
	assert.Equal(t, "task body", body)
}

func TestCreateTask_InvalidEpic(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	_, err := s.CreateTask("Task", p.ID, TaskCreateOpts{Epic: "TP-TZZZZZ"})
	assert.Error(t, err)
}

func TestCreateTask_InvalidDependency(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	_, err := s.CreateTask("Task", p.ID, TaskCreateOpts{DependsOn: []string{"TP-TZZZZZ"}})
	assert.Error(t, err)
}

func TestCreateTask_CyclicDep(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	t1, _ := s.CreateTask("T1", p.ID, TaskCreateOpts{})
	t2, _ := s.CreateTask("T2", p.ID, TaskCreateOpts{DependsOn: []string{t1.ID}})

	// T1 depends on T2 would create a cycle
	_, err := s.UpdateTask(t1.ID, TaskUpdate{DependsOn: &[]string{t2.ID}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestListTasks_FilterByStatus(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
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
	p, _ := s.CreateProject("Test Project", "TP", "")
	epic, _ := s.CreateTask("Epic", p.ID, TaskCreateOpts{Type: model.TypeEpic})
	s.CreateTask("T1", p.ID, TaskCreateOpts{Epic: epic.ID})
	s.CreateTask("T2", p.ID, TaskCreateOpts{})

	tasks, err := s.ListTasks(TaskFilter{EpicID: epic.ID})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestListTasks_FilterByType(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	s.CreateTask("T1", p.ID, TaskCreateOpts{})
	s.CreateTask("Epic", p.ID, TaskCreateOpts{Type: model.TypeEpic})

	tasks, err := s.ListTasks(TaskFilter{Type: model.TypeTask})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "T1", tasks[0].Title)
}

func TestUpdateTask_Title(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("Old Title", p.ID, TaskCreateOpts{})

	newTitle := "New Title"
	updated, err := s.UpdateTask(task.ID, TaskUpdate{Title: &newTitle})
	require.NoError(t, err)
	assert.Equal(t, "New Title", updated.Title)

	got, _, _ := s.GetTask(task.ID)
	assert.Equal(t, "New Title", got.Title)
}

func TestUpdateTask_Body(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("Task", p.ID, TaskCreateOpts{Body: "old body"})

	newBody := "new body"
	_, err := s.UpdateTask(task.ID, TaskUpdate{Body: &newBody})
	require.NoError(t, err)

	_, body, _ := s.GetTask(task.ID)
	assert.Equal(t, "new body", body)
}

func TestUpdateTask_Status(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("Task", p.ID, TaskCreateOpts{})

	status := model.StatusInProgress
	updated, err := s.UpdateTask(task.ID, TaskUpdate{Status: &status})
	require.NoError(t, err)
	assert.Equal(t, model.StatusInProgress, updated.Status)
}

func TestUpdateTask_UpdatesTimestamp(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("Task", p.ID, TaskCreateOpts{})

	status := model.StatusInProgress
	updated, err := s.UpdateTask(task.ID, TaskUpdate{Status: &status})
	require.NoError(t, err)
	assert.True(t, !updated.UpdatedAt.Before(task.UpdatedAt))
}

func TestAllTaskMap(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	t1, _ := s.CreateTask("T1", p.ID, TaskCreateOpts{})
	t2, _ := s.CreateTask("T2", p.ID, TaskCreateOpts{})

	m, err := s.AllTaskMap(p.ID)
	require.NoError(t, err)
	assert.Contains(t, m, t1.ID)
	assert.Contains(t, m, t2.ID)
}

// --- Ready tests ---

func TestReadyTasks_NoTasks(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	ready, err := s.ReadyTasks(p.ID)
	require.NoError(t, err)
	assert.Empty(t, ready)
}

func TestReadyTasks_AllReady(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	s.CreateTask("T1", p.ID, TaskCreateOpts{})
	s.CreateTask("T2", p.ID, TaskCreateOpts{})

	ready, err := s.ReadyTasks(p.ID)
	require.NoError(t, err)
	assert.Len(t, ready, 2)
}

func TestReadyTasks_BlockedExcluded(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	t1, _ := s.CreateTask("T1", p.ID, TaskCreateOpts{})
	s.CreateTask("T2", p.ID, TaskCreateOpts{DependsOn: []string{t1.ID}})

	ready, err := s.ReadyTasks(p.ID)
	require.NoError(t, err)
	assert.Len(t, ready, 1)
	assert.Equal(t, t1.ID, ready[0].ID)
}

func TestReadyTasks_ClosedExcluded(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	t1, _ := s.CreateTask("T1", p.ID, TaskCreateOpts{})
	closed := model.StatusClosed
	s.UpdateTask(t1.ID, TaskUpdate{Status: &closed})

	ready, err := s.ReadyTasks(p.ID)
	require.NoError(t, err)
	assert.Empty(t, ready)
}

func TestReadyTasks_EpicsExcluded(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	s.CreateTask("Epic", p.ID, TaskCreateOpts{Type: model.TypeEpic})
	s.CreateTask("Task", p.ID, TaskCreateOpts{})

	ready, err := s.ReadyTasks(p.ID)
	require.NoError(t, err)
	assert.Len(t, ready, 1)
	assert.Equal(t, "Task", ready[0].Title)
}

func TestReadyTasks_UnblocksAfterClose(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	t1, _ := s.CreateTask("T1", p.ID, TaskCreateOpts{})
	t2, _ := s.CreateTask("T2", p.ID, TaskCreateOpts{DependsOn: []string{t1.ID}})

	// Before closing t1, only t1 is ready
	ready, err := s.ReadyTasks(p.ID)
	require.NoError(t, err)
	assert.Len(t, ready, 1)

	// Close t1, t2 becomes ready
	closed := model.StatusClosed
	s.UpdateTask(t1.ID, TaskUpdate{Status: &closed})

	ready, err = s.ReadyTasks(p.ID)
	require.NoError(t, err)
	assert.Len(t, ready, 1)
	assert.Equal(t, t2.ID, ready[0].ID)
}

// --- Priority tests ---

func TestCreateTask_WithPriority(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	pri := 1
	task, err := s.CreateTask("Task", p.ID, TaskCreateOpts{Priority: &pri})
	require.NoError(t, err)
	require.NotNil(t, task.Priority)
	assert.Equal(t, 1, *task.Priority)

	got, _, err := s.GetTask(task.ID)
	require.NoError(t, err)
	require.NotNil(t, got.Priority)
	assert.Equal(t, 1, *got.Priority)
}

func TestCreateTask_WithoutPriority(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	task, err := s.CreateTask("Task", p.ID, TaskCreateOpts{})
	require.NoError(t, err)
	assert.Nil(t, task.Priority)

	got, _, err := s.GetTask(task.ID)
	require.NoError(t, err)
	assert.Nil(t, got.Priority)
}

func TestCreateTask_InvalidPriority(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")

	pri := 5
	_, err := s.CreateTask("Task", p.ID, TaskCreateOpts{Priority: &pri})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid priority")
}

func TestUpdateTask_SetPriority(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("Task", p.ID, TaskCreateOpts{})

	pri := 2
	pp := &pri
	updated, err := s.UpdateTask(task.ID, TaskUpdate{Priority: &pp})
	require.NoError(t, err)
	require.NotNil(t, updated.Priority)
	assert.Equal(t, 2, *updated.Priority)
}

func TestUpdateTask_ClearPriority(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	pri := 1
	task, _ := s.CreateTask("Task", p.ID, TaskCreateOpts{Priority: &pri})

	var cleared *int
	updated, err := s.UpdateTask(task.ID, TaskUpdate{Priority: &cleared})
	require.NoError(t, err)
	assert.Nil(t, updated.Priority)
}

func TestUpdateTask_InvalidPriority(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("Task", p.ID, TaskCreateOpts{})

	pri := 4
	pp := &pri
	_, err := s.UpdateTask(task.ID, TaskUpdate{Priority: &pp})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid priority")
}

// --- Delete tests ---

func TestDeleteTask(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("Task", p.ID, TaskCreateOpts{})

	require.NoError(t, s.DeleteTask(task.ID))

	_, _, err := s.GetTask(task.ID)
	assert.Error(t, err)
}

func TestDeleteTask_NotFound(t *testing.T) {
	s := newTestStore(t)
	assert.Error(t, s.DeleteTask("ZZZZ-TZZZZZ"))
}

func TestDeleteDocument(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	d, _ := s.CreateDocument("Doc", p.ID, "body")

	require.NoError(t, s.DeleteDocument(d.ID))

	_, _, err := s.GetDocument(d.ID)
	assert.Error(t, err)
}

func TestDeleteDocument_NotFound(t *testing.T) {
	s := newTestStore(t)
	assert.Error(t, s.DeleteDocument("ZZZZ-DZZZZZ"))
}

// --- Search tests ---

func TestSearch_MatchTitle(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Authentication Service", "", "")
	s.CreateTask("Auth Epic", p.ID, TaskCreateOpts{Type: model.TypeEpic})
	s.CreateTask("Login Form", p.ID, TaskCreateOpts{})

	results, err := s.Search("auth", "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 2)
}

func TestSearch_MatchBody(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Project Test", "", "")
	s.CreateDocument("Doc", p.ID, "This mentions authentication details.")

	results, err := s.Search("authentication", "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestSearch_CaseInsensitive(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject("Authentication", "", "")

	results, err := s.Search("AUTHENTICATION", "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestSearch_NoResults(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject("Test Project", "TP", "")

	results, err := s.Search("nonexistent", "")
	require.NoError(t, err)
	assert.Empty(t, results)
}

// --- Checkout/Checkin tests ---

func TestCheckoutEntity_Task(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("My Task", p.ID, TaskCreateOpts{Body: "task body"})

	destDir := t.TempDir()
	localPath, err := s.CheckoutEntity(task.ID, destDir)
	require.NoError(t, err)
	assert.FileExists(t, localPath)
	assert.Contains(t, localPath, task.ID+".md")

	// Verify contents match
	got, body, err := ReadEntity[model.Task](localPath)
	require.NoError(t, err)
	assert.Equal(t, task.ID, got.ID)
	assert.Equal(t, "My Task", got.Title)
	assert.Equal(t, "task body", body)
}

func TestCheckoutEntity_Document(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	doc, _ := s.CreateDocument("My Doc", p.ID, "doc body")

	destDir := t.TempDir()
	localPath, err := s.CheckoutEntity(doc.ID, destDir)
	require.NoError(t, err)
	assert.FileExists(t, localPath)

	got, body, err := ReadEntity[model.Document](localPath)
	require.NoError(t, err)
	assert.Equal(t, doc.ID, got.ID)
	assert.Equal(t, "doc body", body)
}

func TestCheckoutEntity_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CheckoutEntity("ZZZZ-TZZZZZ", t.TempDir())
	assert.Error(t, err)
}

func TestCheckinTask_RoundTrip(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("Original", p.ID, TaskCreateOpts{Body: "old body"})

	destDir := t.TempDir()
	localPath, err := s.CheckoutEntity(task.ID, destDir)
	require.NoError(t, err)

	// Modify the local file: change the body
	modified, _, err := ReadEntity[model.Task](localPath)
	require.NoError(t, err)
	require.NoError(t, s.WriteEntity(localPath, &modified, "new body"))

	// Checkin
	result, err := s.CheckinTask(localPath)
	require.NoError(t, err)
	assert.Equal(t, task.ID, result.ID)

	// Local file should be removed
	assert.NoFileExists(t, localPath)

	// Store should have the updated body
	_, body, err := s.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, "new body", body)
}

func TestCheckinDocument_RoundTrip(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	doc, _ := s.CreateDocument("Original", p.ID, "old body")

	destDir := t.TempDir()
	localPath, err := s.CheckoutEntity(doc.ID, destDir)
	require.NoError(t, err)

	// Modify the local file
	modified, _, err := ReadEntity[model.Document](localPath)
	require.NoError(t, err)
	modified.Title = "Updated Title"
	require.NoError(t, s.WriteEntity(localPath, &modified, "new body"))

	// Checkin
	result, err := s.CheckinDocument(localPath)
	require.NoError(t, err)
	assert.Equal(t, doc.ID, result.ID)

	// Local file should be removed
	assert.NoFileExists(t, localPath)

	// Store should have updates
	got, body, err := s.GetDocument(doc.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", got.Title)
	assert.Equal(t, "new body", body)
}

func TestCheckinTask_InvalidFrontmatter(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	task, _ := s.CreateTask("Task", p.ID, TaskCreateOpts{})

	destDir := t.TempDir()
	localPath, err := s.CheckoutEntity(task.ID, destDir)
	require.NoError(t, err)

	// Corrupt the file: clear the title to trigger validation error
	modified, body, err := ReadEntity[model.Task](localPath)
	require.NoError(t, err)
	modified.Title = ""
	require.NoError(t, s.WriteEntity(localPath, &modified, body))

	_, err = s.CheckinTask(localPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title")

	// Local file should NOT be removed on error
	assert.FileExists(t, localPath)
}

func TestCheckinDocument_InvalidFrontmatter(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Test Project", "TP", "")
	doc, _ := s.CreateDocument("Doc", p.ID, "body")

	destDir := t.TempDir()
	localPath, err := s.CheckoutEntity(doc.ID, destDir)
	require.NoError(t, err)

	// Clear the title
	modified, body, err := ReadEntity[model.Document](localPath)
	require.NoError(t, err)
	modified.Title = ""
	require.NoError(t, s.WriteEntity(localPath, &modified, body))

	_, err = s.CheckinDocument(localPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title")
	assert.FileExists(t, localPath)
}

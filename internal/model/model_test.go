package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProject_Validate_Valid(t *testing.T) {
	p := &Project{ID: "AUTH", Name: "Test"}
	assert.NoError(t, p.Validate())
}

func TestProject_Validate_MissingName(t *testing.T) {
	p := &Project{ID: "AUTH"}
	assert.Error(t, p.Validate())
}

func TestProject_Validate_MissingID(t *testing.T) {
	p := &Project{Name: "Test"}
	assert.Error(t, p.Validate())
}

func TestTask_Validate_ValidStatuses(t *testing.T) {
	for _, s := range []Status{StatusOpen, StatusInProgress, StatusClosed} {
		task := &Task{ID: "TEST-TABCDE", Title: "Test", Project: "TEST", Type: TypeTask, Status: s}
		assert.NoError(t, task.Validate())
	}
}

func TestTask_Validate_DefaultTypeInvalid(t *testing.T) {
	task := &Task{ID: "TEST-TABCDE", Title: "Test", Project: "TEST", Status: StatusOpen}
	assert.Error(t, task.Validate())
}

func TestTask_Validate_InvalidType(t *testing.T) {
	task := &Task{ID: "TEST-TABCDE", Title: "Test", Project: "TEST", Type: "story", Status: StatusOpen}
	assert.Error(t, task.Validate())
}

func TestTask_Validate_EpicType(t *testing.T) {
	task := &Task{ID: "TEST-TABCDE", Title: "Test", Project: "TEST", Type: TypeEpic}
	assert.NoError(t, task.Validate())
}

func TestTask_Validate_EpicRejectsStoredStatus(t *testing.T) {
	task := &Task{ID: "TEST-TABCDE", Title: "Test", Project: "TEST", Type: TypeEpic, Status: StatusOpen}
	assert.Error(t, task.Validate())
	assert.Contains(t, task.Validate().Error(), "must not have a stored status")
}

func TestTask_Validate_EpicCannotHaveDeps(t *testing.T) {
	task := &Task{
		ID: "TEST-TABCDE", Title: "Test", Project: "TEST",
		Type: TypeEpic, DependsOn: []string{"TEST-T22222"},
	}
	assert.Error(t, task.Validate())
	assert.Contains(t, task.Validate().Error(), "epic-type tasks cannot have dependencies")
}

func TestTask_Validate_SelfDependency(t *testing.T) {
	task := &Task{
		ID: "TEST-TABCDE", Title: "Test", Project: "TEST",
		Type: TypeTask, Status: StatusOpen, DependsOn: []string{"TEST-TABCDE"},
	}
	assert.Error(t, task.Validate())
}

func TestTask_Validate_DuplicateDeps(t *testing.T) {
	task := &Task{
		ID: "TEST-TABCDE", Title: "Test", Project: "TEST",
		Type: TypeTask, Status: StatusOpen, DependsOn: []string{"TEST-T22222", "TEST-T22222"},
	}
	assert.Error(t, task.Validate())
}

// --- ComputeEpicStatus tests ---

func TestComputeEpicStatus_NoChildren(t *testing.T) {
	assert.Equal(t, StatusOpen, ComputeEpicStatus(nil))
}

func TestComputeEpicStatus_AllOpen(t *testing.T) {
	children := []*Task{
		{Status: StatusOpen},
		{Status: StatusOpen},
	}
	assert.Equal(t, StatusOpen, ComputeEpicStatus(children))
}

func TestComputeEpicStatus_AnyInProgress(t *testing.T) {
	children := []*Task{
		{Status: StatusOpen},
		{Status: StatusInProgress},
		{Status: StatusClosed},
	}
	assert.Equal(t, StatusInProgress, ComputeEpicStatus(children))
}

func TestComputeEpicStatus_AllClosed(t *testing.T) {
	children := []*Task{
		{Status: StatusClosed},
		{Status: StatusClosed},
	}
	assert.Equal(t, StatusClosed, ComputeEpicStatus(children))
}

func TestComputeEpicStatus_MixOpenClosed(t *testing.T) {
	children := []*Task{
		{Status: StatusOpen},
		{Status: StatusClosed},
	}
	assert.Equal(t, StatusOpen, ComputeEpicStatus(children))
}

// --- ChildrenOf tests ---

func TestChildrenOf(t *testing.T) {
	allTasks := map[string]*Task{
		"TP-T11111": {ID: "TP-T11111", Epic: "TP-TEPIC1"},
		"TP-T22222": {ID: "TP-T22222", Epic: "TP-TEPIC1"},
		"TP-T33333": {ID: "TP-T33333", Epic: ""},
		"TP-TEPIC1": {ID: "TP-TEPIC1", Type: TypeEpic},
	}
	children := ChildrenOf("TP-TEPIC1", allTasks)
	assert.Len(t, children, 2)
}

func TestChildrenOf_NoChildren(t *testing.T) {
	allTasks := map[string]*Task{
		"TP-T11111": {ID: "TP-T11111", Epic: ""},
		"TP-TEPIC1": {ID: "TP-TEPIC1", Type: TypeEpic},
	}
	children := ChildrenOf("TP-TEPIC1", allTasks)
	assert.Empty(t, children)
}

// --- IsBlocked tests ---

func TestTask_IsBlocked_NoDeps(t *testing.T) {
	task := &Task{ID: "TEST-TABCDE", Status: StatusOpen}
	assert.False(t, task.IsBlocked(nil))
}

func TestTask_IsBlocked_AllDepsClosed(t *testing.T) {
	all := map[string]*Task{
		"TEST-T11111": {ID: "TEST-T11111", Status: StatusClosed},
	}
	task := &Task{ID: "TEST-TABCDE", Status: StatusOpen, DependsOn: []string{"TEST-T11111"}}
	assert.False(t, task.IsBlocked(all))
}

func TestTask_IsBlocked_SomeDepOpen(t *testing.T) {
	all := map[string]*Task{
		"TEST-T11111": {ID: "TEST-T11111", Status: StatusOpen},
	}
	task := &Task{ID: "TEST-TABCDE", Status: StatusOpen, DependsOn: []string{"TEST-T11111"}}
	assert.True(t, task.IsBlocked(all))
}

func TestTask_IsBlocked_MixedStatuses(t *testing.T) {
	all := map[string]*Task{
		"TEST-T11111": {ID: "TEST-T11111", Status: StatusClosed},
		"TEST-T22222": {ID: "TEST-T22222", Status: StatusInProgress},
	}
	task := &Task{
		ID: "TEST-TABCDE", Status: StatusOpen,
		DependsOn: []string{"TEST-T11111", "TEST-T22222"},
	}
	assert.True(t, task.IsBlocked(all))
}

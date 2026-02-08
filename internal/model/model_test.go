package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProject_Validate_Valid(t *testing.T) {
	p := &Project{ID: "PROJ-ABCDE", Name: "Test"}
	assert.NoError(t, p.Validate())
}

func TestProject_Validate_MissingName(t *testing.T) {
	p := &Project{ID: "PROJ-ABCDE"}
	assert.Error(t, p.Validate())
}

func TestProject_Validate_MissingID(t *testing.T) {
	p := &Project{Name: "Test"}
	assert.Error(t, p.Validate())
}

func TestEpic_Validate_InvalidStatus(t *testing.T) {
	e := &Epic{ID: "EPIC-ABCDE", Title: "Test", Project: "PROJ-ABCDE", Status: "done"}
	assert.Error(t, e.Validate())
}

func TestTask_Validate_ValidStatuses(t *testing.T) {
	for _, s := range []Status{StatusOpen, StatusInProgress, StatusClosed} {
		task := &Task{ID: "TASK-ABCDE", Title: "Test", Project: "PROJ-ABCDE", Status: s}
		assert.NoError(t, task.Validate())
	}
}

func TestTask_Validate_SelfDependency(t *testing.T) {
	task := &Task{
		ID: "TASK-ABCDE", Title: "Test", Project: "PROJ-ABCDE",
		Status: StatusOpen, DependsOn: []string{"TASK-ABCDE"},
	}
	assert.Error(t, task.Validate())
}

func TestTask_Validate_DuplicateDeps(t *testing.T) {
	task := &Task{
		ID: "TASK-ABCDE", Title: "Test", Project: "PROJ-ABCDE",
		Status: StatusOpen, DependsOn: []string{"TASK-22222", "TASK-22222"},
	}
	assert.Error(t, task.Validate())
}

func TestTask_IsBlocked_NoDeps(t *testing.T) {
	task := &Task{ID: "TASK-ABCDE", Status: StatusOpen}
	assert.False(t, task.IsBlocked(nil))
}

func TestTask_IsBlocked_AllDepsClosed(t *testing.T) {
	all := map[string]*Task{
		"TASK-11111": {ID: "TASK-11111", Status: StatusClosed},
	}
	task := &Task{ID: "TASK-ABCDE", Status: StatusOpen, DependsOn: []string{"TASK-11111"}}
	assert.False(t, task.IsBlocked(all))
}

func TestTask_IsBlocked_SomeDepOpen(t *testing.T) {
	all := map[string]*Task{
		"TASK-11111": {ID: "TASK-11111", Status: StatusOpen},
	}
	task := &Task{ID: "TASK-ABCDE", Status: StatusOpen, DependsOn: []string{"TASK-11111"}}
	assert.True(t, task.IsBlocked(all))
}

func TestTask_IsBlocked_MixedStatuses(t *testing.T) {
	all := map[string]*Task{
		"TASK-11111": {ID: "TASK-11111", Status: StatusClosed},
		"TASK-22222": {ID: "TASK-22222", Status: StatusInProgress},
	}
	task := &Task{
		ID: "TASK-ABCDE", Status: StatusOpen,
		DependsOn: []string{"TASK-11111", "TASK-22222"},
	}
	assert.True(t, task.IsBlocked(all))
}

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
	task := &Task{ID: "TEST-TABCDE", Title: "Test", Project: "TEST", Type: TypeEpic, Status: StatusOpen}
	assert.NoError(t, task.Validate())
}

func TestTask_Validate_EpicCannotHaveDeps(t *testing.T) {
	task := &Task{
		ID: "TEST-TABCDE", Title: "Test", Project: "TEST",
		Type: TypeEpic, Status: StatusOpen, DependsOn: []string{"TEST-T22222"},
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

package model

import (
	"fmt"
	"time"
)

type TaskType string

const (
	TypeTask TaskType = "task"
	TypeEpic TaskType = "epic"
)

type Task struct {
	ID        string   `yaml:"id"`
	Title     string   `yaml:"title"`
	Type      TaskType `yaml:"type"`
	Project   string   `yaml:"project"`
	Epic      string   `yaml:"epic,omitempty"`
	Status    Status   `yaml:"status"`
	DependsOn []string `yaml:"depends_on,omitempty"`
	CreatedBy string   `yaml:"created_by"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

func (t *Task) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("task id is required")
	}
	if t.Title == "" {
		return fmt.Errorf("task title is required")
	}
	if t.Project == "" {
		return fmt.Errorf("task project is required")
	}
	if t.Type != TypeTask && t.Type != TypeEpic {
		return fmt.Errorf("invalid task type %q: must be task or epic", t.Type)
	}
	if err := ValidateStatus(t.Status); err != nil {
		return err
	}
	if t.Type == TypeEpic && len(t.DependsOn) > 0 {
		return fmt.Errorf("epic-type tasks cannot have dependencies")
	}
	seen := make(map[string]bool)
	for _, dep := range t.DependsOn {
		if dep == t.ID {
			return fmt.Errorf("task cannot depend on itself")
		}
		if seen[dep] {
			return fmt.Errorf("duplicate dependency %q", dep)
		}
		seen[dep] = true
	}
	return nil
}

// IsBlocked returns true if any dependency is not closed.
func (t *Task) IsBlocked(allTasks map[string]*Task) bool {
	for _, dep := range t.DependsOn {
		dt, ok := allTasks[dep]
		if !ok || dt.Status != StatusClosed {
			return true
		}
	}
	return false
}

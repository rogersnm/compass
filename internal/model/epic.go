package model

import (
	"fmt"
	"time"
)

type Epic struct {
	ID        string    `yaml:"id"`
	Title     string    `yaml:"title"`
	Project   string    `yaml:"project"`
	Status    Status    `yaml:"status"`
	CreatedBy string    `yaml:"created_by"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

func (e *Epic) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("epic id is required")
	}
	if e.Title == "" {
		return fmt.Errorf("epic title is required")
	}
	if e.Project == "" {
		return fmt.Errorf("epic project is required")
	}
	return ValidateStatus(e.Status)
}

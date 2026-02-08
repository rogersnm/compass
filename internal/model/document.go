package model

import (
	"fmt"
	"time"
)

type Document struct {
	ID        string    `yaml:"id"`
	Title     string    `yaml:"title"`
	Project   string    `yaml:"project"`
	CreatedBy string    `yaml:"created_by"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

func (d *Document) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("document id is required")
	}
	if d.Title == "" {
		return fmt.Errorf("document title is required")
	}
	if d.Project == "" {
		return fmt.Errorf("document project is required")
	}
	return nil
}

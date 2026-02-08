package model

import (
	"fmt"
	"time"
)

type Project struct {
	ID        string    `yaml:"id"`
	Name      string    `yaml:"name"`
	CreatedBy string    `yaml:"created_by"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

func (p *Project) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("project id is required")
	}
	if p.Name == "" {
		return fmt.Errorf("project name is required")
	}
	return nil
}

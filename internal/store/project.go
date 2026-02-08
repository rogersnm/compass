package store

import (
	"fmt"
	"path/filepath"

	"github.com/rogersnm/compass/internal/id"
	"github.com/rogersnm/compass/internal/model"
)

func (s *Store) CreateProject(name, body string) (*model.Project, error) {
	pid, err := id.New(id.Project)
	if err != nil {
		return nil, err
	}

	p := &model.Project{
		ID:        pid,
		Name:      name,
		CreatedBy: currentUser(),
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}

	if err := s.EnsureProjectDirs(pid); err != nil {
		return nil, err
	}

	path := filepath.Join(s.ProjectDir(pid), "project.md")
	if err := s.WriteEntity(path, p, body); err != nil {
		return nil, fmt.Errorf("writing project: %w", err)
	}
	return p, nil
}

func (s *Store) GetProject(projectID string) (*model.Project, string, error) {
	path, err := s.ResolveEntityPath(projectID)
	if err != nil {
		return nil, "", err
	}
	p, body, err := ReadEntity[model.Project](path)
	if err != nil {
		return nil, "", err
	}
	return &p, body, nil
}

func (s *Store) ListProjects() ([]model.Project, error) {
	dirs, err := s.listProjectDirs()
	if err != nil {
		return nil, err
	}

	var projects []model.Project
	for _, d := range dirs {
		path := filepath.Join(d, "project.md")
		p, _, err := ReadEntity[model.Project](path)
		if err != nil {
			continue
		}
		projects = append(projects, p)
	}
	return projects, nil
}

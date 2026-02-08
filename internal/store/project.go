package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rogersnm/compass/internal/id"
	"github.com/rogersnm/compass/internal/model"
)

func (s *LocalStore) CreateProject(name, key, body string) (*model.Project, error) {
	if key == "" {
		generated, err := id.GenerateKey(name)
		if err != nil {
			return nil, err
		}
		// Check for collision and append digit if needed
		candidate := generated
		if s.projectKeyExists(candidate) {
			found := false
			for i := 2; i <= 9; i++ {
				candidate = fmt.Sprintf("%s%d", generated, i)
				if err := id.ValidateKey(candidate); err != nil {
					break // key would be too long
				}
				if !s.projectKeyExists(candidate) {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("cannot auto-generate unique key for %q: all variants taken (use --key)", name)
			}
		}
		key = candidate
	} else {
		if err := id.ValidateKey(key); err != nil {
			return nil, err
		}
		if s.projectKeyExists(key) {
			return nil, fmt.Errorf("project key %q already exists", key)
		}
	}

	p := &model.Project{
		ID:        key,
		Name:      name,
		CreatedBy: currentUser(),
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}

	if err := s.EnsureProjectDirs(key); err != nil {
		return nil, err
	}

	path := filepath.Join(s.ProjectDir(key), "project.md")
	if err := s.WriteEntity(path, p, body); err != nil {
		return nil, fmt.Errorf("writing project: %w", err)
	}
	return p, nil
}

func (s *LocalStore) projectKeyExists(key string) bool {
	_, err := os.Stat(s.ProjectDir(key))
	return err == nil
}

func (s *LocalStore) GetProject(projectID string) (*model.Project, string, error) {
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

func (s *LocalStore) DeleteProject(projectID string) error {
	dir := s.ProjectDir(projectID)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("%s not found", projectID)
	}
	return os.RemoveAll(dir)
}

func (s *LocalStore) ListProjects() ([]model.Project, error) {
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

package store

import (
	"fmt"
	"path/filepath"

	"github.com/rogersnm/compass/internal/id"
	"github.com/rogersnm/compass/internal/model"
)

func (s *Store) CreateEpic(title, projectID, body string) (*model.Epic, error) {
	if _, _, err := s.GetProject(projectID); err != nil {
		return nil, fmt.Errorf("project %s not found", projectID)
	}

	eid, err := id.New(id.Epic)
	if err != nil {
		return nil, err
	}

	e := &model.Epic{
		ID:        eid,
		Title:     title,
		Project:   projectID,
		Status:    model.StatusOpen,
		CreatedBy: currentUser(),
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	if err := e.Validate(); err != nil {
		return nil, err
	}

	path := filepath.Join(s.ProjectDir(projectID), "epics", eid+".md")
	if err := s.WriteEntity(path, e, body); err != nil {
		return nil, fmt.Errorf("writing epic: %w", err)
	}
	return e, nil
}

func (s *Store) GetEpic(epicID string) (*model.Epic, string, error) {
	path, err := s.ResolveEntityPath(epicID)
	if err != nil {
		return nil, "", err
	}
	e, body, err := ReadEntity[model.Epic](path)
	if err != nil {
		return nil, "", err
	}
	return &e, body, nil
}

func (s *Store) ListEpics(projectID string) ([]model.Epic, error) {
	var dirs []string
	if projectID != "" {
		dirs = []string{s.ProjectDir(projectID)}
	} else {
		var err error
		dirs, err = s.listProjectDirs()
		if err != nil {
			return nil, err
		}
	}

	var epics []model.Epic
	for _, d := range dirs {
		epicDir := filepath.Join(d, "epics")
		files, err := s.ListFiles(epicDir, "EPIC-*.md")
		if err != nil {
			continue
		}
		for _, f := range files {
			e, _, err := ReadEntity[model.Epic](f)
			if err != nil {
				continue
			}
			epics = append(epics, e)
		}
	}
	return epics, nil
}

func (s *Store) UpdateEpic(epicID string, status *model.Status) (*model.Epic, error) {
	path, err := s.ResolveEntityPath(epicID)
	if err != nil {
		return nil, err
	}
	e, body, err := ReadEntity[model.Epic](path)
	if err != nil {
		return nil, err
	}

	if status != nil {
		e.Status = *status
	}
	e.UpdatedAt = now()

	if err := e.Validate(); err != nil {
		return nil, err
	}
	if err := s.WriteEntity(path, &e, body); err != nil {
		return nil, err
	}
	return &e, nil
}

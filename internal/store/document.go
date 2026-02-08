package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rogersnm/compass/internal/id"
	"github.com/rogersnm/compass/internal/model"
)

func (s *Store) CreateDocument(title, projectID, body string) (*model.Document, error) {
	if _, _, err := s.GetProject(projectID); err != nil {
		return nil, fmt.Errorf("project %s not found", projectID)
	}

	did, err := id.New(id.Document)
	if err != nil {
		return nil, err
	}

	d := &model.Document{
		ID:        did,
		Title:     title,
		Project:   projectID,
		CreatedBy: currentUser(),
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	if err := d.Validate(); err != nil {
		return nil, err
	}

	path := filepath.Join(s.ProjectDir(projectID), "documents", did+".md")
	if err := s.WriteEntity(path, d, body); err != nil {
		return nil, fmt.Errorf("writing document: %w", err)
	}
	return d, nil
}

func (s *Store) GetDocument(docID string) (*model.Document, string, error) {
	path, err := s.ResolveEntityPath(docID)
	if err != nil {
		return nil, "", err
	}
	d, body, err := ReadEntity[model.Document](path)
	if err != nil {
		return nil, "", err
	}
	return &d, body, nil
}

func (s *Store) ListDocuments(projectID string) ([]model.Document, error) {
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

	var docs []model.Document
	for _, d := range dirs {
		docDir := filepath.Join(d, "documents")
		files, err := s.ListFiles(docDir, "DOC-*.md")
		if err != nil {
			continue
		}
		for _, f := range files {
			doc, _, err := ReadEntity[model.Document](f)
			if err != nil {
				continue
			}
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

func (s *Store) UpdateDocument(docID string, title, body *string) (*model.Document, error) {
	path, err := s.ResolveEntityPath(docID)
	if err != nil {
		return nil, err
	}
	d, existingBody, err := ReadEntity[model.Document](path)
	if err != nil {
		return nil, err
	}

	if title != nil {
		d.Title = *title
	}
	finalBody := existingBody
	if body != nil {
		finalBody = *body
	}
	d.UpdatedAt = now()

	if err := d.Validate(); err != nil {
		return nil, err
	}
	if err := s.WriteEntity(path, &d, finalBody); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) DeleteDocument(docID string) error {
	path, err := s.ResolveEntityPath(docID)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

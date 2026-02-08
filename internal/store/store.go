package store

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/rogersnm/compass/internal/id"
	"github.com/rogersnm/compass/internal/markdown"
)

type Store struct {
	BaseDir string
}

func New(baseDir string) *Store {
	return &Store{BaseDir: baseDir}
}

func (s *Store) ProjectsDir() string {
	return filepath.Join(s.BaseDir, "projects")
}

func (s *Store) ProjectDir(projectID string) string {
	return filepath.Join(s.ProjectsDir(), projectID)
}

func (s *Store) EnsureProjectDirs(projectID string) error {
	dirs := []string{
		filepath.Join(s.ProjectDir(projectID), "documents"),
		filepath.Join(s.ProjectDir(projectID), "epics"),
		filepath.Join(s.ProjectDir(projectID), "tasks"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}
	return nil
}

func (s *Store) WriteEntity(path string, meta any, body string) error {
	data, err := markdown.Marshal(meta, body)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating parent dir: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func ReadEntity[T any](path string) (T, string, error) {
	f, err := os.Open(path)
	if err != nil {
		var zero T
		return zero, "", fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()
	return markdown.Parse[T](f)
}

func (s *Store) ListFiles(dir, pattern string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return nil, fmt.Errorf("globbing %s/%s: %w", dir, pattern, err)
	}
	return matches, nil
}

// ResolveEntityPath finds the file path for an entity ID by scanning project directories.
func (s *Store) ResolveEntityPath(entityID string) (string, error) {
	entityType, err := id.TypeOf(entityID)
	if err != nil {
		return "", err
	}

	projects, err := s.listProjectDirs()
	if err != nil {
		return "", err
	}

	for _, projDir := range projects {
		var searchDir, filename string
		switch entityType {
		case id.Project:
			path := filepath.Join(projDir, "project.md")
			if _, err := os.Stat(path); err == nil {
				meta, _, err := ReadEntity[struct {
					ID string `yaml:"id"`
				}](path)
				if err == nil && meta.ID == entityID {
					return path, nil
				}
			}
			continue
		case id.Document:
			searchDir = filepath.Join(projDir, "documents")
			filename = entityID + ".md"
		case id.Epic:
			searchDir = filepath.Join(projDir, "epics")
			filename = entityID + ".md"
		case id.Task:
			searchDir = filepath.Join(projDir, "tasks")
			filename = entityID + ".md"
		}

		path := filepath.Join(searchDir, filename)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("%s not found (try: compass %s list)", entityID, strings.ToLower(string(entityType)))
}

func (s *Store) listProjectDirs() ([]string, error) {
	entries, err := os.ReadDir(s.ProjectsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(s.ProjectsDir(), e.Name()))
		}
	}
	return dirs, nil
}

func now() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}

func currentUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "unknown"
}

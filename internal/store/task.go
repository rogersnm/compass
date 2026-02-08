package store

import (
	"fmt"
	"path/filepath"

	"github.com/rogersnm/compass/internal/dag"
	"github.com/rogersnm/compass/internal/id"
	"github.com/rogersnm/compass/internal/model"
)

type TaskCreateOpts struct {
	Epic      string
	DependsOn []string
	Body      string
}

type TaskFilter struct {
	ProjectID string
	EpicID    string
	Status    model.Status
}

type TaskUpdate struct {
	Status    *model.Status
	DependsOn *[]string
}

func (s *Store) CreateTask(title, projectID string, opts TaskCreateOpts) (*model.Task, error) {
	if _, _, err := s.GetProject(projectID); err != nil {
		return nil, fmt.Errorf("project %s not found", projectID)
	}

	if opts.Epic != "" {
		if _, _, err := s.GetEpic(opts.Epic); err != nil {
			return nil, fmt.Errorf("epic %s not found", opts.Epic)
		}
	}

	tid, err := id.New(id.Task)
	if err != nil {
		return nil, err
	}

	t := &model.Task{
		ID:        tid,
		Title:     title,
		Project:   projectID,
		Epic:      opts.Epic,
		Status:    model.StatusOpen,
		DependsOn: opts.DependsOn,
		CreatedBy: currentUser(),
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}

	if err := s.validateDeps(t, projectID); err != nil {
		return nil, err
	}

	path := filepath.Join(s.ProjectDir(projectID), "tasks", tid+".md")
	if err := s.WriteEntity(path, t, opts.Body); err != nil {
		return nil, fmt.Errorf("writing task: %w", err)
	}
	return t, nil
}

func (s *Store) GetTask(taskID string) (*model.Task, string, error) {
	path, err := s.ResolveEntityPath(taskID)
	if err != nil {
		return nil, "", err
	}
	t, body, err := ReadEntity[model.Task](path)
	if err != nil {
		return nil, "", err
	}
	return &t, body, nil
}

func (s *Store) ListTasks(filter TaskFilter) ([]model.Task, error) {
	var dirs []string
	if filter.ProjectID != "" {
		dirs = []string{s.ProjectDir(filter.ProjectID)}
	} else {
		var err error
		dirs, err = s.listProjectDirs()
		if err != nil {
			return nil, err
		}
	}

	var tasks []model.Task
	for _, d := range dirs {
		taskDir := filepath.Join(d, "tasks")
		files, err := s.ListFiles(taskDir, "TASK-*.md")
		if err != nil {
			continue
		}
		for _, f := range files {
			t, _, err := ReadEntity[model.Task](f)
			if err != nil {
				continue
			}
			if filter.EpicID != "" && t.Epic != filter.EpicID {
				continue
			}
			if filter.Status != "" && t.Status != filter.Status {
				continue
			}
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}

func (s *Store) UpdateTask(taskID string, upd TaskUpdate) (*model.Task, error) {
	path, err := s.ResolveEntityPath(taskID)
	if err != nil {
		return nil, err
	}
	t, body, err := ReadEntity[model.Task](path)
	if err != nil {
		return nil, err
	}

	if upd.Status != nil {
		t.Status = *upd.Status
	}
	if upd.DependsOn != nil {
		t.DependsOn = *upd.DependsOn
	}
	t.UpdatedAt = now()

	if err := t.Validate(); err != nil {
		return nil, err
	}

	if upd.DependsOn != nil {
		if err := s.validateDeps(&t, t.Project); err != nil {
			return nil, err
		}
	}

	if err := s.WriteEntity(path, &t, body); err != nil {
		return nil, err
	}
	return &t, nil
}

// AllTaskMap returns a map of all tasks in a project, keyed by ID.
func (s *Store) AllTaskMap(projectID string) (map[string]*model.Task, error) {
	tasks, err := s.ListTasks(TaskFilter{ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	m := make(map[string]*model.Task, len(tasks))
	for i := range tasks {
		m[tasks[i].ID] = &tasks[i]
	}
	return m, nil
}

func (s *Store) validateDeps(t *model.Task, projectID string) error {
	for _, dep := range t.DependsOn {
		dt, _, err := s.GetTask(dep)
		if err != nil {
			return fmt.Errorf("dependency %s not found", dep)
		}
		if dt.Project != projectID {
			return fmt.Errorf("dependency %s is in project %s, not %s", dep, dt.Project, projectID)
		}
	}

	existing, err := s.ListTasks(TaskFilter{ProjectID: projectID})
	if err != nil {
		return err
	}

	// Build task list including the new/updated task
	var allTasks []*model.Task
	found := false
	for i := range existing {
		if existing[i].ID == t.ID {
			allTasks = append(allTasks, t)
			found = true
		} else {
			allTasks = append(allTasks, &existing[i])
		}
	}
	if !found {
		allTasks = append(allTasks, t)
	}

	g := dag.BuildFromTasks(allTasks)
	return g.ValidateAcyclic()
}

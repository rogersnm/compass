package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rogersnm/compass/internal/dag"
	"github.com/rogersnm/compass/internal/id"
	"github.com/rogersnm/compass/internal/model"
)

type TaskCreateOpts struct {
	Type      model.TaskType
	Epic      string
	Priority  *int
	DependsOn []string
	Body      string
}

type TaskFilter struct {
	ProjectID string
	EpicID    string
	Status    model.Status
	Type      model.TaskType
}

type TaskUpdate struct {
	Title     *string
	Status    *model.Status
	Priority  **int
	DependsOn *[]string
	Body      *string
}

func (s *Store) CreateTask(title, projectID string, opts TaskCreateOpts) (*model.Task, error) {
	if _, _, err := s.GetProject(projectID); err != nil {
		return nil, fmt.Errorf("project %s not found", projectID)
	}

	taskType := opts.Type
	if taskType == "" {
		taskType = model.TypeTask
	}

	if opts.Epic != "" {
		epic, _, err := s.GetTask(opts.Epic)
		if err != nil {
			return nil, fmt.Errorf("epic %s not found", opts.Epic)
		}
		if epic.Type != model.TypeEpic {
			return nil, fmt.Errorf("%s is not an epic-type task", opts.Epic)
		}
	}

	tid, err := id.NewTaskID(projectID)
	if err != nil {
		return nil, err
	}

	t := &model.Task{
		ID:        tid,
		Title:     title,
		Type:      taskType,
		Project:   projectID,
		Epic:      opts.Epic,
		Status:    model.StatusOpen,
		Priority:  opts.Priority,
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
		files, err := s.ListFiles(taskDir, "*.md")
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
			if filter.Type != "" && t.Type != filter.Type {
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

	if upd.Title != nil {
		t.Title = *upd.Title
	}
	if upd.Status != nil {
		t.Status = *upd.Status
	}
	if upd.Priority != nil {
		t.Priority = *upd.Priority
	}
	if upd.DependsOn != nil {
		t.DependsOn = *upd.DependsOn
	}
	if upd.Body != nil {
		body = *upd.Body
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

func (s *Store) DeleteTask(taskID string) error {
	path, err := s.ResolveEntityPath(taskID)
	if err != nil {
		return err
	}
	return os.Remove(path)
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

// ReadyTasks returns open, unblocked tasks (type=task only) in topological order.
func (s *Store) ReadyTasks(projectID string) ([]*model.Task, error) {
	tasks, err := s.ListTasks(TaskFilter{ProjectID: projectID, Type: model.TypeTask})
	if err != nil {
		return nil, err
	}

	// Build task map for blocked check
	allTasks, err := s.AllTaskMap(projectID)
	if err != nil {
		return nil, err
	}

	// Filter to open, unblocked
	var candidates []*model.Task
	candidateSet := make(map[string]bool)
	for i := range tasks {
		t := &tasks[i]
		if t.Status == model.StatusOpen && !t.IsBlocked(allTasks) {
			candidates = append(candidates, t)
			candidateSet[t.ID] = true
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// Sort by topological order for determinism
	ptrs := make([]*model.Task, 0, len(tasks))
	for i := range tasks {
		ptrs = append(ptrs, &tasks[i])
	}
	g := dag.BuildFromTasks(ptrs)
	order, err := g.TopologicalSort()
	if err != nil {
		return candidates, nil // fallback to unsorted
	}

	var result []*model.Task
	for _, id := range order {
		if candidateSet[id] {
			result = append(result, allTasks[id])
		}
	}
	return result, nil
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
		if dt.Type == model.TypeEpic {
			return fmt.Errorf("cannot depend on epic-type task %s", dep)
		}
	}

	existing, err := s.ListTasks(TaskFilter{ProjectID: projectID, Type: model.TypeTask})
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

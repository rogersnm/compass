package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rogersnm/compass/internal/model"
)

// CloudStore implements Store using the compass-cloud HTTP API.
type CloudStore struct {
	apiURL string
	apiKey string
	client *http.Client
}

// compile-time check
var _ Store = (*CloudStore)(nil)

func NewCloudStore(apiURL, apiKey string) *CloudStore {
	return &CloudStore{
		apiURL: apiURL,
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// --- HTTP helpers ---

func (cs *CloudStore) doJSON(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, cs.apiURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cs.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return cs.client.Do(req)
}

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeResponse[T any](resp *http.Response) (T, error) {
	defer resp.Body.Close()
	var zero T

	if resp.StatusCode >= 400 {
		var apiErr apiError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error.Message != "" {
			return zero, fmt.Errorf("API error %d: %s", resp.StatusCode, apiErr.Error.Message)
		}
		return zero, fmt.Errorf("API error %d", resp.StatusCode)
	}

	var wrapper struct {
		Data T `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return zero, fmt.Errorf("decoding response: %w", err)
	}
	return wrapper.Data, nil
}

// --- API response types ---

type apiProject struct {
	ProjectID string     `json:"project_id"`
	Key       string     `json:"key"`
	Name      string     `json:"name"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

func (p *apiProject) toModel() *model.Project {
	return &model.Project{
		ID:        p.Key,
		Name:      p.Name,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.CreatedAt,
	}
}

type apiTask struct {
	TaskID    string     `json:"task_id"`
	DisplayID string     `json:"display_id"`
	Title     string     `json:"title"`
	Type      string     `json:"type"`
	Status    string     `json:"status"`
	Priority  *int       `json:"priority"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

func (t *apiTask) toModel() *model.Task {
	return &model.Task{
		ID:        t.DisplayID,
		Title:     t.Title,
		Type:      model.TaskType(t.Type),
		Status:    model.Status(t.Status),
		Priority:  t.Priority,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.CreatedAt,
	}
}

type apiDocument struct {
	DocumentID string     `json:"document_id"`
	DisplayID  string     `json:"display_id"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	CreatedAt  time.Time  `json:"created_at"`
	DeletedAt  *time.Time `json:"deleted_at"`
}

func (d *apiDocument) toModel() *model.Document {
	return &model.Document{
		ID:        d.DisplayID,
		Title:     d.Title,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.CreatedAt,
	}
}

type apiSearchResult struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

// --- Projects ---

func (cs *CloudStore) CreateProject(name, key, body string) (*model.Project, error) {
	payload := map[string]string{"name": name}
	if key != "" {
		payload["key"] = key
	}
	if body != "" {
		payload["body"] = body
	}
	resp, err := cs.doJSON("POST", "/api/v1/projects", payload)
	if err != nil {
		return nil, err
	}
	ap, err := decodeResponse[apiProject](resp)
	if err != nil {
		return nil, err
	}
	return ap.toModel(), nil
}

func (cs *CloudStore) GetProject(projectID string) (*model.Project, string, error) {
	resp, err := cs.doJSON("GET", "/api/v1/projects/"+url.PathEscape(projectID), nil)
	if err != nil {
		return nil, "", err
	}
	ap, err := decodeResponse[apiProject](resp)
	if err != nil {
		return nil, "", err
	}
	return ap.toModel(), ap.Body, nil
}

func (cs *CloudStore) ListProjects() ([]model.Project, error) {
	var all []model.Project
	cursor := ""
	for {
		path := "/api/v1/projects?limit=100"
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		resp, err := cs.doJSON("GET", path, nil)
		if err != nil {
			return nil, err
		}
		page, err := decodePagedResponse[apiProject](resp)
		if err != nil {
			return nil, err
		}
		for _, ap := range page.data {
			all = append(all, *ap.toModel())
		}
		if page.nextCursor == "" {
			break
		}
		cursor = page.nextCursor
	}
	return all, nil
}

func (cs *CloudStore) DeleteProject(projectID string) error {
	resp, err := cs.doJSON("DELETE", "/api/v1/projects/"+url.PathEscape(projectID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d deleting project", resp.StatusCode)
	}
	return nil
}

// --- Tasks ---

func (cs *CloudStore) CreateTask(title, projectID string, opts TaskCreateOpts) (*model.Task, error) {
	payload := map[string]any{
		"title": title,
	}
	if opts.Type != "" {
		payload["type"] = string(opts.Type)
	}
	if opts.Priority != nil {
		payload["priority"] = *opts.Priority
	}
	if opts.Epic != "" {
		payload["epic_task_id"] = opts.Epic
	}
	if opts.Body != "" {
		payload["body"] = opts.Body
	}

	resp, err := cs.doJSON("POST", "/api/v1/projects/"+url.PathEscape(projectID)+"/tasks", payload)
	if err != nil {
		return nil, err
	}
	at, err := decodeResponse[apiTask](resp)
	if err != nil {
		return nil, err
	}
	t := at.toModel()
	t.Project = projectID
	t.Epic = opts.Epic
	t.DependsOn = opts.DependsOn
	return t, nil
}

func (cs *CloudStore) GetTask(taskID string) (*model.Task, string, error) {
	resp, err := cs.doJSON("GET", "/api/v1/tasks/"+url.PathEscape(taskID), nil)
	if err != nil {
		return nil, "", err
	}
	at, err := decodeResponse[apiTask](resp)
	if err != nil {
		return nil, "", err
	}
	return at.toModel(), at.Body, nil
}

func (cs *CloudStore) ListTasks(filter TaskFilter) ([]model.Task, error) {
	if filter.ProjectID == "" {
		// List across all projects: list projects first, then tasks per project
		projects, err := cs.ListProjects()
		if err != nil {
			return nil, err
		}
		var all []model.Task
		for _, p := range projects {
			f := filter
			f.ProjectID = p.ID
			tasks, err := cs.ListTasks(f)
			if err != nil {
				continue
			}
			all = append(all, tasks...)
		}
		return all, nil
	}

	var all []model.Task
	cursor := ""
	for {
		path := "/api/v1/projects/" + url.PathEscape(filter.ProjectID) + "/tasks?limit=100"
		if filter.Status != "" {
			path += "&status=" + url.QueryEscape(string(filter.Status))
		}
		if filter.Type != "" {
			path += "&type=" + url.QueryEscape(string(filter.Type))
		}
		if filter.EpicID != "" {
			path += "&epic=" + url.QueryEscape(filter.EpicID)
		}
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		resp, err := cs.doJSON("GET", path, nil)
		if err != nil {
			return nil, err
		}
		page, err := decodePagedResponse[apiTask](resp)
		if err != nil {
			return nil, err
		}
		for _, at := range page.data {
			t := *at.toModel()
			t.Project = filter.ProjectID
			all = append(all, t)
		}
		if page.nextCursor == "" {
			break
		}
		cursor = page.nextCursor
	}
	return all, nil
}

func (cs *CloudStore) UpdateTask(taskID string, upd TaskUpdate) (*model.Task, error) {
	payload := map[string]any{}
	if upd.Title != nil {
		payload["title"] = *upd.Title
	}
	if upd.Status != nil {
		payload["status"] = string(*upd.Status)
	}
	if upd.Priority != nil {
		payload["priority"] = *upd.Priority // can be nil to clear
	}
	if upd.Body != nil {
		payload["body"] = *upd.Body
	}

	resp, err := cs.doJSON("PATCH", "/api/v1/tasks/"+url.PathEscape(taskID), payload)
	if err != nil {
		return nil, err
	}
	at, err := decodeResponse[apiTask](resp)
	if err != nil {
		return nil, err
	}
	return at.toModel(), nil
}

func (cs *CloudStore) DeleteTask(taskID string) error {
	resp, err := cs.doJSON("DELETE", "/api/v1/tasks/"+url.PathEscape(taskID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d deleting task", resp.StatusCode)
	}
	return nil
}

func (cs *CloudStore) AllTaskMap(projectID string) (map[string]*model.Task, error) {
	tasks, err := cs.ListTasks(TaskFilter{ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	m := make(map[string]*model.Task, len(tasks))
	for i := range tasks {
		m[tasks[i].ID] = &tasks[i]
	}
	return m, nil
}

func (cs *CloudStore) ReadyTasks(projectID string) ([]*model.Task, error) {
	// Use the dedicated ready endpoint
	resp, err := cs.doJSON("GET", "/api/v1/projects/"+url.PathEscape(projectID)+"/tasks/ready", nil)
	if err != nil {
		return nil, err
	}
	items, err := decodeResponse[[]apiTask](resp)
	if err != nil {
		return nil, err
	}
	var result []*model.Task
	for _, at := range items {
		t := at.toModel()
		t.Project = projectID
		result = append(result, t)
	}
	return result, nil
}

// --- Documents ---

func (cs *CloudStore) CreateDocument(title, projectID, body string) (*model.Document, error) {
	payload := map[string]string{"title": title}
	if body != "" {
		payload["body"] = body
	}
	resp, err := cs.doJSON("POST", "/api/v1/projects/"+url.PathEscape(projectID)+"/documents", payload)
	if err != nil {
		return nil, err
	}
	ad, err := decodeResponse[apiDocument](resp)
	if err != nil {
		return nil, err
	}
	d := ad.toModel()
	d.Project = projectID
	return d, nil
}

func (cs *CloudStore) GetDocument(docID string) (*model.Document, string, error) {
	resp, err := cs.doJSON("GET", "/api/v1/documents/"+url.PathEscape(docID), nil)
	if err != nil {
		return nil, "", err
	}
	ad, err := decodeResponse[apiDocument](resp)
	if err != nil {
		return nil, "", err
	}
	return ad.toModel(), ad.Body, nil
}

func (cs *CloudStore) ListDocuments(projectID string) ([]model.Document, error) {
	if projectID == "" {
		projects, err := cs.ListProjects()
		if err != nil {
			return nil, err
		}
		var all []model.Document
		for _, p := range projects {
			docs, err := cs.ListDocuments(p.ID)
			if err != nil {
				continue
			}
			all = append(all, docs...)
		}
		return all, nil
	}

	var all []model.Document
	cursor := ""
	for {
		path := "/api/v1/projects/" + url.PathEscape(projectID) + "/documents?limit=100"
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		resp, err := cs.doJSON("GET", path, nil)
		if err != nil {
			return nil, err
		}
		page, err := decodePagedResponse[apiDocument](resp)
		if err != nil {
			return nil, err
		}
		for _, ad := range page.data {
			d := *ad.toModel()
			d.Project = projectID
			all = append(all, d)
		}
		if page.nextCursor == "" {
			break
		}
		cursor = page.nextCursor
	}
	return all, nil
}

func (cs *CloudStore) UpdateDocument(docID string, title, body *string) (*model.Document, error) {
	payload := map[string]any{}
	if title != nil {
		payload["title"] = *title
	}
	if body != nil {
		payload["body"] = *body
	}
	resp, err := cs.doJSON("PATCH", "/api/v1/documents/"+url.PathEscape(docID), payload)
	if err != nil {
		return nil, err
	}
	ad, err := decodeResponse[apiDocument](resp)
	if err != nil {
		return nil, err
	}
	return ad.toModel(), nil
}

func (cs *CloudStore) DeleteDocument(docID string) error {
	resp, err := cs.doJSON("DELETE", "/api/v1/documents/"+url.PathEscape(docID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d deleting document", resp.StatusCode)
	}
	return nil
}

// --- Search ---

func (cs *CloudStore) Search(query, projectID string) ([]SearchResult, error) {
	path := "/api/v1/search?q=" + url.QueryEscape(query)
	if projectID != "" {
		path += "&project=" + url.QueryEscape(projectID)
	}
	resp, err := cs.doJSON("GET", path, nil)
	if err != nil {
		return nil, err
	}
	items, err := decodeResponse[[]apiSearchResult](resp)
	if err != nil {
		return nil, err
	}
	var results []SearchResult
	for _, item := range items {
		results = append(results, SearchResult{
			Type:    item.Type,
			ID:      item.ID,
			Title:   item.Title,
			Snippet: item.Snippet,
		})
	}
	return results, nil
}

// --- Entity operations (local-only, not applicable to cloud) ---

func (cs *CloudStore) ResolveEntityPath(entityID string) (string, error) {
	return "", fmt.Errorf("ResolveEntityPath not supported in cloud mode")
}

func (cs *CloudStore) CheckoutEntity(entityID, destDir string) (string, error) {
	return "", fmt.Errorf("checkout not supported in cloud mode")
}

func (cs *CloudStore) CheckinTask(localPath string) (*model.Task, error) {
	return nil, fmt.Errorf("checkin not supported in cloud mode")
}

func (cs *CloudStore) CheckinDocument(localPath string) (*model.Document, error) {
	return nil, fmt.Errorf("checkin not supported in cloud mode")
}

func (cs *CloudStore) WriteEntity(path string, meta any, body string) error {
	return fmt.Errorf("WriteEntity not supported in cloud mode")
}

// --- Pagination helper ---

type pagedResult[T any] struct {
	data       []T
	nextCursor string
}

func decodePagedResponse[T any](resp *http.Response) (pagedResult[T], error) {
	defer resp.Body.Close()
	var zero pagedResult[T]

	if resp.StatusCode >= 400 {
		var apiErr apiError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error.Message != "" {
			return zero, fmt.Errorf("API error %d: %s", resp.StatusCode, apiErr.Error.Message)
		}
		return zero, fmt.Errorf("API error %d", resp.StatusCode)
	}

	var wrapper struct {
		Data       []T    `json:"data"`
		NextCursor string `json:"next_cursor"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return zero, fmt.Errorf("decoding response: %w", err)
	}
	return pagedResult[T]{data: wrapper.Data, nextCursor: wrapper.NextCursor}, nil
}

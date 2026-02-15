package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/rogersnm/compass/internal/config"
	"github.com/rogersnm/compass/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAPI is a minimal in-memory compass-cloud API server for cmd-level tests.
type fakeAPI struct {
	mu        sync.Mutex
	projects  map[string]map[string]any
	tasks     map[string]map[string]any
	documents map[string]map[string]any
	taskSeq   int
	docSeq    int
}

func newFakeAPI() *fakeAPI {
	return &fakeAPI{
		projects:  make(map[string]map[string]any),
		tasks:     make(map[string]map[string]any),
		documents: make(map[string]map[string]any),
	}
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	path := r.URL.Path
	// Strip /api/v1 prefix if present
	if hasPrefix(path, "/api/v1") {
		path = path[len("/api/v1"):]
	}

	const (
		pfxProjects  = "/projects/"
		pfxTasks     = "/tasks/"
		pfxDocuments = "/documents/"
	)

	switch {
	// Search
	case r.Method == "GET" && path == "/search":
		f.handleSearch(w, r)

	// Projects (exact path, no trailing segments)
	case r.Method == "POST" && path == "/projects":
		f.handleCreateProject(w, r)
	case r.Method == "GET" && path == "/projects":
		f.handleListProjects(w)

	// Project sub-resources: /projects/{id}/tasks/ready
	case r.Method == "GET" && hasPrefix(path, pfxProjects) && matchSuffix(path, "/tasks/ready"):
		projID := extractProjectID(path, pfxProjects, "/tasks/ready")
		f.handleReadyTasks(w, projID)
	// Project sub-resources: /projects/{id}/tasks
	case r.Method == "POST" && hasPrefix(path, pfxProjects) && matchSuffix(path, "/tasks"):
		projID := extractProjectID(path, pfxProjects, "/tasks")
		f.handleCreateTask(w, r, projID)
	case r.Method == "GET" && hasPrefix(path, pfxProjects) && matchSuffix(path, "/tasks"):
		projID := extractProjectID(path, pfxProjects, "/tasks")
		f.handleListTasks(w, projID)
	// Project sub-resources: /projects/{id}/documents
	case r.Method == "POST" && hasPrefix(path, pfxProjects) && matchSuffix(path, "/documents"):
		projID := extractProjectID(path, pfxProjects, "/documents")
		f.handleCreateDocument(w, r, projID)
	case r.Method == "GET" && hasPrefix(path, pfxProjects) && matchSuffix(path, "/documents"):
		projID := extractProjectID(path, pfxProjects, "/documents")
		f.handleListDocuments(w, projID)

	// Single project: /projects/{id}
	case r.Method == "GET" && hasPrefix(path, pfxProjects) && !contains(path[len(pfxProjects):], "/"):
		f.handleGetProject(w, path[len(pfxProjects):])
	case r.Method == "DELETE" && hasPrefix(path, pfxProjects) && !contains(path[len(pfxProjects):], "/"):
		f.handleDeleteProject(w, path[len(pfxProjects):])

	// Single task: /tasks/{id}
	case r.Method == "GET" && hasPrefix(path, pfxTasks):
		f.handleGetTask(w, path[len(pfxTasks):])
	case r.Method == "PATCH" && hasPrefix(path, pfxTasks):
		f.handleUpdateTask(w, r, path[len(pfxTasks):])
	case r.Method == "DELETE" && hasPrefix(path, pfxTasks):
		f.handleDeleteTask(w, path[len(pfxTasks):])

	// Single document: /documents/{id}
	case r.Method == "GET" && hasPrefix(path, pfxDocuments):
		f.handleGetDocument(w, path[len(pfxDocuments):])
	case r.Method == "PATCH" && hasPrefix(path, pfxDocuments):
		f.handleUpdateDocument(w, r, path[len(pfxDocuments):])
	case r.Method == "DELETE" && hasPrefix(path, pfxDocuments):
		f.handleDeleteDocument(w, path[len(pfxDocuments):])

	default:
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "NOT_FOUND", "message": "Not found"},
		})
	}
}

func (f *fakeAPI) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	json.NewDecoder(r.Body).Decode(&body)
	key := body["key"]
	if key == "" {
		name := body["name"]
		if len(name) >= 4 {
			key = name[:4]
		} else {
			key = name
		}
	}
	p := map[string]any{
		"project_id": "uuid-" + key,
		"key":        key,
		"name":       body["name"],
		"body":       body["body"],
		"created_at": "2026-01-01T00:00:00Z",
	}
	f.projects[key] = p
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]any{"data": p})
}

func (f *fakeAPI) handleGetProject(w http.ResponseWriter, key string) {
	p, ok := f.projects[key]
	if !ok {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "NOT_FOUND", "message": "Project not found"},
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"data": p})
}

func (f *fakeAPI) handleListProjects(w http.ResponseWriter) {
	var list []map[string]any
	for _, p := range f.projects {
		list = append(list, p)
	}
	if list == nil {
		list = []map[string]any{}
	}
	json.NewEncoder(w).Encode(map[string]any{"data": list})
}

func (f *fakeAPI) handleDeleteProject(w http.ResponseWriter, key string) {
	delete(f.projects, key)
	json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message": "deleted"}})
}

// Hash chars from charset: 23456789ABCDEFGHJKMNPQRSTUVWXYZ
var taskHashSuffixes = []string{"ABCDE", "BCDEF", "CDEFG", "DEFGH", "EFGHJ", "FGHJK", "GHJKM", "HJKMN", "JKMNP"}

func (f *fakeAPI) handleCreateTask(w http.ResponseWriter, r *http.Request, projID string) {
	var body map[string]any
	json.NewDecoder(r.Body).Decode(&body)
	f.taskSeq++
	hash := taskHashSuffixes[f.taskSeq%len(taskHashSuffixes)]
	displayID := projID + "-T" + hash
	taskType := "task"
	if t, ok := body["type"].(string); ok && t != "" {
		taskType = t
	}
	t := map[string]any{
		"task_id":    "uuid-task-" + displayID,
		"key":        displayID,
		"title":      body["title"],
		"type":       taskType,
		"status":     "open",
		"body":       body["body"],
		"priority":   body["priority"],
		"project_key": projID,
		"project":    projID,
		"created_at": "2026-01-01T00:00:00Z",
	}
	f.tasks[displayID] = t
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]any{"data": t})
}

func (f *fakeAPI) handleGetTask(w http.ResponseWriter, taskID string) {
	t, ok := f.tasks[taskID]
	if !ok {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "NOT_FOUND", "message": "Task not found"},
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"data": t})
}

func (f *fakeAPI) handleListTasks(w http.ResponseWriter, projID string) {
	var list []map[string]any
	for _, t := range f.tasks {
		if t["project"] == projID {
			list = append(list, t)
		}
	}
	if list == nil {
		list = []map[string]any{}
	}
	json.NewEncoder(w).Encode(map[string]any{"data": list})
}

func (f *fakeAPI) handleUpdateTask(w http.ResponseWriter, r *http.Request, taskID string) {
	t, ok := f.tasks[taskID]
	if !ok {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "NOT_FOUND", "message": "Task not found"},
		})
		return
	}
	var body map[string]any
	json.NewDecoder(r.Body).Decode(&body)
	if v, ok := body["title"]; ok {
		t["title"] = v
	}
	if v, ok := body["status"]; ok {
		t["status"] = v
	}
	if v, ok := body["body"]; ok {
		t["body"] = v
	}
	if v, ok := body["priority"]; ok {
		t["priority"] = v
	}
	f.tasks[taskID] = t
	json.NewEncoder(w).Encode(map[string]any{"data": t})
}

func (f *fakeAPI) handleDeleteTask(w http.ResponseWriter, taskID string) {
	delete(f.tasks, taskID)
	json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message": "deleted"}})
}

func (f *fakeAPI) handleReadyTasks(w http.ResponseWriter, projID string) {
	var list []map[string]any
	for _, t := range f.tasks {
		if t["project"] == projID && t["status"] == "open" {
			list = append(list, t)
		}
	}
	if list == nil {
		list = []map[string]any{}
	}
	json.NewEncoder(w).Encode(map[string]any{"data": list})
}

func (f *fakeAPI) handleCreateDocument(w http.ResponseWriter, r *http.Request, projID string) {
	var body map[string]string
	json.NewDecoder(r.Body).Decode(&body)
	f.docSeq++
	hash := taskHashSuffixes[f.docSeq%len(taskHashSuffixes)]
	displayID := projID + "-D" + hash
	d := map[string]any{
		"document_id": "uuid-doc-" + displayID,
		"key":         displayID,
		"title":       body["title"],
		"body":        body["body"],
		"project":     projID,
		"created_at":  "2026-01-01T00:00:00Z",
	}
	f.documents[displayID] = d
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]any{"data": d})
}

func (f *fakeAPI) handleGetDocument(w http.ResponseWriter, docID string) {
	d, ok := f.documents[docID]
	if !ok {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "NOT_FOUND", "message": "Document not found"},
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"data": d})
}

func (f *fakeAPI) handleListDocuments(w http.ResponseWriter, projID string) {
	var list []map[string]any
	for _, d := range f.documents {
		if d["project"] == projID {
			list = append(list, d)
		}
	}
	if list == nil {
		list = []map[string]any{}
	}
	json.NewEncoder(w).Encode(map[string]any{"data": list})
}

func (f *fakeAPI) handleUpdateDocument(w http.ResponseWriter, r *http.Request, docID string) {
	d, ok := f.documents[docID]
	if !ok {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "NOT_FOUND", "message": "Document not found"},
		})
		return
	}
	var body map[string]any
	json.NewDecoder(r.Body).Decode(&body)
	if v, ok := body["title"]; ok {
		d["title"] = v
	}
	if v, ok := body["body"]; ok {
		d["body"] = v
	}
	f.documents[docID] = d
	json.NewEncoder(w).Encode(map[string]any{"data": d})
}

func (f *fakeAPI) handleDeleteDocument(w http.ResponseWriter, docID string) {
	delete(f.documents, docID)
	json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message": "deleted"}})
}

func (f *fakeAPI) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var results []map[string]any
	for id, t := range f.tasks {
		if contains(t["title"].(string), q) {
			results = append(results, map[string]any{
				"type": "task", "id": id, "title": t["title"], "snippet": "...",
			})
		}
	}
	for id, d := range f.documents {
		if contains(d["title"].(string), q) {
			results = append(results, map[string]any{
				"type": "document", "id": id, "title": d["title"], "snippet": "...",
			})
		}
	}
	if results == nil {
		results = []map[string]any{}
	}
	json.NewEncoder(w).Encode(map[string]any{"data": results})
}

// helpers

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func matchSuffix(path, suffix string) bool {
	return len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix
}

func extractProjectID(path, prefix, suffix string) string {
	after := path[len(prefix):]
	return after[:len(after)-len(suffix)]
}

// setupCloudEnv starts a fake API server and prepares global state with a
// registry containing the cloud store.
func setupCloudEnv(t *testing.T) *fakeAPI {
	t.Helper()
	api := newFakeAPI()
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	dataDir = dir

	// Parse the test server URL so PersistentPreRunE can rebuild correctly
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	hostname := u.Host // e.g. "127.0.0.1:12345"

	cfg = &config.Config{
		Version:      2,
		DefaultStore: hostname,
		Stores: map[string]config.CloudStoreConfig{
			hostname: {
				Hostname: hostname,
				APIKey:   "test-key",
				Protocol: u.Scheme, // "http"
				// Path defaults to "/api/v1"
			},
		},
		Projects: map[string]string{},
	}
	require.NoError(t, config.Save(dir, cfg))

	sc := cfg.Stores[hostname]
	reg = store.NewRegistry(cfg, dir)
	reg.Add(hostname, store.NewCloudStoreWithBase(sc.URL(), sc.APIKey))

	return api
}

// seedProject adds a project to both the fake API and the registry cache.
func seedProject(api *fakeAPI, key string) {
	api.projects[key] = map[string]any{
		"project_id": "uuid-" + key, "key": key, "name": "Cloud Project",
		"body": "", "created_at": "2026-01-01T00:00:00Z",
	}
	reg.CacheProject(key, cfg.DefaultStore)
}

// seedTask adds a task with a valid hash ID to the fake API.
func seedTask(api *fakeAPI, projID, hash, title string) string {
	taskID := projID + "-T" + hash
	api.tasks[taskID] = map[string]any{
		"task_id": "uuid-task-" + taskID, "key": taskID, "title": title,
		"type": "task", "status": "open", "body": "", "project": projID,
		"project_key": projID, "created_at": "2026-01-01T00:00:00Z",
	}
	return taskID
}

// seedDoc adds a document with a valid hash ID to the fake API.
func seedDoc(api *fakeAPI, projID, hash, title string) string {
	docID := projID + "-D" + hash
	api.documents[docID] = map[string]any{
		"document_id": "uuid-doc-" + docID, "key": docID, "title": title,
		"body": "doc body", "project": projID, "created_at": "2026-01-01T00:00:00Z",
	}
	return docID
}

// --- Store routing tests ---

func TestCloudRouting_PromptWhenNoConfig(t *testing.T) {
	dir := t.TempDir()
	dataDir = dir
	cfg = &config.Config{}

	require.NoError(t, config.Save(dir, cfg))

	// Non-interactive stdin falls through to default error
	err := run(t, "project", "list")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compass store add")
}

func TestCloudRouting_LocalMode(t *testing.T) {
	dir := t.TempDir()
	dataDir = dir
	cfg = &config.Config{Mode: "local"}

	require.NoError(t, config.Save(dir, cfg))

	// Should work fine with local store after migration
	err := run(t, "project", "list")
	assert.NoError(t, err)
}

// --- Config command tests ---

func TestConfig_StatusLocal(t *testing.T) {
	dir := t.TempDir()
	dataDir = dir
	cfg = &config.Config{
		Version: 2, LocalEnabled: true, DefaultStore: "local",
		Projects: map[string]string{},
	}
	require.NoError(t, config.Save(dir, cfg))
	reg = store.NewRegistry(cfg, dir)
	reg.Add("local", store.NewLocal(dir))

	require.NoError(t, run(t, "config", "status"))
}

func TestConfig_StatusCloud(t *testing.T) {
	setupCloudEnv(t)

	require.NoError(t, run(t, "config", "status"))
}

func TestConfig_StatusUnconfigured(t *testing.T) {
	dir := t.TempDir()
	dataDir = dir
	cfg = &config.Config{Version: 2, Projects: map[string]string{}}
	require.NoError(t, config.Save(dir, cfg))
	reg = store.NewRegistry(cfg, dir)

	require.NoError(t, run(t, "config", "status"))
}

func TestConfig_Logout(t *testing.T) {
	setupCloudEnv(t)

	require.NoError(t, run(t, "config", "logout"))

	c, err := config.Load(dataDir)
	require.NoError(t, err)
	_, hasCloud := c.Stores["compasscloud.io"]
	assert.False(t, hasCloud)
}

func TestConfig_LogoutWhenLocal(t *testing.T) {
	dir := t.TempDir()
	dataDir = dir
	cfg = &config.Config{Version: 2, LocalEnabled: true, DefaultStore: "local", Projects: map[string]string{}}
	require.NoError(t, config.Save(dir, cfg))
	reg = store.NewRegistry(cfg, dir)
	reg.Add("local", store.NewLocal(dir))

	require.NoError(t, run(t, "config", "logout"))
}

func TestConfig_BypassesSetupPrompt(t *testing.T) {
	dir := t.TempDir()
	dataDir = dir
	cfg = &config.Config{Version: 2, Projects: map[string]string{}}
	require.NoError(t, config.Save(dir, cfg))
	reg = store.NewRegistry(cfg, dir)

	require.NoError(t, run(t, "config", "status"))
	require.NoError(t, run(t, "config", "logout"))
}

// --- Cloud mode project tests ---

func TestCloud_ProjectCreate(t *testing.T) {
	setupCloudEnv(t)
	require.NoError(t, run(t, "project", "create", "Cloud Project", "--key", "CP"))
}

func TestCloud_ProjectList(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	api.mu.Unlock()

	require.NoError(t, run(t, "project", "list"))
}

func TestCloud_ProjectShow(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	api.mu.Unlock()

	require.NoError(t, run(t, "project", "show", "CP"))
}

func TestCloud_ProjectShow_NotFound(t *testing.T) {
	setupCloudEnv(t)
	assert.Error(t, run(t, "project", "show", "ZZZZ"))
}

func TestCloud_ProjectDelete(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	api.mu.Unlock()

	require.NoError(t, run(t, "project", "delete", "CP", "--force"))

	api.mu.Lock()
	_, exists := api.projects["CP"]
	api.mu.Unlock()
	assert.False(t, exists)
}

func TestCloud_ProjectSetStore(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	api.mu.Unlock()

	require.NoError(t, run(t, "project", "set-store", "CP", cfg.DefaultStore))

	c, err := config.Load(dataDir)
	require.NoError(t, err)
	assert.Equal(t, cfg.DefaultStore, c.Projects["CP"])
}

// --- Cloud mode task tests ---

func TestCloud_TaskCreate(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	api.mu.Unlock()

	require.NoError(t, run(t, "task", "create", "Cloud Task", "--project", "CP"))

	api.mu.Lock()
	assert.Len(t, api.tasks, 1)
	api.mu.Unlock()
}

func TestCloud_TaskCreate_Epic(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	api.mu.Unlock()

	require.NoError(t, run(t, "task", "create", "Cloud Epic", "--project", "CP", "--type", "epic"))

	api.mu.Lock()
	assert.Len(t, api.tasks, 1)
	for _, task := range api.tasks {
		assert.Equal(t, "epic", task["type"])
	}
	api.mu.Unlock()
}

func TestCloud_TaskList(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	seedTask(api, "CP", "ABCDE", "Task 1")
	api.mu.Unlock()

	require.NoError(t, run(t, "task", "list", "--project", "CP"))
}

func TestCloud_TaskShow(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	taskID := seedTask(api, "CP", "ABCDE", "Task 1")
	api.mu.Unlock()

	require.NoError(t, run(t, "task", "show", taskID))
}

func TestCloud_TaskStart(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	taskID := seedTask(api, "CP", "ABCDE", "Task 1")
	api.mu.Unlock()

	require.NoError(t, run(t, "task", "start", taskID))

	api.mu.Lock()
	assert.Equal(t, "in_progress", api.tasks[taskID]["status"])
	api.mu.Unlock()
}

func TestCloud_TaskClose(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	taskID := seedTask(api, "CP", "ABCDE", "Task 1")
	api.mu.Unlock()

	require.NoError(t, run(t, "task", "close", taskID))

	api.mu.Lock()
	assert.Equal(t, "closed", api.tasks[taskID]["status"])
	api.mu.Unlock()
}

func TestCloud_TaskDelete(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	taskID := seedTask(api, "CP", "ABCDE", "Task 1")
	api.mu.Unlock()

	require.NoError(t, run(t, "task", "delete", taskID, "--force"))

	api.mu.Lock()
	assert.Len(t, api.tasks, 0)
	api.mu.Unlock()
}

func TestCloud_TaskReady(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	seedTask(api, "CP", "ABCDE", "Ready Task")
	api.mu.Unlock()

	require.NoError(t, run(t, "task", "ready", "--project", "CP"))
}

// --- Cloud mode document tests ---

func TestCloud_DocCreate(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	api.mu.Unlock()

	require.NoError(t, run(t, "doc", "create", "Cloud Doc", "--project", "CP"))

	api.mu.Lock()
	assert.Len(t, api.documents, 1)
	api.mu.Unlock()
}

func TestCloud_DocShow(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	docID := seedDoc(api, "CP", "ABCDE", "My Doc")
	api.mu.Unlock()

	require.NoError(t, run(t, "doc", "show", docID))
}

func TestCloud_DocDelete(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	docID := seedDoc(api, "CP", "ABCDE", "My Doc")
	api.mu.Unlock()

	require.NoError(t, run(t, "doc", "delete", docID, "--force"))

	api.mu.Lock()
	assert.Len(t, api.documents, 0)
	api.mu.Unlock()
}

// --- Cloud mode download/upload tests ---

func TestCloud_TaskDownload(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	taskID := seedTask(api, "CP", "ABCDE", "Download Task")
	api.tasks[taskID]["body"] = "task body"
	api.mu.Unlock()

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	require.NoError(t, run(t, "task", "download", taskID))
	assert.FileExists(t, filepath.Join(".compass", taskID+".md"))
}

func TestCloud_TaskUpload(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	taskID := seedTask(api, "CP", "ABCDE", "Upload Task")
	api.tasks[taskID]["body"] = "original body"
	api.mu.Unlock()

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	require.NoError(t, run(t, "task", "download", taskID))

	localPath := filepath.Join(".compass", taskID+".md")
	assert.FileExists(t, localPath)

	require.NoError(t, run(t, "task", "upload", taskID))
	assert.NoFileExists(t, localPath)
}

func TestCloud_DocDownload(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	docID := seedDoc(api, "CP", "ABCDE", "Download Doc")
	api.mu.Unlock()

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	require.NoError(t, run(t, "doc", "download", docID))
	assert.FileExists(t, filepath.Join(".compass", docID+".md"))
}

func TestCloud_DocUpload(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	docID := seedDoc(api, "CP", "ABCDE", "Upload Doc")
	api.mu.Unlock()

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	require.NoError(t, run(t, "doc", "download", docID))

	localPath := filepath.Join(".compass", docID+".md")
	assert.FileExists(t, localPath)

	require.NoError(t, run(t, "doc", "upload", docID))
	assert.NoFileExists(t, localPath)
}

// --- Cloud mode search ---

func TestCloud_Search(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	seedTask(api, "CP", "ABCDE", "Auth Task")
	api.mu.Unlock()

	require.NoError(t, run(t, "search", "Auth"))
}

func TestCloud_Search_NoResults(t *testing.T) {
	setupCloudEnv(t)
	require.NoError(t, run(t, "search", "xyznonexistent"))
}

// --- Cloud mode priority tests ---

func TestCloud_TaskCreate_WithPriority(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	api.mu.Unlock()

	require.NoError(t, run(t, "task", "create", "Urgent", "--project", "CP", "--type", "task", "--priority", "1"))

	api.mu.Lock()
	require.Len(t, api.tasks, 1)
	for _, task := range api.tasks {
		assert.Equal(t, float64(1), task["priority"])
	}
	api.mu.Unlock()
}

func TestCloud_TaskCreate_NoPriority(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	api.mu.Unlock()

	require.NoError(t, run(t, "task", "create", "Normal", "--project", "CP", "--type", "task", "--priority", "-1"))

	api.mu.Lock()
	require.Len(t, api.tasks, 1)
	for _, task := range api.tasks {
		assert.Nil(t, task["priority"])
	}
	api.mu.Unlock()
}

func TestCloud_TaskUpdate_Priority(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	taskID := seedTask(api, "CP", "ABCDE", "Task 1")
	api.tasks[taskID]["priority"] = nil
	api.mu.Unlock()

	require.NoError(t, run(t, "task", "update", taskID, "--priority", "0"))

	api.mu.Lock()
	assert.Equal(t, float64(0), api.tasks[taskID]["priority"])
	api.mu.Unlock()
}

func TestCloud_TaskUpdate_Status(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	taskID := seedTask(api, "CP", "ABCDE", "Task 1")
	api.mu.Unlock()

	require.NoError(t, run(t, "task", "update", taskID, "--status", "in_progress"))

	api.mu.Lock()
	assert.Equal(t, "in_progress", api.tasks[taskID]["status"])
	api.mu.Unlock()
}

// --- Cloud mode task ready --all ---

func TestCloud_TaskReady_All(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	seedTask(api, "CP", "ABCDE", "T1")
	seedTask(api, "CP", "BCDEF", "T2")
	api.mu.Unlock()

	require.NoError(t, run(t, "task", "ready", "--project", "CP", "--all"))
}

// --- Cloud mode project delete clears default ---

func TestCloud_ProjectDelete_ClearsDefault(t *testing.T) {
	api := setupCloudEnv(t)
	api.mu.Lock()
	seedProject(api, "CP")
	api.mu.Unlock()

	cfg.DefaultProject = "CP"
	require.NoError(t, config.Save(dataDir, cfg))

	require.NoError(t, run(t, "project", "delete", "CP", "--force"))

	c, err := config.Load(dataDir)
	require.NoError(t, err)
	assert.Empty(t, c.DefaultProject)
}

// --- Cloud mode E2E workflow ---

func TestCloud_E2EWorkflow(t *testing.T) {
	api := setupCloudEnv(t)

	// 1. Create project via CLI
	require.NoError(t, run(t, "project", "create", "E2E Project", "--key", "EP"))

	// 2. Create docs via CLI
	require.NoError(t, run(t, "doc", "create", "Design Doc", "--project", "EP"))
	require.NoError(t, run(t, "doc", "create", "API Spec", "--project", "EP"))

	api.mu.Lock()
	assert.Len(t, api.documents, 2)
	api.mu.Unlock()

	// 3. Create epic via CLI
	require.NoError(t, run(t, "task", "create", "Auth Epic", "--project", "EP", "--type", "epic"))

	// 4. Create tasks via CLI
	require.NoError(t, run(t, "task", "create", "Task A", "--project", "EP", "--type", "task"))
	require.NoError(t, run(t, "task", "create", "Task B", "--project", "EP", "--type", "task"))

	api.mu.Lock()
	assert.Len(t, api.tasks, 3) // 1 epic + 2 tasks
	api.mu.Unlock()

	// 5. List tasks
	require.NoError(t, run(t, "task", "list", "--project", "EP"))

	// 6. Start a task
	api.mu.Lock()
	var taskAID string
	for id, task := range api.tasks {
		if task["title"] == "Task A" {
			taskAID = id
			break
		}
	}
	api.mu.Unlock()

	require.NoError(t, run(t, "task", "start", taskAID))

	api.mu.Lock()
	assert.Equal(t, "in_progress", api.tasks[taskAID]["status"])
	api.mu.Unlock()

	// 7. Close the task
	require.NoError(t, run(t, "task", "close", taskAID))

	api.mu.Lock()
	assert.Equal(t, "closed", api.tasks[taskAID]["status"])
	api.mu.Unlock()

	// 8. Ready tasks (should still have Task B)
	require.NoError(t, run(t, "task", "ready", "--project", "EP"))

	// 9. Search
	require.NoError(t, run(t, "search", "Auth"))

	// 10. Delete a task
	require.NoError(t, run(t, "task", "delete", taskAID, "--force"))

	api.mu.Lock()
	assert.Len(t, api.tasks, 2) // epic + Task B
	api.mu.Unlock()

	// 11. Download and upload a doc
	api.mu.Lock()
	var docID string
	for id := range api.documents {
		docID = id
		break
	}
	api.mu.Unlock()

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	require.NoError(t, run(t, "doc", "download", docID))
	localPath := filepath.Join(".compass", docID+".md")
	assert.FileExists(t, localPath)

	require.NoError(t, run(t, "doc", "upload", docID))
	assert.NoFileExists(t, localPath)

	// 12. Delete project
	require.NoError(t, run(t, "project", "delete", "EP", "--force"))

	api.mu.Lock()
	assert.Len(t, api.projects, 0)
	api.mu.Unlock()
}

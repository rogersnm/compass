package store

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rogersnm/compass/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCloudStore(handler http.HandlerFunc) (*CloudStore, *httptest.Server) {
	srv := httptest.NewServer(handler)
	cs := NewCloudStoreWithBase(srv.URL, "test-api-key")
	return cs, srv
}

func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func TestCloudStore_CreateProject(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/projects", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "My Project", body["name"])
		assert.Equal(t, "MP", body["key"])

		jsonResponse(w, 201, map[string]any{
			"data": map[string]any{
				"project_id": "uuid-123",
				"key":        "MP",
				"name":       "My Project",
				"body":       "",
				"created_at": "2026-01-01T00:00:00Z",
			},
		})
	})
	defer srv.Close()

	p, err := cs.CreateProject("My Project", "MP", "")
	require.NoError(t, err)
	assert.Equal(t, "MP", p.ID)
	assert.Equal(t, "My Project", p.Name)
}

func TestCloudStore_GetProject(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/projects/MP", r.URL.Path)

		jsonResponse(w, 200, map[string]any{
			"data": map[string]any{
				"project_id": "uuid-123",
				"key":        "MP",
				"name":       "My Project",
				"body":       "project body",
				"created_at": "2026-01-01T00:00:00Z",
			},
		})
	})
	defer srv.Close()

	p, body, err := cs.GetProject("MP")
	require.NoError(t, err)
	assert.Equal(t, "MP", p.ID)
	assert.Equal(t, "project body", body)
}

func TestCloudStore_ListProjects(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]any{
			"data": []map[string]any{
				{"project_id": "uuid-1", "key": "P1", "name": "Project 1", "body": "", "created_at": "2026-01-01T00:00:00Z"},
				{"project_id": "uuid-2", "key": "P2", "name": "Project 2", "body": "", "created_at": "2026-01-01T00:00:00Z"},
			},
		})
	})
	defer srv.Close()

	projects, err := cs.ListProjects()
	require.NoError(t, err)
	assert.Len(t, projects, 2)
	assert.Equal(t, "P1", projects[0].ID)
	assert.Equal(t, "P2", projects[1].ID)
}

func TestCloudStore_DeleteProject(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/projects/MP", r.URL.Path)
		jsonResponse(w, 200, map[string]any{"data": map[string]any{"message": "Project deleted"}})
	})
	defer srv.Close()

	err := cs.DeleteProject("MP")
	require.NoError(t, err)
}

func TestCloudStore_CreateTask(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/projects/MP/tasks", r.URL.Path)

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "My Task", body["title"])

		jsonResponse(w, 201, map[string]any{
			"data": map[string]any{
				"task_id":    "uuid-task",
				"key": "MP-TABCDE",
				"title":      "My Task",
				"type":       "task",
				"status":     "open",
				"body":       "",
				"created_at": "2026-01-01T00:00:00Z",
			},
		})
	})
	defer srv.Close()

	task, err := cs.CreateTask("My Task", "MP", TaskCreateOpts{})
	require.NoError(t, err)
	assert.Equal(t, "MP-TABCDE", task.ID)
	assert.Equal(t, "My Task", task.Title)
	assert.Equal(t, "MP", task.Project)
}

func TestCloudStore_GetTask(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/tasks/MP-TABCDE", r.URL.Path)

		jsonResponse(w, 200, map[string]any{
			"data": map[string]any{
				"task_id":    "uuid-task",
				"key": "MP-TABCDE",
				"title":      "My Task",
				"type":       "task",
				"status":     "open",
				"body":       "task body",
				"created_at": "2026-01-01T00:00:00Z",
			},
		})
	})
	defer srv.Close()

	task, body, err := cs.GetTask("MP-TABCDE")
	require.NoError(t, err)
	assert.Equal(t, "MP-TABCDE", task.ID)
	assert.Equal(t, "task body", body)
}

func TestCloudStore_UpdateTask(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// Epic guard: GET to check task type
			jsonResponse(w, 200, map[string]any{
				"data": map[string]any{
					"task_id":    "uuid-task",
					"key": "MP-TABCDE",
					"title":      "My Task",
					"type":       "task",
					"status":     "open",
					"body":       "",
					"created_at": "2026-01-01T00:00:00Z",
				},
			})
			return
		}

		assert.Equal(t, "PATCH", r.Method)
		assert.Equal(t, "/tasks/MP-TABCDE", r.URL.Path)

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "in_progress", body["status"])

		jsonResponse(w, 200, map[string]any{
			"data": map[string]any{
				"task_id":    "uuid-task",
				"key": "MP-TABCDE",
				"title":      "My Task",
				"type":       "task",
				"status":     "in_progress",
				"body":       "",
				"created_at": "2026-01-01T00:00:00Z",
			},
		})
	})
	defer srv.Close()

	s := model.StatusInProgress
	task, err := cs.UpdateTask("MP-TABCDE", TaskUpdate{Status: &s})
	require.NoError(t, err)
	assert.Equal(t, model.StatusInProgress, task.Status)
}

func TestCloudStore_UpdateTask_EpicStatusRejected(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		// GET to check task type
		jsonResponse(w, 200, map[string]any{
			"data": map[string]any{
				"task_id":    "uuid-epic",
				"key": "MP-TEPIC1",
				"title":      "My Epic",
				"type":       "epic",
				"status":     "open",
				"body":       "",
				"created_at": "2026-01-01T00:00:00Z",
			},
		})
	})
	defer srv.Close()

	s := model.StatusInProgress
	_, err := cs.UpdateTask("MP-TEPIC1", TaskUpdate{Status: &s})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "epics do not have a status")
}

func TestCloudStore_DeleteTask(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/tasks/MP-TABCDE", r.URL.Path)
		jsonResponse(w, 200, map[string]any{"data": map[string]any{"message": "Task deleted"}})
	})
	defer srv.Close()

	err := cs.DeleteTask("MP-TABCDE")
	require.NoError(t, err)
}

func TestCloudStore_ListTasks(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/projects/MP/tasks")
		jsonResponse(w, 200, map[string]any{
			"data": []map[string]any{
				{"task_id": "uuid-1", "key": "MP-T00001", "title": "T1", "type": "task", "status": "open", "body": "", "created_at": "2026-01-01T00:00:00Z"},
				{"task_id": "uuid-2", "key": "MP-T00002", "title": "T2", "type": "task", "status": "open", "body": "", "created_at": "2026-01-01T00:00:00Z"},
			},
		})
	})
	defer srv.Close()

	tasks, err := cs.ListTasks(TaskFilter{ProjectID: "MP"})
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, "MP-T00001", tasks[0].ID)
}

func TestCloudStore_ReadyTasks(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/projects/MP/tasks/ready", r.URL.Path)
		jsonResponse(w, 200, map[string]any{
			"data": []map[string]any{
				{"task_id": "uuid-1", "key": "MP-T00001", "title": "Ready Task", "type": "task", "status": "open", "body": "", "created_at": "2026-01-01T00:00:00Z"},
			},
		})
	})
	defer srv.Close()

	ready, err := cs.ReadyTasks("MP")
	require.NoError(t, err)
	assert.Len(t, ready, 1)
	assert.Equal(t, "Ready Task", ready[0].Title)
}

func TestCloudStore_CreateDocument(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/projects/MP/documents", r.URL.Path)

		jsonResponse(w, 201, map[string]any{
			"data": map[string]any{
				"document_id": "uuid-doc",
				"key":  "MP-DABCDE",
				"title":       "My Doc",
				"body":        "doc body",
				"created_at":  "2026-01-01T00:00:00Z",
			},
		})
	})
	defer srv.Close()

	d, err := cs.CreateDocument("My Doc", "MP", "doc body")
	require.NoError(t, err)
	assert.Equal(t, "MP-DABCDE", d.ID)
	assert.Equal(t, "MP", d.Project)
}

func TestCloudStore_Search(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "auth", r.URL.Query().Get("q"))
		jsonResponse(w, 200, map[string]any{
			"data": []map[string]any{
				{"type": "task", "id": "MP-TABCDE", "title": "Auth Task", "snippet": "...auth..."},
			},
		})
	})
	defer srv.Close()

	results, err := cs.Search("auth", "")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Auth Task", results[0].Title)
}

func TestCloudStore_APIError(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 404, map[string]any{
			"error": map[string]any{
				"code":    "NOT_FOUND",
				"message": "Project not found",
			},
		})
	})
	defer srv.Close()

	_, _, err := cs.GetProject("ZZZZ")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Project not found")
}

func TestCloudStore_CheckoutTask(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/tasks/MP-TABCDE", r.URL.Path)
		jsonResponse(w, 200, map[string]any{
			"data": map[string]any{
				"task_id":    "uuid-task",
				"key": "MP-TABCDE",
				"title":      "My Task",
				"type":       "task",
				"status":     "open",
				"body":       "task body content",
				"created_at": "2026-01-01T00:00:00Z",
			},
		})
	})
	defer srv.Close()

	destDir := t.TempDir()
	localPath, err := cs.CheckoutEntity("MP-TABCDE", destDir)
	require.NoError(t, err)
	assert.FileExists(t, localPath)
	assert.Contains(t, localPath, "MP-TABCDE.md")

	// Verify file is parseable
	task, body, err := ReadEntity[model.Task](localPath)
	require.NoError(t, err)
	assert.Equal(t, "MP-TABCDE", task.ID)
	assert.Equal(t, "My Task", task.Title)
	assert.Equal(t, "task body content", body)
}

func TestCloudStore_CheckoutDocument(t *testing.T) {
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/documents/MP-DABCDE", r.URL.Path)
		jsonResponse(w, 200, map[string]any{
			"data": map[string]any{
				"document_id": "uuid-doc",
				"key":  "MP-DABCDE",
				"title":       "My Doc",
				"body":        "doc body",
				"created_at":  "2026-01-01T00:00:00Z",
			},
		})
	})
	defer srv.Close()

	destDir := t.TempDir()
	localPath, err := cs.CheckoutEntity("MP-DABCDE", destDir)
	require.NoError(t, err)
	assert.FileExists(t, localPath)

	doc, body, err := ReadEntity[model.Document](localPath)
	require.NoError(t, err)
	assert.Equal(t, "MP-DABCDE", doc.ID)
	assert.Equal(t, "doc body", body)
}

func TestCloudStore_CheckinTask(t *testing.T) {
	var patchBody map[string]any
	callCount := 0
	cs, srv := newTestCloudStore(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "PATCH" {
			json.NewDecoder(r.Body).Decode(&patchBody)
			jsonResponse(w, 200, map[string]any{
				"data": map[string]any{
					"task_id":    "uuid-task",
					"key": "MP-TABCDE",
					"title":      "Updated Title",
					"type":       "task",
					"status":     "open",
					"body":       "updated body",
					"created_at": "2026-01-01T00:00:00Z",
				},
			})
		}
	})
	defer srv.Close()

	// Write a local task file
	destDir := t.TempDir()
	task := &model.Task{
		ID:     "MP-TABCDE",
		Title:  "Updated Title",
		Type:   model.TypeTask,
		Status: model.StatusOpen,
	}
	localPath := destDir + "/MP-TABCDE.md"
	require.NoError(t, cs.WriteEntity(localPath, task, "updated body"))

	result, err := cs.CheckinTask(localPath)
	require.NoError(t, err)
	assert.Equal(t, "MP-TABCDE", result.ID)
	assert.Equal(t, "Updated Title", patchBody["title"])
	assert.Equal(t, "updated body", patchBody["body"])

	// Local file should be removed
	assert.NoFileExists(t, localPath)
}

func TestCloudStore_ResolveEntityPath_Unsupported(t *testing.T) {
	cs := NewCloudStoreWithBase("http://localhost", "key")
	_, err := cs.ResolveEntityPath("MP-TABCDE")
	assert.Error(t, err)
}

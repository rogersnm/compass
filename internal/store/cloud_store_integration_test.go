//go:build integration

package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/rogersnm/compass/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests run against a live compass-cloud API.
// Set COMPASS_TEST_API_URL to the API base URL (e.g. http://localhost:3000).
//
// Run: go test -tags integration ./internal/store/ -v

func getTestAPIURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("COMPASS_TEST_API_URL")
	if url == "" {
		t.Skip("COMPASS_TEST_API_URL not set, skipping integration tests")
	}
	return url
}

// setupTestUser registers a user+org in one call, creates an API key, and returns it.
func setupTestUser(t *testing.T, apiURL string) string {
	t.Helper()

	unique := fmt.Sprintf("test-%d-%d", os.Getpid(), time.Now().UnixNano())
	email := unique + "@test.com"
	orgSlug := "org-" + unique

	// Register with org
	regBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": "testpassword123",
		"name":     "Test User",
		"org_name": "Test Org",
		"org_slug": orgSlug,
	})
	resp, err := http.Post(apiURL+"/api/v1/auth/register", "application/json", bytes.NewReader(regBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, 201, resp.StatusCode, "register should return 201")

	var regResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&regResp))
	token := regResp.Data.AccessToken
	require.NotEmpty(t, token, "register should return an access token")

	// Create API key (JWT auth needs X-Org-Slug)
	keyBody, _ := json.Marshal(map[string]string{"name": "test-key"})
	req, _ := http.NewRequest("POST", apiURL+"/api/v1/auth/keys", bytes.NewReader(keyBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Org-Slug", orgSlug)
	req.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp2.Body.Close()
	require.Equal(t, 201, resp2.StatusCode, "create API key should return 201")

	var keyResp struct {
		Data struct {
			Key string `json:"key"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&keyResp))
	require.NotEmpty(t, keyResp.Data.Key, "API key should not be empty")
	return keyResp.Data.Key
}

func TestIntegration_ProjectCRUD(t *testing.T) {
	apiURL := getTestAPIURL(t)
	apiKey := setupTestUser(t, apiURL)
	cs := NewCloudStoreWithBase(apiURL, apiKey)

	// Create
	p, err := cs.CreateProject("Integration Test", "IT", "")
	require.NoError(t, err)
	assert.Equal(t, "IT", p.ID)
	assert.Equal(t, "Integration Test", p.Name)

	// Get
	got, body, err := cs.GetProject("IT")
	require.NoError(t, err)
	assert.Equal(t, "IT", got.ID)
	assert.Equal(t, "", body)

	// List
	projects, err := cs.ListProjects()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(projects), 1)

	// Delete
	err = cs.DeleteProject("IT")
	require.NoError(t, err)

	_, _, err = cs.GetProject("IT")
	assert.Error(t, err)
}

func TestIntegration_TaskCRUD(t *testing.T) {
	apiURL := getTestAPIURL(t)
	apiKey := setupTestUser(t, apiURL)
	cs := NewCloudStoreWithBase(apiURL, apiKey)

	p, err := cs.CreateProject("Task Test", "TT", "")
	require.NoError(t, err)

	// Create
	task, err := cs.CreateTask("My Task", p.ID, TaskCreateOpts{Body: "task body"})
	require.NoError(t, err)
	assert.Equal(t, "My Task", task.Title)

	// Get
	got, body, err := cs.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, got.ID)
	assert.Equal(t, "task body", body)

	// Update
	newTitle := "Updated Task"
	updated, err := cs.UpdateTask(task.ID, TaskUpdate{Title: &newTitle})
	require.NoError(t, err)
	assert.Equal(t, "Updated Task", updated.Title)

	// List
	tasks, err := cs.ListTasks(TaskFilter{ProjectID: p.ID})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasks), 1)

	// Delete
	err = cs.DeleteTask(task.ID)
	require.NoError(t, err)
}

func TestIntegration_DocumentCRUD(t *testing.T) {
	apiURL := getTestAPIURL(t)
	apiKey := setupTestUser(t, apiURL)
	cs := NewCloudStoreWithBase(apiURL, apiKey)

	p, err := cs.CreateProject("Doc Test", "DT", "")
	require.NoError(t, err)

	// Create
	doc, err := cs.CreateDocument("My Doc", p.ID, "doc body")
	require.NoError(t, err)
	assert.Equal(t, "My Doc", doc.Title)

	// Get
	got, body, err := cs.GetDocument(doc.ID)
	require.NoError(t, err)
	assert.Equal(t, doc.ID, got.ID)
	assert.Equal(t, "doc body", body)

	// Update
	newTitle := "Updated Doc"
	newBody := "new body"
	updated, err := cs.UpdateDocument(doc.ID, &newTitle, &newBody)
	require.NoError(t, err)
	assert.Equal(t, "Updated Doc", updated.Title)

	// List
	docs, err := cs.ListDocuments(p.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(docs), 1)

	// Delete
	err = cs.DeleteDocument(doc.ID)
	require.NoError(t, err)
}

func TestIntegration_ReadyTasks(t *testing.T) {
	apiURL := getTestAPIURL(t)
	apiKey := setupTestUser(t, apiURL)
	cs := NewCloudStoreWithBase(apiURL, apiKey)

	p, err := cs.CreateProject("Ready Test", "RT", "")
	require.NoError(t, err)

	// Create tasks
	t1, err := cs.CreateTask("Task 1", p.ID, TaskCreateOpts{})
	require.NoError(t, err)
	_, err = cs.CreateTask("Task 2", p.ID, TaskCreateOpts{})
	require.NoError(t, err)

	// Both should be ready
	ready, err := cs.ReadyTasks(p.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(ready), 2)

	// Close one
	closed := model.StatusClosed
	_, err = cs.UpdateTask(t1.ID, TaskUpdate{Status: &closed})
	require.NoError(t, err)

	// Should still have at least one ready
	ready, err = cs.ReadyTasks(p.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(ready), 1)
}

func TestIntegration_CheckoutCheckin(t *testing.T) {
	apiURL := getTestAPIURL(t)
	apiKey := setupTestUser(t, apiURL)
	cs := NewCloudStoreWithBase(apiURL, apiKey)

	p, err := cs.CreateProject("Checkout Test", "CT", "")
	require.NoError(t, err)

	task, err := cs.CreateTask("Checkout Task", p.ID, TaskCreateOpts{Body: "original body"})
	require.NoError(t, err)

	// Checkout
	destDir := t.TempDir()
	localPath, err := cs.CheckoutEntity(task.ID, destDir)
	require.NoError(t, err)
	assert.FileExists(t, localPath)

	// Read and verify
	local, body, err := ReadEntity[model.Task](localPath)
	require.NoError(t, err)
	assert.Equal(t, task.ID, local.ID)
	assert.Equal(t, "original body", body)

	// Modify locally
	local.Title = "Modified Title"
	require.NoError(t, cs.WriteEntity(localPath, &local, "modified body"))

	// Checkin
	result, err := cs.CheckinTask(localPath)
	require.NoError(t, err)
	assert.Equal(t, "Modified Title", result.Title)

	// Local file should be gone
	assert.NoFileExists(t, localPath)

	// Verify API updated
	got, body, err := cs.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, "Modified Title", got.Title)
	assert.Equal(t, "modified body", body)
}

func TestIntegration_Search(t *testing.T) {
	apiURL := getTestAPIURL(t)
	apiKey := setupTestUser(t, apiURL)
	cs := NewCloudStoreWithBase(apiURL, apiKey)

	p, err := cs.CreateProject("Search Test", "ST", "")
	require.NoError(t, err)

	_, err = cs.CreateTask("Authentication Module", p.ID, TaskCreateOpts{})
	require.NoError(t, err)

	results, err := cs.Search("Authentication", "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

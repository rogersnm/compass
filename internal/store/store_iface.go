package store

import "github.com/rogersnm/compass/internal/model"

// Store defines the interface for all storage operations. LocalStore implements
// this for file-based storage; CloudStore will implement it for HTTP-backed storage.
type Store interface {
	// Projects
	CreateProject(name, key, body string) (*model.Project, error)
	GetProject(projectID string) (*model.Project, string, error)
	ListProjects() ([]model.Project, error)
	DeleteProject(projectID string) error

	// Tasks
	CreateTask(title, projectID string, opts TaskCreateOpts) (*model.Task, error)
	GetTask(taskID string) (*model.Task, string, error)
	ListTasks(filter TaskFilter) ([]model.Task, error)
	UpdateTask(taskID string, upd TaskUpdate) (*model.Task, error)
	DeleteTask(taskID string) error
	AllTaskMap(projectID string) (map[string]*model.Task, error)
	ReadyTasks(projectID string) ([]*model.Task, error)

	// Documents
	CreateDocument(title, projectID, body string) (*model.Document, error)
	GetDocument(docID string) (*model.Document, string, error)
	ListDocuments(projectID string) ([]model.Document, error)
	UpdateDocument(docID string, title, body *string) (*model.Document, error)
	DeleteDocument(docID string) error

	// Search
	Search(query, projectID string) ([]SearchResult, error)

	// Entity operations
	ResolveEntityPath(entityID string) (string, error)
	DownloadEntity(entityID, destDir string) (string, error)
	UploadTask(localPath string) (*model.Task, error)
	UploadDocument(localPath string) (*model.Document, error)

	// Low-level (used by commands for raw file access)
	WriteEntity(path string, meta any, body string) error
}

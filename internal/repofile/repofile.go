package repofile

import (
	"os"
	"path/filepath"
	"strings"
)

const FileName = ".compass-project"

// Find walks up from startDir looking for a .compass-project file.
// Returns the project ID and the directory containing the file.
// Returns ("", "", nil) if not found.
func Find(startDir string) (projectID, dir string, err error) {
	dir = startDir
	for {
		id, err := Read(dir)
		if err != nil {
			return "", "", err
		}
		if id != "" {
			return id, dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", nil
		}
		dir = parent
	}
}

// Write writes projectID to dir/.compass-project.
func Write(dir, projectID string) error {
	return os.WriteFile(filepath.Join(dir, FileName), []byte(projectID+"\n"), 0644)
}

// Read reads and trims the .compass-project file in dir.
// Returns ("", nil) if the file does not exist.
func Read(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

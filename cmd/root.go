package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	mtp "github.com/modeltoolsprotocol/go-sdk"
	"github.com/rogersnm/compass/internal/config"
	"github.com/rogersnm/compass/internal/store"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	dataDir string
	st      *store.Store
	cfg     *config.Config
)

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".compass")
	}
	return filepath.Join(home, ".compass")
}

var rootCmd = &cobra.Command{
	Use:     "compass",
	Short:   "Markdown-native task and document tracking",
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return fmt.Errorf("creating data directory: %w", err)
		}

		st = store.New(dataDir)

		var err error
		cfg, err = config.Load(dataDir)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		return nil
	},
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", defaultDataDir(), "data directory path")

	mtpOpts := &mtp.DescribeOptions{
		Commands: map[string]*mtp.CommandAnnotation{
			"project create": {
				Examples: []mtp.Example{
					{Description: "Create a new project", Command: "compass project create \"My Project\""},
					{Description: "Create with explicit key", Command: "compass project create \"My Project\" --key MYPR"},
				},
			},
			"doc create": {
				Stdin: &mtp.IODescriptor{
					ContentType: "text/markdown",
					Description: "Markdown body content for the document",
				},
				Examples: []mtp.Example{
					{Description: "Create doc with piped content", Command: "echo '# Design' | compass doc create \"Design Doc\" --project AUTH"},
				},
			},
			"doc checkout": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "Local file path where the document was checked out (e.g. .compass/AUTH-DXXXXX.md)",
				},
				Examples: []mtp.Example{
					{Description: "Checkout a document for local editing", Command: "compass doc checkout AUTH-DXXXXX"},
				},
			},
			"doc checkin": {
				Examples: []mtp.Example{
					{Description: "Check in a locally edited document", Command: "compass doc checkin AUTH-DXXXXX"},
				},
			},
			"doc update": {
				Stdin: &mtp.IODescriptor{
					ContentType: "text/markdown",
					Description: "New markdown body content for the document",
				},
				Examples: []mtp.Example{
					{Description: "Update document title", Command: "compass doc update AUTH-DXXXXX --title \"New Title\""},
					{Description: "Update document body", Command: "echo '# Updated' | compass doc update AUTH-DXXXXX"},
				},
			},
			"doc delete": {
				Examples: []mtp.Example{
					{Description: "Delete a document (interactive confirm)", Command: "compass doc delete AUTH-DXXXXX"},
					{Description: "Delete a document (skip confirm)", Command: "compass doc delete AUTH-DXXXXX --force"},
				},
			},
			"task create": {
				Stdin: &mtp.IODescriptor{
					ContentType: "text/markdown",
					Description: "Markdown body content for the task",
				},
				Examples: []mtp.Example{
					{Description: "Create a task with dependencies", Command: "compass task create \"Login\" --project AUTH --epic AUTH-TXXXXX --depends-on AUTH-TAAAAA,AUTH-TBBBBB"},
					{Description: "Create an epic", Command: "compass task create \"Auth\" --project AUTH --type epic"},
					{Description: "Create a high-priority task", Command: "compass task create \"Urgent fix\" --project AUTH --priority 0"},
				},
			},
			"task start": {
				Examples: []mtp.Example{
					{Description: "Start a task", Command: "compass task start AUTH-TXXXXX"},
				},
			},
			"task close": {
				Examples: []mtp.Example{
					{Description: "Close a task", Command: "compass task close AUTH-TXXXXX"},
				},
			},
			"task ready": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "Next ready task or table of all ready tasks with --all",
				},
				Examples: []mtp.Example{
					{Description: "Show next ready task", Command: "compass task ready --project AUTH"},
					{Description: "Show all ready tasks", Command: "compass task ready --project AUTH --all"},
				},
			},
			"task update": {
				Stdin: &mtp.IODescriptor{
					ContentType: "text/markdown",
					Description: "New markdown body content for the task",
				},
				Examples: []mtp.Example{
					{Description: "Update task title", Command: "compass task update AUTH-TXXXXX --title \"New Title\""},
					{Description: "Update task body", Command: "echo '# Updated' | compass task update AUTH-TXXXXX"},
					{Description: "Set task priority", Command: "compass task update AUTH-TXXXXX --priority 1"},
				},
			},
			"task delete": {
				Examples: []mtp.Example{
					{Description: "Delete a task (interactive confirm)", Command: "compass task delete AUTH-TXXXXX"},
					{Description: "Delete a task (skip confirm)", Command: "compass task delete AUTH-TXXXXX --force"},
				},
			},
			"task checkout": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "Local file path where the task was checked out (e.g. .compass/AUTH-TXXXXX.md)",
				},
				Examples: []mtp.Example{
					{Description: "Checkout a task for local editing", Command: "compass task checkout AUTH-TXXXXX"},
				},
			},
			"task checkin": {
				Examples: []mtp.Example{
					{Description: "Check in a locally edited task", Command: "compass task checkin AUTH-TXXXXX"},
				},
			},
			"task list": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "Table of tasks with ID, title, status (with blocked annotation), and project",
				},
			},
			"task graph": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "ASCII tree visualization of the task dependency DAG",
				},
			},
			"search": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "Search results grouped by entity type with ID, title, and snippet",
				},
				Examples: []mtp.Example{
					{Description: "Search across all entities", Command: "compass search \"authentication\""},
				},
			},
		},
	}

	mtp.WithDescribe(rootCmd, mtpOpts)
}

func Execute() error {
	return rootCmd.Execute()
}

// resolveProject returns the project ID from the flag or the default config.
func resolveProject(cmd *cobra.Command) (string, error) {
	p, _ := cmd.Flags().GetString("project")
	if p != "" {
		return p, nil
	}
	if cfg != nil && cfg.DefaultProject != "" {
		return cfg.DefaultProject, nil
	}
	return "", fmt.Errorf("--project is required (or set a default with: compass project set-default <id>)")
}

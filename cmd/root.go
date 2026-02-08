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
	Version: "0.1.0",
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
				},
			},
			"doc create": {
				Stdin: &mtp.IODescriptor{
					ContentType: "text/markdown",
					Description: "Markdown body content for the document",
				},
				Examples: []mtp.Example{
					{Description: "Create doc with piped content", Command: "echo '# Design' | compass doc create \"Design Doc\" --project PROJ-XXXXX"},
				},
			},
			"epic create": {
				Stdin: &mtp.IODescriptor{
					ContentType: "text/markdown",
					Description: "Markdown body content for the epic",
				},
				Examples: []mtp.Example{
					{Description: "Create an epic", Command: "compass epic create \"Auth\" --project PROJ-XXXXX"},
				},
			},
			"task create": {
				Stdin: &mtp.IODescriptor{
					ContentType: "text/markdown",
					Description: "Markdown body content for the task",
				},
				Examples: []mtp.Example{
					{Description: "Create a task with dependencies", Command: "compass task create \"Login\" --project PROJ-XXXXX --epic EPIC-XXXXX --depends-on TASK-AAAAA,TASK-BBBBB"},
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

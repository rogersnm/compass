package cmd

import (
	"fmt"
	"os"

	"github.com/rogersnm/compass/internal/config"
	"github.com/rogersnm/compass/internal/markdown"
	"github.com/rogersnm/compass/internal/store"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
}

var projectCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, _ := cmd.Flags().GetString("key")
		body := readStdin()
		p, err := st.CreateProject(args[0], key, body)
		if err != nil {
			return err
		}
		fmt.Printf("Created project %s (%s)\n", p.Name, p.ID)
		return nil
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		projects, err := st.ListProjects()
		if err != nil {
			return err
		}
		fmt.Println(markdown.RenderProjectTable(projects))
		return nil
	},
}

var projectShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show project details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		raw, _ := cmd.Flags().GetBool("raw")
		if raw {
			path, err := st.ResolveEntityPath(args[0])
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fmt.Print(string(data))
			return nil
		}

		p, body, err := st.GetProject(args[0])
		if err != nil {
			return err
		}
		fields := []string{
			markdown.RenderField("ID", p.ID),
			markdown.RenderField("Created by", p.CreatedBy),
			markdown.RenderField("Created", p.CreatedAt.Format("2006-01-02 15:04:05")),
			markdown.RenderField("Updated", p.UpdatedAt.Format("2006-01-02 15:04:05")),
		}
		fmt.Print(markdown.RenderEntityHeader(p.Name, fields))
		if body != "" {
			rendered, err := markdown.RenderMarkdown(body)
			if err != nil {
				return err
			}
			fmt.Print(rendered)
		}
		return nil
	},
}

var projectSetDefaultCmd = &cobra.Command{
	Use:   "set-default <id>",
	Short: "Set the default project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, _, err := st.GetProject(args[0]); err != nil {
			return err
		}
		cfg.DefaultProject = args[0]
		if err := config.Save(dataDir, cfg); err != nil {
			return err
		}
		fmt.Printf("Default project set to %s\n", args[0])
		return nil
	},
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a project and all its tasks and documents",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, _, err := st.GetProject(args[0])
		if err != nil {
			return err
		}

		tasks, _ := st.ListTasks(store.TaskFilter{ProjectID: p.ID})
		docs, _ := st.ListDocuments(p.ID)
		fmt.Printf("Project: %s (%s), %d tasks, %d documents\n", p.Name, p.ID, len(tasks), len(docs))

		if err := confirmDelete(cmd, p.ID); err != nil {
			return err
		}
		if err := st.DeleteProject(p.ID); err != nil {
			return err
		}

		if cfg.DefaultProject == p.ID {
			cfg.DefaultProject = ""
			config.Save(dataDir, cfg)
		}

		fmt.Printf("Deleted project %s\n", p.ID)
		return nil
	},
}

func init() {
	projectCreateCmd.Flags().StringP("key", "k", "", "project key (2-5 uppercase alphanumeric chars)")
	projectShowCmd.Flags().Bool("raw", false, "output raw markdown file (no ANSI styling)")
	projectDeleteCmd.Flags().BoolP("force", "f", false, "skip confirmation")

	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectShowCmd)
	projectCmd.AddCommand(projectSetDefaultCmd)
	projectCmd.AddCommand(projectDeleteCmd)
	rootCmd.AddCommand(projectCmd)
}

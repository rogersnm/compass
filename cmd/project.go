package cmd

import (
	"fmt"

	"github.com/rogersnm/compass/internal/config"
	"github.com/rogersnm/compass/internal/markdown"
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
		p, err := st.CreateProject(args[0], "")
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

func init() {
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectShowCmd)
	projectCmd.AddCommand(projectSetDefaultCmd)
	rootCmd.AddCommand(projectCmd)
}

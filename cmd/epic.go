package cmd

import (
	"fmt"

	"github.com/rogersnm/compass/internal/markdown"
	"github.com/rogersnm/compass/internal/model"
	"github.com/rogersnm/compass/internal/store"
	"github.com/spf13/cobra"
)

var epicCmd = &cobra.Command{
	Use:   "epic",
	Short: "Manage epics",
}

var epicCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new epic",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(cmd)
		if err != nil {
			return err
		}

		body := readStdin()

		e, err := st.CreateEpic(args[0], projectID, body)
		if err != nil {
			return err
		}
		fmt.Printf("Created epic %s (%s)\n", e.Title, e.ID)
		return nil
	},
}

var epicListCmd = &cobra.Command{
	Use:   "list",
	Short: "List epics",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, _ := cmd.Flags().GetString("project")
		epics, err := st.ListEpics(projectID)
		if err != nil {
			return err
		}
		fmt.Println(markdown.RenderEpicTable(epics))
		return nil
	},
}

var epicShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show epic details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		e, body, err := st.GetEpic(args[0])
		if err != nil {
			return err
		}

		fields := []string{
			markdown.RenderField("ID", e.ID),
			markdown.RenderField("Project", e.Project),
			markdown.RenderField("Status", markdown.RenderStatus(string(e.Status), false)),
			markdown.RenderField("Created by", e.CreatedBy),
			markdown.RenderField("Created", e.CreatedAt.Format("2006-01-02 15:04:05")),
			markdown.RenderField("Updated", e.UpdatedAt.Format("2006-01-02 15:04:05")),
		}
		fmt.Print(markdown.RenderEntityHeader(e.Title, fields))

		if body != "" {
			rendered, err := markdown.RenderMarkdown(body)
			if err != nil {
				return err
			}
			fmt.Print(rendered)
		}

		// List tasks belonging to this epic
		tasks, err := st.ListTasks(store.TaskFilter{EpicID: e.ID})
		if err != nil {
			return err
		}
		if len(tasks) > 0 {
			allTasks, _ := st.AllTaskMap(e.Project)
			fmt.Println("\nTasks:")
			fmt.Println(markdown.RenderTaskTable(tasks, allTasks))
		}
		return nil
	},
}

var epicUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an epic",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		statusStr, _ := cmd.Flags().GetString("status")
		if statusStr == "" {
			return fmt.Errorf("at least one update flag is required (--status)")
		}
		status := model.Status(statusStr)
		e, err := st.UpdateEpic(args[0], &status)
		if err != nil {
			return err
		}
		fmt.Printf("Updated epic %s\n", e.ID)
		return nil
	},
}

func init() {
	epicCreateCmd.Flags().StringP("project", "p", "", "project ID")
	epicListCmd.Flags().StringP("project", "p", "", "filter by project")
	epicUpdateCmd.Flags().StringP("status", "s", "", "new status (open, in_progress, closed)")
	epicCmd.AddCommand(epicCreateCmd)
	epicCmd.AddCommand(epicListCmd)
	epicCmd.AddCommand(epicShowCmd)
	epicCmd.AddCommand(epicUpdateCmd)
	rootCmd.AddCommand(epicCmd)
}

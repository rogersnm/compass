package cmd

import (
	"fmt"
	"strings"

	"github.com/rogersnm/compass/internal/dag"
	"github.com/rogersnm/compass/internal/editor"
	"github.com/rogersnm/compass/internal/markdown"
	"github.com/rogersnm/compass/internal/model"
	"github.com/rogersnm/compass/internal/store"
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
}

var taskCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(cmd)
		if err != nil {
			return err
		}

		epicID, _ := cmd.Flags().GetString("epic")
		depsStr, _ := cmd.Flags().GetString("depends-on")

		var deps []string
		if depsStr != "" {
			deps = strings.Split(depsStr, ",")
		}

		body := readStdin()

		t, err := st.CreateTask(args[0], projectID, store.TaskCreateOpts{
			Epic:      epicID,
			DependsOn: deps,
			Body:      body,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Created task %s (%s)\n", t.Title, t.ID)
		return nil
	},
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, _ := cmd.Flags().GetString("project")
		epicID, _ := cmd.Flags().GetString("epic")
		statusStr, _ := cmd.Flags().GetString("status")

		filter := store.TaskFilter{
			ProjectID: projectID,
			EpicID:    epicID,
			Status:    model.Status(statusStr),
		}

		tasks, err := st.ListTasks(filter)
		if err != nil {
			return err
		}

		allTasks, _ := st.AllTaskMap(projectID)
		fmt.Println(markdown.RenderTaskTable(tasks, allTasks))
		return nil
	},
}

var taskShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show task details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, body, err := st.GetTask(args[0])
		if err != nil {
			return err
		}

		allTasks, _ := st.AllTaskMap(t.Project)
		blocked := t.IsBlocked(allTasks)

		fields := []string{
			markdown.RenderField("ID", t.ID),
			markdown.RenderField("Project", t.Project),
			markdown.RenderField("Status", markdown.RenderStatus(string(t.Status), blocked)),
			markdown.RenderField("Created by", t.CreatedBy),
			markdown.RenderField("Created", t.CreatedAt.Format("2006-01-02 15:04:05")),
			markdown.RenderField("Updated", t.UpdatedAt.Format("2006-01-02 15:04:05")),
		}
		if t.Epic != "" {
			fields = append(fields, markdown.RenderField("Epic", t.Epic))
		}
		if len(t.DependsOn) > 0 {
			fields = append(fields, markdown.RenderField("Depends on", strings.Join(t.DependsOn, ", ")))
		}

		// Show dependents
		projectTasks, _ := st.ListTasks(store.TaskFilter{ProjectID: t.Project})
		var dependents []string
		for _, pt := range projectTasks {
			for _, dep := range pt.DependsOn {
				if dep == t.ID {
					dependents = append(dependents, pt.ID)
				}
			}
		}
		if len(dependents) > 0 {
			fields = append(fields, markdown.RenderField("Dependents", strings.Join(dependents, ", ")))
		}

		fmt.Print(markdown.RenderEntityHeader(t.Title, fields))
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

var taskUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		upd := store.TaskUpdate{}

		if cmd.Flags().Changed("status") {
			statusStr, _ := cmd.Flags().GetString("status")
			s := model.Status(statusStr)
			upd.Status = &s
		}
		if cmd.Flags().Changed("depends-on") {
			depsStr, _ := cmd.Flags().GetString("depends-on")
			var deps []string
			if depsStr != "" {
				deps = strings.Split(depsStr, ",")
			}
			upd.DependsOn = &deps
		}

		if upd.Status == nil && upd.DependsOn == nil {
			return fmt.Errorf("at least one update flag is required (--status, --depends-on)")
		}

		t, err := st.UpdateTask(args[0], upd)
		if err != nil {
			return err
		}
		fmt.Printf("Updated task %s\n", t.ID)
		return nil
	},
}

var taskEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a task in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := st.ResolveEntityPath(args[0])
		if err != nil {
			return err
		}
		return editor.Open(path)
	},
}

var taskGraphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Show task dependency graph",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(cmd)
		if err != nil {
			return err
		}

		tasks, err := st.ListTasks(store.TaskFilter{ProjectID: projectID})
		if err != nil {
			return err
		}

		ptrs := make([]*model.Task, len(tasks))
		for i := range tasks {
			ptrs[i] = &tasks[i]
		}

		g := dag.BuildFromTasks(ptrs)
		fmt.Println(dag.RenderASCII(g))
		return nil
	},
}

func init() {
	taskCreateCmd.Flags().StringP("project", "p", "", "project ID")
	taskCreateCmd.Flags().StringP("epic", "e", "", "epic ID")
	taskCreateCmd.Flags().String("depends-on", "", "comma-separated task IDs")

	taskListCmd.Flags().StringP("project", "p", "", "filter by project")
	taskListCmd.Flags().StringP("epic", "e", "", "filter by epic")
	taskListCmd.Flags().StringP("status", "s", "", "filter by status (open, in_progress, closed)")

	taskUpdateCmd.Flags().StringP("status", "s", "", "new status (open, in_progress, closed)")
	taskUpdateCmd.Flags().String("depends-on", "", "comma-separated task IDs (replaces existing)")

	taskGraphCmd.Flags().StringP("project", "p", "", "project ID")

	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskShowCmd)
	taskCmd.AddCommand(taskUpdateCmd)
	taskCmd.AddCommand(taskEditCmd)
	taskCmd.AddCommand(taskGraphCmd)
	rootCmd.AddCommand(taskCmd)
}

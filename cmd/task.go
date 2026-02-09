package cmd

import (
	"fmt"
	"os"
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
		typeStr, _ := cmd.Flags().GetString("type")

		var deps []string
		if depsStr != "" {
			deps = strings.Split(depsStr, ",")
		}

		body := readStdin()

		var priority *int
		if p, _ := cmd.Flags().GetInt("priority"); p >= 0 {
			priority = &p
		}

		t, err := st.CreateTask(args[0], projectID, store.TaskCreateOpts{
			Type:      model.TaskType(typeStr),
			Epic:      epicID,
			Priority:  priority,
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
		typeStr, _ := cmd.Flags().GetString("type")

		filter := store.TaskFilter{
			ProjectID: projectID,
			EpicID:    epicID,
			Status:    model.Status(statusStr),
			Type:      model.TaskType(typeStr),
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
		pretty, _ := cmd.Flags().GetBool("pretty")
		if !pretty {
			path, err := st.ResolveEntityPath(args[0])
			if err == nil {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				fmt.Print(string(data))
				return nil
			}
			// Cloud mode: marshal from API response
			t, body, err := st.GetTask(args[0])
			if err != nil {
				return err
			}
			data, err := markdown.Marshal(t, body)
			if err != nil {
				return err
			}
			fmt.Print(string(data))
			return nil
		}

		t, body, err := st.GetTask(args[0])
		if err != nil {
			return err
		}

		allTasks, _ := st.AllTaskMap(t.Project)
		blocked := t.IsBlocked(allTasks)

		fields := []string{
			markdown.RenderField("ID", t.ID),
			markdown.RenderField("Type", string(t.Type)),
			markdown.RenderField("Project", t.Project),
			markdown.RenderField("Status", markdown.RenderStatus(string(t.Status), blocked)),
		}
		if t.Priority != nil {
			fields = append(fields, markdown.RenderField("Priority", model.FormatPriority(t.Priority)))
		}
		fields = append(fields,
			markdown.RenderField("Created by", t.CreatedBy),
			markdown.RenderField("Created", t.CreatedAt.Format("2006-01-02 15:04:05")),
			markdown.RenderField("Updated", t.UpdatedAt.Format("2006-01-02 15:04:05")),
		)
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

		// If epic-type, list child tasks
		if t.Type == model.TypeEpic {
			children, err := st.ListTasks(store.TaskFilter{EpicID: t.ID})
			if err != nil {
				return err
			}
			if len(children) > 0 {
				fmt.Println("\nTasks:")
				fmt.Println(markdown.RenderTaskTable(children, allTasks))
			}
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

		if cmd.Flags().Changed("title") {
			title, _ := cmd.Flags().GetString("title")
			upd.Title = &title
		}
		if cmd.Flags().Changed("status") {
			statusStr, _ := cmd.Flags().GetString("status")
			s := model.Status(statusStr)
			upd.Status = &s
		}
		if p, _ := cmd.Flags().GetInt("priority"); p >= 0 {
			pp := &p
			upd.Priority = &pp
		}
		if cmd.Flags().Changed("depends-on") {
			depsStr, _ := cmd.Flags().GetString("depends-on")
			var deps []string
			if depsStr != "" {
				deps = strings.Split(depsStr, ",")
			}
			upd.DependsOn = &deps
		}

		body := readStdin()
		if body != "" {
			upd.Body = &body
		}

		if upd.Title == nil && upd.Status == nil && upd.Priority == nil && upd.DependsOn == nil && upd.Body == nil {
			return fmt.Errorf("at least one update flag or piped body is required (--title, --status, --priority, --depends-on, stdin)")
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

		tasks, err := st.ListTasks(store.TaskFilter{ProjectID: projectID, Type: model.TypeTask})
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

var taskStartCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Start a task (set status to in_progress)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := model.StatusInProgress
		t, err := st.UpdateTask(args[0], store.TaskUpdate{Status: &s})
		if err != nil {
			return err
		}
		fmt.Printf("Started task %s\n", t.ID)
		return nil
	},
}

var taskCloseCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Close a task (set status to closed)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := model.StatusClosed
		t, err := st.UpdateTask(args[0], store.TaskUpdate{Status: &s})
		if err != nil {
			return err
		}
		fmt.Printf("Closed task %s\n", t.ID)
		return nil
	},
}

var taskDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, _, err := st.GetTask(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Task: %s (%s)\n", t.Title, t.ID)
		if err := confirmDelete(cmd, t.ID); err != nil {
			return err
		}
		if err := st.DeleteTask(t.ID); err != nil {
			return err
		}
		fmt.Printf("Deleted task %s\n", t.ID)
		return nil
	},
}

var taskReadyCmd = &cobra.Command{
	Use:   "ready",
	Short: "Show next ready task(s)",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(cmd)
		if err != nil {
			return err
		}

		ready, err := st.ReadyTasks(projectID)
		if err != nil {
			return err
		}

		if len(ready) == 0 {
			fmt.Println("No ready tasks.")
			return nil
		}

		showAll, _ := cmd.Flags().GetBool("all")
		if showAll {
			tasks := make([]model.Task, len(ready))
			for i, t := range ready {
				tasks[i] = *t
			}
			allTasks, _ := st.AllTaskMap(projectID)
			fmt.Println(markdown.RenderTaskTable(tasks, allTasks))
		} else {
			fmt.Printf("%s  %s\n", ready[0].ID, ready[0].Title)
		}
		return nil
	},
}

var taskCheckoutCmd = &cobra.Command{
	Use:   "checkout <id>",
	Short: "Copy a task to .compass/ in the current directory for local editing",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := st.CheckoutEntity(args[0], ".compass")
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	},
}

var taskCheckinCmd = &cobra.Command{
	Use:   "checkin <id>",
	Short: "Write a locally edited task back to the store and remove the local copy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		localPath := fmt.Sprintf(".compass/%s.md", args[0])
		t, err := st.CheckinTask(localPath)
		if err != nil {
			return err
		}
		fmt.Printf("Checked in task %s\n", t.ID)
		return nil
	},
}

func init() {
	taskCreateCmd.Flags().StringP("project", "P", "", "project ID")
	taskCreateCmd.Flags().StringP("epic", "e", "", "epic ID (must reference a type=epic task)")
	taskCreateCmd.Flags().StringP("type", "t", "task", "task type (task, epic)")
	taskCreateCmd.Flags().IntP("priority", "p", -1, "priority (0=P0 critical, 1=P1 high, 2=P2 medium, 3=P3 low)")
	taskCreateCmd.Flags().String("depends-on", "", "comma-separated task IDs")

	taskShowCmd.Flags().Bool("pretty", false, "render with ANSI styling")

	taskListCmd.Flags().StringP("project", "P", "", "filter by project")
	taskListCmd.Flags().StringP("epic", "e", "", "filter by epic")
	taskListCmd.Flags().StringP("status", "s", "", "filter by status (open, in_progress, closed)")
	taskListCmd.Flags().StringP("type", "t", "", "filter by type (task, epic)")

	taskUpdateCmd.Flags().String("title", "", "new title")
	taskUpdateCmd.Flags().StringP("status", "s", "", "new status (open, in_progress, closed)")
	taskUpdateCmd.Flags().IntP("priority", "p", -1, "priority (0-3, or -1 to clear)")
	taskUpdateCmd.Flags().String("depends-on", "", "comma-separated task IDs (replaces existing)")

	taskGraphCmd.Flags().StringP("project", "P", "", "project ID")

	taskReadyCmd.Flags().StringP("project", "P", "", "project ID")
	taskReadyCmd.Flags().BoolP("all", "a", false, "show all ready tasks")

	taskDeleteCmd.Flags().BoolP("force", "f", false, "skip confirmation")

	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskShowCmd)
	taskCmd.AddCommand(taskUpdateCmd)
	taskCmd.AddCommand(taskEditCmd)
	taskCmd.AddCommand(taskGraphCmd)
	taskCmd.AddCommand(taskStartCmd)
	taskCmd.AddCommand(taskCloseCmd)
	taskCmd.AddCommand(taskDeleteCmd)
	taskCmd.AddCommand(taskReadyCmd)
	taskCmd.AddCommand(taskCheckoutCmd)
	taskCmd.AddCommand(taskCheckinCmd)
	rootCmd.AddCommand(taskCmd)
}

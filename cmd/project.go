package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/rogersnm/compass/internal/config"
	"github.com/rogersnm/compass/internal/markdown"
	"github.com/rogersnm/compass/internal/repofile"
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

		// Resolve which store to use
		storeName, _ := cmd.Flags().GetString("store")
		var s store.Store
		var err error

		if storeName != "" {
			s, err = reg.Get(storeName)
			if err != nil {
				return err
			}
		} else {
			s, storeName, err = reg.Default()
			if err != nil {
				// Multiple stores, no default: prompt
				names := reg.Names()
				if len(names) == 0 {
					return fmt.Errorf("no stores configured; run 'compass store add local' or 'compass store add <hostname>'")
				}
				opts := make([]huh.Option[string], len(names))
				for i, n := range names {
					opts[i] = huh.NewOption(n, n)
				}
				if err := huh.NewSelect[string]().
					Title("Which store should this project be created on?").
					Options(opts...).
					Value(&storeName).
					Run(); err != nil {
					return fmt.Errorf("selection cancelled")
				}
				s, err = reg.Get(storeName)
				if err != nil {
					return err
				}
			}
		}

		p, err := s.CreateProject(args[0], key, body)
		if err != nil {
			return err
		}
		reg.CacheProject(p.ID, storeName)
		fmt.Printf("Created project %s (%s)\n", p.Name, p.ID)
		return nil
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		var rows []markdown.ProjectRow

		// Read from cache, fetch metadata from each mapped store
		if cfg.Projects != nil {
			for key, storeName := range cfg.Projects {
				s, err := reg.Get(storeName)
				if err != nil {
					continue
				}
				p, _, err := s.GetProject(key)
				if err != nil {
					continue // stale cache entry
				}
				rows = append(rows, markdown.ProjectRow{Project: *p, StoreName: storeName})
			}
		}

		// Also discover any local projects not yet cached
		if cfg.LocalEnabled {
			s, err := reg.Get("local")
			if err == nil {
				projects, err := s.ListProjects()
				if err == nil {
					for _, p := range projects {
						if cfg.Projects == nil || cfg.Projects[p.ID] == "" {
							rows = append(rows, markdown.ProjectRow{Project: p, StoreName: "local"})
							reg.CacheProject(p.ID, "local")
						}
					}
				}
			}
		}

		fmt.Println(markdown.RenderProjectTableWithStores(rows))
		return nil
	},
}

var projectShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show project details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := storeForProject(args[0])
		if err != nil {
			return err
		}

		pretty, _ := cmd.Flags().GetBool("pretty")
		if !pretty {
			path, err := s.ResolveEntityPath(args[0])
			if err == nil {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				fmt.Print(string(data))
				return nil
			}
			// Cloud mode: marshal from API response
			p, body, err := s.GetProject(args[0])
			if err != nil {
				return err
			}
			data, err := markdown.Marshal(p, body)
			if err != nil {
				return err
			}
			fmt.Print(string(data))
			return nil
		}

		p, body, err := s.GetProject(args[0])
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
		s, err := storeForProject(args[0])
		if err != nil {
			return err
		}
		if _, _, err := s.GetProject(args[0]); err != nil {
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
		s, err := storeForProject(args[0])
		if err != nil {
			return err
		}

		p, _, err := s.GetProject(args[0])
		if err != nil {
			return err
		}

		tasks, _ := s.ListTasks(store.TaskFilter{ProjectID: p.ID})
		docs, _ := s.ListDocuments(p.ID)
		fmt.Printf("Project: %s (%s), %d tasks, %d documents\n", p.Name, p.ID, len(tasks), len(docs))

		if err := confirmDelete(cmd, p.ID); err != nil {
			return err
		}
		if err := s.DeleteProject(p.ID); err != nil {
			return err
		}

		reg.UncacheProject(p.ID)

		if cfg.DefaultProject == p.ID {
			cfg.DefaultProject = ""
			config.Save(dataDir, cfg)
		}

		fmt.Printf("Deleted project %s\n", p.ID)
		return nil
	},
}

var projectSetStoreCmd = &cobra.Command{
	Use:   "set-store <project-key> <store-name>",
	Short: "Change which store a project is mapped to",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		storeName := args[1]

		s, err := reg.Get(storeName)
		if err != nil {
			return fmt.Errorf("store %q not found: %w", storeName, err)
		}

		if _, _, err := s.GetProject(key); err != nil {
			return fmt.Errorf("project %s not found on store %q: %w", key, storeName, err)
		}

		reg.CacheProject(key, storeName)
		if err := config.Save(dataDir, cfg); err != nil {
			return err
		}
		fmt.Printf("Project %s mapped to %s\n", key, storeName)
		return nil
	},
}

var projectLinkCmd = &cobra.Command{
	Use:   "link [project-id]",
	Short: "Link the current directory to a project",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var projectID string
		if len(args) == 1 {
			projectID = args[0]
		} else {
			// Collect projects from all stores
			var rows []markdown.ProjectRow
			for name, s := range reg.All() {
				projects, err := s.ListProjects()
				if err != nil {
					continue
				}
				for _, p := range projects {
					rows = append(rows, markdown.ProjectRow{Project: p, StoreName: name})
				}
			}
			if len(rows) == 0 {
				return fmt.Errorf("no projects exist; create one first with: compass project create <name>")
			}
			opts := make([]huh.Option[string], len(rows))
			for i, r := range rows {
				opts[i] = huh.NewOption(fmt.Sprintf("%s  %s  (%s)", r.Project.ID, r.Project.Name, r.StoreName), r.Project.ID)
			}
			if err := huh.NewSelect[string]().
				Title("Select a project").
				Options(opts...).
				Value(&projectID).
				Run(); err != nil {
				return fmt.Errorf("selection cancelled")
			}
		}

		s, err := storeForProject(projectID)
		if err != nil {
			return fmt.Errorf("project %s not found", projectID)
		}
		if _, _, err := s.GetProject(projectID); err != nil {
			return fmt.Errorf("project %s not found", projectID)
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if err := repofile.Write(cwd, projectID); err != nil {
			return err
		}
		fmt.Printf("Linked %s to project %s\n", repofile.FileName, projectID)
		return nil
	},
}

var projectUnlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Remove the repo-local project link",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		path := cwd + "/" + repofile.FileName
		if err := os.Remove(path); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No project linked.")
				return nil
			}
			return err
		}
		fmt.Println("Unlinked project.")
		return nil
	},
}

func init() {
	projectCreateCmd.Flags().StringP("key", "k", "", "project key (2-5 uppercase alphanumeric chars)")
	projectCreateCmd.Flags().String("store", "", "store to create the project on (\"local\" or hostname)")
	projectShowCmd.Flags().Bool("pretty", false, "render with ANSI styling")
	projectDeleteCmd.Flags().BoolP("force", "f", false, "skip confirmation")

	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectShowCmd)
	projectCmd.AddCommand(projectSetDefaultCmd)
	projectCmd.AddCommand(projectDeleteCmd)
	projectCmd.AddCommand(projectSetStoreCmd)
	projectCmd.AddCommand(projectLinkCmd)
	projectCmd.AddCommand(projectUnlinkCmd)
	rootCmd.AddCommand(projectCmd)
}

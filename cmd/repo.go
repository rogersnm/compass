package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/rogersnm/compass/internal/repofile"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repo-local project linking",
}

var repoInitCmd = &cobra.Command{
	Use:   "init [project-id]",
	Short: "Link the current directory to a project",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var projectID string
		if len(args) == 1 {
			projectID = args[0]
		} else {
			projects, err := st.ListProjects()
			if err != nil {
				return err
			}
			if len(projects) == 0 {
				return fmt.Errorf("no projects exist; create one first with: compass project create <name>")
			}
			opts := make([]huh.Option[string], len(projects))
			for i, p := range projects {
				opts[i] = huh.NewOption(fmt.Sprintf("%s  %s", p.ID, p.Name), p.ID)
			}
			if err := huh.NewSelect[string]().
				Title("Select a project").
				Options(opts...).
				Value(&projectID).
				Run(); err != nil {
				return fmt.Errorf("selection cancelled")
			}
		}

		if _, _, err := st.GetProject(projectID); err != nil {
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

var repoShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the repo-local project link",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		id, dir, err := repofile.Find(cwd)
		if err != nil {
			return err
		}
		if id == "" {
			fmt.Println("No project linked. Run: compass repo init")
			return nil
		}
		fmt.Printf("%s (from %s/%s)\n", id, dir, repofile.FileName)
		return nil
	},
}

var repoUnlinkCmd = &cobra.Command{
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
	repoCmd.AddCommand(repoInitCmd)
	repoCmd.AddCommand(repoShowCmd)
	repoCmd.AddCommand(repoUnlinkCmd)
	rootCmd.AddCommand(repoCmd)
}

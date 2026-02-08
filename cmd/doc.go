package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/rogersnm/compass/internal/editor"
	"github.com/rogersnm/compass/internal/markdown"
	"github.com/spf13/cobra"
)

var docCmd = &cobra.Command{
	Use:   "doc",
	Short: "Manage documents",
}

var docCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new document",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(cmd)
		if err != nil {
			return err
		}

		body := readStdin()

		d, err := st.CreateDocument(args[0], projectID, body)
		if err != nil {
			return err
		}
		fmt.Printf("Created document %s (%s)\n", d.Title, d.ID)
		return nil
	},
}

var docListCmd = &cobra.Command{
	Use:   "list",
	Short: "List documents",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, _ := cmd.Flags().GetString("project")
		docs, err := st.ListDocuments(projectID)
		if err != nil {
			return err
		}
		fmt.Println(markdown.RenderDocumentTable(docs))
		return nil
	},
}

var docShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show document details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		d, body, err := st.GetDocument(args[0])
		if err != nil {
			return err
		}
		fields := []string{
			markdown.RenderField("ID", d.ID),
			markdown.RenderField("Project", d.Project),
			markdown.RenderField("Created by", d.CreatedBy),
			markdown.RenderField("Created", d.CreatedAt.Format("2006-01-02 15:04:05")),
			markdown.RenderField("Updated", d.UpdatedAt.Format("2006-01-02 15:04:05")),
		}
		fmt.Print(markdown.RenderEntityHeader(d.Title, fields))
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

var docEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a document in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := st.ResolveEntityPath(args[0])
		if err != nil {
			return err
		}
		return editor.Open(path)
	},
}

func init() {
	docCreateCmd.Flags().StringP("project", "p", "", "project ID")
	docListCmd.Flags().StringP("project", "p", "", "filter by project")
	docCmd.AddCommand(docCreateCmd)
	docCmd.AddCommand(docListCmd)
	docCmd.AddCommand(docShowCmd)
	docCmd.AddCommand(docEditCmd)
	rootCmd.AddCommand(docCmd)
}

func readStdin() string {
	info, err := os.Stdin.Stat()
	if err != nil {
		return ""
	}
	// Only read if stdin is explicitly a pipe (not a terminal, not a socket)
	if info.Mode()&os.ModeNamedPipe == 0 && info.Size() == 0 {
		return ""
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return ""
	}
	return string(data)
}

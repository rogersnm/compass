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

var docUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a document",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var titlePtr, bodyPtr *string

		if cmd.Flags().Changed("title") {
			title, _ := cmd.Flags().GetString("title")
			titlePtr = &title
		}

		body := readStdin()
		if body != "" {
			bodyPtr = &body
		}

		if titlePtr == nil && bodyPtr == nil {
			return fmt.Errorf("at least one update is required (--title, stdin)")
		}

		d, err := st.UpdateDocument(args[0], titlePtr, bodyPtr)
		if err != nil {
			return err
		}
		fmt.Printf("Updated document %s\n", d.ID)
		return nil
	},
}

var docDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a document",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		d, _, err := st.GetDocument(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Document: %s (%s)\n", d.Title, d.ID)
		if err := confirmDelete(cmd, d.ID); err != nil {
			return err
		}
		if err := st.DeleteDocument(d.ID); err != nil {
			return err
		}
		fmt.Printf("Deleted document %s\n", d.ID)
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

var docCheckoutCmd = &cobra.Command{
	Use:   "checkout <id>",
	Short: "Copy a document to .compass/ in the current directory for local editing",
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

var docCheckinCmd = &cobra.Command{
	Use:   "checkin <id>",
	Short: "Write a locally edited document back to the store and remove the local copy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		localPath := fmt.Sprintf(".compass/%s.md", args[0])
		d, err := st.CheckinDocument(localPath)
		if err != nil {
			return err
		}
		fmt.Printf("Checked in document %s\n", d.ID)
		return nil
	},
}

func init() {
	docCreateCmd.Flags().StringP("project", "P", "", "project ID")
	docShowCmd.Flags().Bool("raw", false, "output raw markdown file (no ANSI styling)")
	docListCmd.Flags().StringP("project", "P", "", "filter by project")
	docUpdateCmd.Flags().String("title", "", "new title")
	docDeleteCmd.Flags().BoolP("force", "f", false, "skip confirmation")

	docCmd.AddCommand(docCreateCmd)
	docCmd.AddCommand(docListCmd)
	docCmd.AddCommand(docShowCmd)
	docCmd.AddCommand(docUpdateCmd)
	docCmd.AddCommand(docDeleteCmd)
	docCmd.AddCommand(docEditCmd)
	docCmd.AddCommand(docCheckoutCmd)
	docCmd.AddCommand(docCheckinCmd)
	rootCmd.AddCommand(docCmd)
}

// confirmDelete prompts the user to type the entity ID to confirm deletion.
// Returns nil if confirmed, error otherwise. Skipped with --force.
func confirmDelete(cmd *cobra.Command, entityID string) error {
	force, _ := cmd.Flags().GetBool("force")
	if force {
		return nil
	}
	fmt.Printf("Type %s to confirm deletion: ", entityID)
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil || input != entityID {
		return fmt.Errorf("deletion cancelled")
	}
	return nil
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

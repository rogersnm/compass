package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across all entities",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, _ := cmd.Flags().GetString("project")

		results, err := st.Search(args[0], projectID)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Println("No results found.")
			return nil
		}

		// Group by type
		grouped := map[string][]struct {
			id, title, snippet string
		}{}
		for _, r := range results {
			grouped[r.Type] = append(grouped[r.Type], struct {
				id, title, snippet string
			}{r.ID, r.Title, r.Snippet})
		}

		typeOrder := []string{"project", "epic", "task", "document"}
		for _, typ := range typeOrder {
			items, ok := grouped[typ]
			if !ok {
				continue
			}
			fmt.Printf("\n%ss:\n", capitalize(typ))
			for _, item := range items {
				fmt.Printf("  %s  %s\n", item.id, item.title)
				if item.snippet != "" {
					fmt.Printf("    %s\n", item.snippet)
				}
			}
		}
		fmt.Println()
		return nil
	},
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func init() {
	searchCmd.Flags().StringP("project", "p", "", "filter by project")
	rootCmd.AddCommand(searchCmd)
}

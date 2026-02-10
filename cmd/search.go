package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across all entities",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, _ := cmd.Flags().GetString("project")

		type result struct {
			typ, id, title, snippet string
		}
		var results []result

		if projectID != "" {
			s, err := storeForProject(projectID)
			if err != nil {
				return err
			}
			sr, err := s.Search(args[0], projectID)
			if err != nil {
				return err
			}
			for _, r := range sr {
				results = append(results, result{r.Type, r.ID, r.Title, r.Snippet})
			}
		} else {
			// Fan out across all stores
			for name, s := range reg.All() {
				sr, err := s.Search(args[0], "")
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: %s: %v\n", name, err)
					continue
				}
				for _, r := range sr {
					results = append(results, result{r.Type, r.ID, r.Title, r.Snippet})
				}
			}
		}

		if len(results) == 0 {
			fmt.Println("No results found.")
			return nil
		}

		// Group by type
		grouped := map[string][]result{}
		for _, r := range results {
			grouped[r.typ] = append(grouped[r.typ], r)
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
	searchCmd.Flags().StringP("project", "P", "", "filter by project")
	rootCmd.AddCommand(searchCmd)
}

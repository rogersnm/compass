package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/rogersnm/compass/internal/config"
	"github.com/rogersnm/compass/internal/markdown"
	"github.com/rogersnm/compass/internal/store"
	"github.com/spf13/cobra"
)

var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "Manage stores (local and cloud)",
}

var storeAddCmd = &cobra.Command{
	Use:   "add <hostname>",
	Short: "Add a store (\"local\" or cloud hostname)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := args[0]

		if arg == "local" {
			cfg.Version = 2
			cfg.LocalEnabled = true
			if cfg.DefaultStore == "" {
				cfg.DefaultStore = "local"
			}
			if err := config.Save(dataDir, cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			reg.Add("local", store.NewLocal(dataDir))
			if reg.DefaultName() == "" {
				reg.SetDefault("local")
			}
			fmt.Println("Local store enabled. Data will be stored in " + dataDir)

			// Discover existing local projects
			s, _ := reg.Get("local")
			projects, err := s.ListProjects()
			if err == nil && len(projects) > 0 {
				for _, p := range projects {
					reg.CacheProject(p.ID, "local")
				}
				fmt.Printf("Discovered %d local project(s)\n", len(projects))
			}
			return nil
		}

		// Cloud store
		hostname := arg
		storeName, _ := cmd.Flags().GetString("name")
		if storeName == "" {
			storeName = hostname
		}
		if err := config.ValidateStoreName(storeName); err != nil {
			return err
		}

		// Handle name collision
		if _, exists := cfg.Stores[storeName]; exists {
			var choice string
			if err := huh.NewSelect[string]().
				Title(fmt.Sprintf("Store %q already exists.", storeName)).
				Options(
					huh.NewOption("Keep existing and add with a different name", "rename"),
					huh.NewOption("Cancel", "cancel"),
				).
				Value(&choice).
				Run(); err != nil {
				return fmt.Errorf("cancelled")
			}
			if choice == "rename" {
				var newName string
				if err := huh.NewInput().
					Title("New store name").
					Value(&newName).
					Run(); err != nil {
					return fmt.Errorf("cancelled")
				}
				if err := config.ValidateStoreName(newName); err != nil {
					return err
				}
				storeName = newName
			} else {
				return fmt.Errorf("cancelled")
			}
		}

		apiKey, _ := cmd.Flags().GetString("api-key")
		path, _ := cmd.Flags().GetString("path")
		protocol, _ := cmd.Flags().GetString("protocol")

		sc := config.CloudStoreConfig{
			Hostname: hostname,
			Path:     path,
			Protocol: protocol,
		}

		if apiKey != "" {
			sc.APIKey = apiKey
		} else {
			// Device flow login
			// Temporarily save the store config so runDeviceFlowLogin can use it
			if cfg.Stores == nil {
				cfg.Stores = make(map[string]config.CloudStoreConfig)
			}
			cfg.Stores[storeName] = sc
			if err := runDeviceFlowLogin(storeName); err != nil {
				return err
			}
			// runDeviceFlowLogin updates cfg.Stores[storeName] with the API key
			return fetchProjectsInteractive(storeName)
		}

		if cfg.Stores == nil {
			cfg.Stores = make(map[string]config.CloudStoreConfig)
		}
		cfg.Stores[storeName] = sc
		cfg.Version = 2
		if cfg.DefaultStore == "" {
			cfg.DefaultStore = storeName
		}
		if err := config.Save(dataDir, cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		reg.Add(storeName, store.NewCloudStoreWithBase(sc.URL(), sc.APIKey))
		if reg.DefaultName() == "" {
			reg.SetDefault(storeName)
		}

		if storeName == hostname {
			fmt.Printf("Added cloud store '%s'\n", storeName)
		} else {
			fmt.Printf("Added cloud store '%s' (%s)\n", storeName, hostname)
		}

		return fetchProjectsInteractive(storeName)
	},
}

var storeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured stores",
	RunE: func(cmd *cobra.Command, args []string) error {
		names := cfg.StoreNames()
		if len(names) == 0 {
			fmt.Println("No stores configured. Run: compass store add local")
			return nil
		}
		rows := make([][]string, len(names))
		for i, name := range names {
			def := ""
			if name == cfg.DefaultStore {
				def = "*"
			}
			hostname := ""
			if sc, ok := cfg.Stores[name]; ok {
				hostname = sc.Hostname
			}
			rows[i] = []string{name, hostname, def}
		}
		fmt.Println(markdown.RenderStoreTable(rows))
		return nil
	},
}

var storeRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		force, _ := cmd.Flags().GetBool("force")

		// Count affected projects
		var affected []string
		for key, storeName := range cfg.Projects {
			if storeName == name {
				affected = append(affected, key)
			}
		}

		if len(affected) > 0 && !force {
			msg := fmt.Sprintf("This will remove %d project mapping(s) (%s). Continue?", len(affected), joinKeys(affected))
			var confirm bool
			if err := huh.NewConfirm().Title(msg).Value(&confirm).Run(); err != nil || !confirm {
				return fmt.Errorf("removal cancelled")
			}
		}

		if name == "local" {
			cfg.LocalEnabled = false
		} else {
			delete(cfg.Stores, name)
		}

		// Prune project mappings
		for _, key := range affected {
			delete(cfg.Projects, key)
		}

		if cfg.DefaultStore == name {
			cfg.DefaultStore = ""
		}

		if err := config.Save(dataDir, cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Printf("Removed store %s\n", name)
		return nil
	},
}

var storeSetDefaultCmd = &cobra.Command{
	Use:   "set-default <name>",
	Short: "Set the default store for new projects",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if _, err := reg.Get(name); err != nil {
			return err
		}
		cfg.DefaultStore = name
		reg.SetDefault(name)
		if err := config.Save(dataDir, cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Printf("Default store set to %s\n", name)
		return nil
	},
}

var storeFetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch and cache projects from stores",
	RunE: func(cmd *cobra.Command, args []string) error {
		storeName, _ := cmd.Flags().GetString("store")
		all, _ := cmd.Flags().GetBool("all")

		if storeName != "" {
			if all {
				return fetchProjectsAll(storeName)
			}
			return fetchProjectsInteractive(storeName)
		}

		// Fetch from all stores
		for _, name := range cfg.StoreNames() {
			if all {
				if err := fetchProjectsAll(name); err != nil {
					fmt.Printf("warning: %s: %v\n", name, err)
				}
			} else {
				if err := fetchProjectsInteractive(name); err != nil {
					fmt.Printf("warning: %s: %v\n", name, err)
				}
			}
		}
		return nil
	},
}

func fetchProjectsInteractive(storeName string) error {
	s, err := reg.Get(storeName)
	if err != nil {
		return err
	}

	projects, err := s.ListProjects()
	if err != nil {
		return fmt.Errorf("fetching projects from %s: %w", storeName, err)
	}

	if len(projects) == 0 {
		fmt.Printf("No projects found on %s\n", storeName)
		return nil
	}

	// Build options, noting already-cached ones
	opts := make([]huh.Option[string], len(projects))
	for i, p := range projects {
		label := fmt.Sprintf("%s  %s", p.ID, p.Name)
		if existing, ok := cfg.Projects[p.ID]; ok {
			if existing == storeName {
				label += " (already cached)"
			} else {
				label += fmt.Sprintf(" (mapped to %s)", existing)
			}
		}
		opts[i] = huh.NewOption(label, p.ID)
	}

	var selected []string
	if err := huh.NewMultiSelect[string]().
		Title(fmt.Sprintf("Projects on %s (space to select, enter to confirm)", storeName)).
		Options(opts...).
		Value(&selected).
		Run(); err != nil {
		return nil // user cancelled
	}

	added := 0
	for _, key := range selected {
		if existing, ok := cfg.Projects[key]; ok && existing != storeName {
			// Collision; prompt
			var remap bool
			msg := fmt.Sprintf("%s is mapped to store '%s'. Remap to '%s'?", key, existing, storeName)
			if err := huh.NewConfirm().Title(msg).Value(&remap).Run(); err != nil || !remap {
				continue
			}
		}
		reg.CacheProject(key, storeName)
		added++
	}

	fmt.Printf("Added %d project(s) from %s\n", added, storeName)
	return nil
}

func fetchProjectsAll(storeName string) error {
	s, err := reg.Get(storeName)
	if err != nil {
		return err
	}

	projects, err := s.ListProjects()
	if err != nil {
		return fmt.Errorf("fetching projects from %s: %w", storeName, err)
	}

	added := 0
	for _, p := range projects {
		if existing, ok := cfg.Projects[p.ID]; ok && existing != storeName {
			fmt.Printf("warning: %s already mapped to %s, skipping\n", p.ID, existing)
			continue
		}
		reg.CacheProject(p.ID, storeName)
		added++
	}

	fmt.Printf("Added %d project(s) from %s\n", added, storeName)
	return nil
}

func joinKeys(keys []string) string {
	if len(keys) <= 3 {
		s := ""
		for i, k := range keys {
			if i > 0 {
				s += ", "
			}
			s += k
		}
		return s
	}
	return fmt.Sprintf("%s, %s, ... +%d more", keys[0], keys[1], len(keys)-2)
}

func init() {
	storeAddCmd.Flags().String("name", "", "store name (defaults to hostname)")
	storeAddCmd.Flags().String("api-key", "", "API key (skip device flow)")
	storeAddCmd.Flags().String("path", "", "API path override (default: /api/v1)")
	storeAddCmd.Flags().String("protocol", "", "protocol override (default: https)")

	storeRemoveCmd.Flags().BoolP("force", "f", false, "skip confirmation")

	storeFetchCmd.Flags().String("store", "", "fetch from a specific store")
	storeFetchCmd.Flags().Bool("all", false, "non-interactive, add all projects")

	storeCmd.AddCommand(storeAddCmd)
	storeCmd.AddCommand(storeListCmd)
	storeCmd.AddCommand(storeRemoveCmd)
	storeCmd.AddCommand(storeSetDefaultCmd)
	storeCmd.AddCommand(storeFetchCmd)
	rootCmd.AddCommand(storeCmd)
}

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	mtp "github.com/modeltoolsprotocol/go-sdk"
	"github.com/rogersnm/compass/internal/config"
	"github.com/rogersnm/compass/internal/repofile"
	"github.com/rogersnm/compass/internal/store"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	dataDir string
	reg     *store.Registry
	cfg     *config.Config
)

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".compass")
	}
	return filepath.Join(home, ".compass")
}

var rootCmd = &cobra.Command{
	Use:     "compass",
	Short:   "Markdown-native task and document tracking",
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return fmt.Errorf("creating data directory: %w", err)
		}

		var err error
		cfg, err = config.Load(dataDir)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Migrate v1 config to v2 on disk
		if cfg.Version == 2 && cfg.Mode == "" && cfg.Cloud == nil {
			// already v2, good
		} else if cfg.Version < 2 && (cfg.Mode != "" || cfg.Cloud != nil || cfg.DefaultProject != "") {
			// Migration happened in Load; persist it
			if err := config.Save(dataDir, cfg); err != nil {
				return fmt.Errorf("saving migrated config: %w", err)
			}
		}

		// Build registry
		reg = store.NewRegistry(cfg, dataDir)

		if cfg.LocalEnabled {
			reg.Add("local", store.NewLocal(dataDir))
		}
		for storeName, sc := range cfg.Stores {
			reg.Add(storeName, store.NewCloudStoreWithBase(sc.URL(), sc.APIKey))
		}

		// Store commands work without configured stores
		if cmd.Name() == "store" || (cmd.Parent() != nil && cmd.Parent().Name() == "store") {
			return nil
		}
		// go command just prints a skill file, no store needed
		if cmd.Name() == "go" {
			return nil
		}
		// Config commands (legacy, kept for backwards compat during transition)
		if cmd.Name() == "config" || (cmd.Parent() != nil && cmd.Parent().Name() == "config") {
			return nil
		}

		// First-run setup if no stores configured
		if reg.IsEmpty() {
			return runSetupPrompt(cmd)
		}
		return nil
	},
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", defaultDataDir(), "data directory path")

	mtpOpts := &mtp.DescribeOptions{
		Commands: map[string]*mtp.CommandAnnotation{
			"project create": {
				Examples: []mtp.Example{
					{Description: "Create a new project", Command: "compass project create \"My Project\""},
					{Description: "Create with explicit key", Command: "compass project create \"My Project\" --key MYPR"},
				},
			},
			"doc create": {
				Stdin: &mtp.IODescriptor{
					ContentType: "text/markdown",
					Description: "Markdown body content for the document",
				},
				Examples: []mtp.Example{
					{Description: "Create doc with piped content", Command: "echo '# Design' | compass doc create \"Design Doc\" --project AUTH"},
				},
			},
			"doc download": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "Local file path where the document was downloaded. Use download+upload instead of piping to 'doc update' when you need to make surgical edits without rewriting the entire file.",
				},
				Examples: []mtp.Example{
					{Description: "Download a document for local editing", Command: "compass doc download AUTH-DXXXXX"},
				},
			},
			"doc upload": {
				Examples: []mtp.Example{
					{Description: "Upload a locally edited document", Command: "compass doc upload AUTH-DXXXXX"},
				},
			},
			"doc update": {
				Stdin: &mtp.IODescriptor{
					ContentType: "text/markdown",
					Description: "New markdown body content for the document",
				},
				Examples: []mtp.Example{
					{Description: "Update document title", Command: "compass doc update AUTH-DXXXXX --title \"New Title\""},
					{Description: "Update document body", Command: "echo '# Updated' | compass doc update AUTH-DXXXXX"},
				},
			},
			"doc delete": {
				Examples: []mtp.Example{
					{Description: "Delete a document (interactive confirm)", Command: "compass doc delete AUTH-DXXXXX"},
					{Description: "Delete a document (skip confirm)", Command: "compass doc delete AUTH-DXXXXX --force"},
				},
			},
			"task create": {
				Stdin: &mtp.IODescriptor{
					ContentType: "text/markdown",
					Description: "Markdown body content for the task",
				},
				Examples: []mtp.Example{
					{Description: "Create a task with dependencies", Command: "compass task create \"Login\" --project AUTH --parent-epic AUTH-TXXXXX --depends-on AUTH-TAAAAA,AUTH-TBBBBB"},
					{Description: "Create an epic", Command: "compass task create \"Auth\" --project AUTH --type epic"},
					{Description: "Create a high-priority task", Command: "compass task create \"Urgent fix\" --project AUTH --priority 0"},
				},
			},
			"task start": {
				Examples: []mtp.Example{
					{Description: "Start a task", Command: "compass task start AUTH-TXXXXX"},
				},
			},
			"task close": {
				Examples: []mtp.Example{
					{Description: "Close a task", Command: "compass task close AUTH-TXXXXX"},
				},
			},
			"task ready": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "Next ready task or table of all ready tasks with --all",
				},
				Examples: []mtp.Example{
					{Description: "Show next ready task", Command: "compass task ready --project AUTH"},
					{Description: "Show all ready tasks", Command: "compass task ready --project AUTH --all"},
				},
			},
			"task update": {
				Stdin: &mtp.IODescriptor{
					ContentType: "text/markdown",
					Description: "New markdown body content for the task",
				},
				Examples: []mtp.Example{
					{Description: "Update task title", Command: "compass task update AUTH-TXXXXX --title \"New Title\""},
					{Description: "Update task body", Command: "echo '# Updated' | compass task update AUTH-TXXXXX"},
					{Description: "Set task priority", Command: "compass task update AUTH-TXXXXX --priority 1"},
				},
			},
			"task delete": {
				Examples: []mtp.Example{
					{Description: "Delete a task (interactive confirm)", Command: "compass task delete AUTH-TXXXXX"},
					{Description: "Delete a task (skip confirm)", Command: "compass task delete AUTH-TXXXXX --force"},
				},
			},
			"task download": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "Local file path where the task was downloaded. Use download+upload instead of piping to 'task update' when you need to make surgical edits without rewriting the entire file.",
				},
				Examples: []mtp.Example{
					{Description: "Download a task for local editing", Command: "compass task download AUTH-TXXXXX"},
				},
			},
			"task upload": {
				Examples: []mtp.Example{
					{Description: "Upload a locally edited task", Command: "compass task upload AUTH-TXXXXX"},
				},
			},
			"task list": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "Table of tasks with ID, title, status (with blocked annotation), and project",
				},
			},
			"task graph": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "ASCII tree visualization of the task dependency DAG",
				},
			},
			"search": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "Search results grouped by entity type with ID, title, and snippet",
				},
				Examples: []mtp.Example{
					{Description: "Search across all entities", Command: "compass search \"authentication\""},
				},
			},
			"project set-store": {
				Examples: []mtp.Example{
					{Description: "Reassign project to a different store", Command: "compass project set-store AUTH compasscloud.io"},
				},
			},
			"project link": {
				Examples: []mtp.Example{
					{Description: "Link current directory to a project", Command: "compass project link AUTH"},
				},
			},
			"project unlink": {
				Examples: []mtp.Example{
					{Description: "Remove repo-local project link", Command: "compass project unlink"},
				},
			},
			"store add": {
				Examples: []mtp.Example{
					{Description: "Enable local store", Command: "compass store add local"},
					{Description: "Add cloud store", Command: "compass store add compasscloud.io"},
					{Description: "Add cloud store with API key", Command: "compass store add compasscloud.io --api-key cpk_xxx"},
				},
			},
			"store list": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "Table of configured stores with default indicator",
				},
			},
			"store remove": {
				Examples: []mtp.Example{
					{Description: "Remove a store", Command: "compass store remove compasscloud.io"},
					{Description: "Remove without confirmation", Command: "compass store remove compasscloud.io --force"},
				},
			},
			"store set-default": {
				Examples: []mtp.Example{
					{Description: "Set default store for new projects", Command: "compass store set-default local"},
				},
			},
			"store fetch": {
				Examples: []mtp.Example{
					{Description: "Fetch projects from all stores", Command: "compass store fetch"},
					{Description: "Fetch from one store non-interactively", Command: "compass store fetch --store compasscloud.io --all"},
				},
			},
		},
	}

	mtp.WithDescribe(rootCmd, mtpOpts)
}

func Execute() error {
	return rootCmd.Execute()
}

const signupURL = "https://compasscloud.io/signup"

// runSetupPrompt presents the interactive first-run prompt.
func runSetupPrompt(cmd *cobra.Command) error {
	var choice string
	err := huh.NewSelect[string]().
		Title("Welcome to Compass! No stores configured.").
		Options(
			huh.NewOption("Log in to Compass Cloud", "login"),
			huh.NewOption("Create an account at "+signupURL, "signup"),
			huh.NewOption("Use local mode (offline, file-based)", "local"),
		).
		Value(&choice).
		Run()
	if err != nil {
		return fmt.Errorf("run 'compass store add local' or 'compass store add <hostname>' to get started")
	}

	switch choice {
	case "login":
		fmt.Println()
		if cfg.Stores == nil {
			cfg.Stores = make(map[string]config.CloudStoreConfig)
		}
		cfg.Stores["compasscloud.io"] = config.CloudStoreConfig{Hostname: "compasscloud.io"}
		return runDeviceFlowLogin("compasscloud.io")
	case "signup":
		openBrowser(signupURL)
		fmt.Println("Opening browser... after signing up, run: compass store add compasscloud.io")
		return fmt.Errorf("setup incomplete")
	case "local":
		cfg.Version = 2
		cfg.LocalEnabled = true
		cfg.DefaultStore = "local"
		if err := config.Save(dataDir, cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		reg.Add("local", store.NewLocal(dataDir))
		reg.SetDefault("local")
		fmt.Println("Local mode enabled. Data will be stored in " + dataDir)
		return nil
	default:
		return fmt.Errorf("run 'compass store add local' or 'compass store add <hostname>' to get started")
	}
}

// runDeviceFlowLogin performs the device flow login and saves the API key.
// storeName is the key in cfg.Stores; hostname is read from the config entry.
func runDeviceFlowLogin(storeName string) error {
	sc, ok := cfg.Stores[storeName]
	if !ok {
		return fmt.Errorf("store %q not found in config", storeName)
	}
	server := sc.URL()

	resp, err := http.Post(server+"/auth/device", "application/json", nil)
	if err != nil {
		return fmt.Errorf("requesting device code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("device code request failed with status %d", resp.StatusCode)
	}

	var deviceResp struct {
		Data struct {
			DeviceCode      string `json:"device_code"`
			UserCode        string `json:"user_code"`
			VerificationURI string `json:"verification_uri"`
			ExpiresIn       int    `json:"expires_in"`
			Interval        int    `json:"interval"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return fmt.Errorf("decoding device response: %w", err)
	}
	d := deviceResp.Data

	verifyURL := d.VerificationURI
	if verifyURL != "" && verifyURL[0] == '/' {
		verifyURL = server + verifyURL
	}
	if d.UserCode != "" {
		if strings.Contains(verifyURL, "?") {
			verifyURL += "&user_code=" + d.UserCode
		} else {
			verifyURL += "?user_code=" + d.UserCode
		}
	}

	fmt.Printf("Open this URL in your browser:\n  %s\n\n", verifyURL)
	fmt.Printf("Enter code: %s\n\n", d.UserCode)
	fmt.Println("Waiting for authorization...")

	openBrowser(verifyURL)

	interval := time.Duration(d.Interval) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(d.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		tokenResp, err := pollToken(server, d.DeviceCode)
		if err != nil {
			return err
		}

		if tokenResp.Status == "pending" {
			continue
		}

		if tokenResp.Status == "authorized" {
			sc.APIKey = tokenResp.APIKey
			if cfg.Stores == nil {
				cfg.Stores = make(map[string]config.CloudStoreConfig)
			}
			cfg.Stores[storeName] = sc
			if cfg.DefaultStore == "" {
				cfg.DefaultStore = storeName
			}
			cfg.Version = 2
			if err := config.Save(dataDir, cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			reg.Add(storeName, store.NewCloudStoreWithBase(sc.URL(), sc.APIKey))
			if reg.DefaultName() == "" {
				reg.SetDefault(storeName)
			}

			orgInfo := ""
			if tokenResp.OrgName != "" {
				orgInfo = fmt.Sprintf(" (%s)", tokenResp.OrgName)
			}
			fmt.Printf("Authenticated%s\n", orgInfo)
			return nil
		}

		return fmt.Errorf("unexpected status: %s", tokenResp.Status)
	}

	return fmt.Errorf("authorization timed out")
}

var configCmd = &cobra.Command{
	Use:        "config",
	Short:      "Configure Compass (deprecated, use 'compass store')",
	Deprecated: "use 'compass store' instead",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetupPrompt(cmd)
	},
}

var configLoginCmd = &cobra.Command{
	Use:        "login",
	Short:      "Authenticate with Compass Cloud (deprecated, use 'compass store add')",
	Deprecated: "use 'compass store add <hostname>' instead",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeviceFlowLogin("compasscloud.io")
	},
}

var configLogoutCmd = &cobra.Command{
	Use:        "logout",
	Short:      "Log out of Compass Cloud (deprecated, use 'compass store remove')",
	Deprecated: "use 'compass store remove <hostname>' instead",
	RunE: func(cmd *cobra.Command, args []string) error {
		delete(cfg.Stores, "compasscloud.io")
		if cfg.DefaultStore == "compasscloud.io" {
			cfg.DefaultStore = ""
		}
		if err := config.Save(dataDir, cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Println("Logged out")
		return nil
	},
}

var configStatusCmd = &cobra.Command{
	Use:        "status",
	Short:      "Show current configuration (deprecated, use 'compass store list')",
	Deprecated: "use 'compass store list' instead",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.LocalEnabled {
			fmt.Println("Local store: enabled")
			fmt.Println("Data: " + dataDir)
		}
		for name, sc := range cfg.Stores {
			if name == sc.Hostname {
				fmt.Printf("Cloud store: %s\n", name)
			} else {
				fmt.Printf("Cloud store: %s (%s)\n", name, sc.Hostname)
			}
			fmt.Printf("  API key: %s...\n", sc.APIKey[:min(8, len(sc.APIKey))])
		}
		if cfg.DefaultStore != "" {
			fmt.Printf("Default store: %s\n", cfg.DefaultStore)
		}
		return nil
	},
}

type tokenResult struct {
	Status  string
	APIKey  string
	OrgName string
}

func pollToken(server, deviceCode string) (*tokenResult, error) {
	body := fmt.Sprintf(`{"device_code":"%s"}`, deviceCode)
	resp, err := http.Post(
		server+"/auth/device/token",
		"application/json",
		strings.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("polling token: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		Data struct {
			Status string `json:"status"`
			APIKey string `json:"api_key"`
			Org    *struct {
				Slug string `json:"slug"`
				Name string `json:"name"`
			} `json:"org"`
		} `json:"data"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}

	if tokenResp.Error != nil {
		return nil, fmt.Errorf("token error: %s", tokenResp.Error.Message)
	}

	result := &tokenResult{
		Status: tokenResp.Data.Status,
		APIKey: tokenResp.Data.APIKey,
	}
	if tokenResp.Data.Org != nil {
		result.OrgName = tokenResp.Data.Org.Name
	}
	return result, nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

func init() {
	configCmd.AddCommand(configLoginCmd)
	configCmd.AddCommand(configLogoutCmd)
	configCmd.AddCommand(configStatusCmd)
	rootCmd.AddCommand(configCmd)
}

// storeForProject resolves a project key to its store.
func storeForProject(projectKey string) (store.Store, error) {
	s, _, err := reg.ForProject(projectKey)
	return s, err
}

// storeForEntity resolves an entity ID to its store.
func storeForEntity(entityID string) (store.Store, error) {
	s, _, err := reg.ForEntity(entityID)
	return s, err
}

// resolveProject returns the project ID from the flag, repo-local file, or global default.
func resolveProject(cmd *cobra.Command) (string, error) {
	p, _ := cmd.Flags().GetString("project")
	if p != "" {
		return p, nil
	}
	if cwd, err := os.Getwd(); err == nil {
		if rp, _, _ := repofile.Find(cwd); rp != "" {
			return rp, nil
		}
	}
	return "", fmt.Errorf("--project is required (or link a repo with: compass project link)")
}

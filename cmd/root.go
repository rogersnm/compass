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
	st      store.Store
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

		// Config commands work without a configured store
		if cmd.Name() == "config" || (cmd.Parent() != nil && cmd.Parent().Name() == "config") {
			st = store.NewLocal(dataDir)
			return nil
		}

		if cfg.Mode == "local" {
			st = store.NewLocal(dataDir)
		} else if cfg.Cloud != nil && cfg.Cloud.APIKey != "" {
			st = store.NewCloudStore(cfg.Cloud.APIKey)
		} else {
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
			"doc checkout": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "Local file path where the document was checked out (e.g. .compass/AUTH-DXXXXX.md)",
				},
				Examples: []mtp.Example{
					{Description: "Checkout a document for local editing", Command: "compass doc checkout AUTH-DXXXXX"},
				},
			},
			"doc checkin": {
				Examples: []mtp.Example{
					{Description: "Check in a locally edited document", Command: "compass doc checkin AUTH-DXXXXX"},
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
					{Description: "Create a task with dependencies", Command: "compass task create \"Login\" --project AUTH --epic AUTH-TXXXXX --depends-on AUTH-TAAAAA,AUTH-TBBBBB"},
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
			"task checkout": {
				Stdout: &mtp.IODescriptor{
					ContentType: "text/plain",
					Description: "Local file path where the task was checked out (e.g. .compass/AUTH-TXXXXX.md)",
				},
				Examples: []mtp.Example{
					{Description: "Checkout a task for local editing", Command: "compass task checkout AUTH-TXXXXX"},
				},
			},
			"task checkin": {
				Examples: []mtp.Example{
					{Description: "Check in a locally edited task", Command: "compass task checkin AUTH-TXXXXX"},
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
		},
	}

	mtp.WithDescribe(rootCmd, mtpOpts)
}

func Execute() error {
	return rootCmd.Execute()
}

const signupURL = "https://compasscloud.io/signup"

// runSetupPrompt presents the interactive mode selection prompt.
func runSetupPrompt(cmd *cobra.Command) error {
	var choice string
	err := huh.NewSelect[string]().
		Title("Welcome to Compass!").
		Options(
			huh.NewOption("Log in to Compass Cloud", "login"),
			huh.NewOption("Create an account at "+signupURL, "signup"),
			huh.NewOption("Use local mode (offline, file-based)", "local"),
		).
		Value(&choice).
		Run()
	if err != nil {
		return fmt.Errorf("run 'compass config login' to get started")
	}

	switch choice {
	case "login":
		fmt.Println()
		return configLoginCmd.RunE(cmd, nil)
	case "signup":
		openBrowser(signupURL)
		fmt.Println("Opening browser... after signing up, run: compass config login")
		return fmt.Errorf("setup incomplete")
	case "local":
		cfg.Mode = "local"
		if err := config.Save(dataDir, cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		st = store.NewLocal(dataDir)
		fmt.Println("Local mode enabled. Data will be stored in " + dataDir)
		return nil
	default:
		return fmt.Errorf("run 'compass config login' to get started")
	}
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure Compass (mode, authentication)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetupPrompt(cmd)
	},
}

var configLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Compass Cloud via device flow",
	RunE: func(cmd *cobra.Command, args []string) error {
		server := store.CloudAPIBase

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
				cfg.Cloud = &config.CloudConfig{
					APIKey: tokenResp.APIKey,
				}
				cfg.Mode = ""
				if err := config.Save(dataDir, cfg); err != nil {
					return fmt.Errorf("saving config: %w", err)
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
	},
}

var configLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of Compass Cloud",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.Cloud = nil
		if err := config.Save(dataDir, cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Println("Logged out")
		return nil
	},
}

var configStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current configuration and authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Mode == "local" {
			fmt.Println("Mode: local")
			fmt.Println("Data: " + dataDir)
			return nil
		}
		if cfg.Cloud == nil || cfg.Cloud.APIKey == "" {
			fmt.Println("Not configured. Run: compass config")
			return nil
		}
		fmt.Println("Mode: cloud")
		fmt.Printf("Server: %s\n", store.CloudAPIBase)
		fmt.Printf("API key: %s...\n", cfg.Cloud.APIKey[:min(8, len(cfg.Cloud.APIKey))])
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
	if cfg != nil && cfg.DefaultProject != "" {
		return cfg.DefaultProject, nil
	}
	return "", fmt.Errorf("--project is required (or set a default with: compass project set-default <id>, or link a repo with: compass project link)")
}

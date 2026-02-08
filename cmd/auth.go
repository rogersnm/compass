package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/rogersnm/compass/internal/config"
	"github.com/rogersnm/compass/internal/store"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with compass cloud",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with compass cloud via device flow",
	RunE: func(cmd *cobra.Command, args []string) error {
		server := store.CloudAPIBase

		// Step 1: Request device code
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

		// Try to open browser
		openBrowser(verifyURL)

		// Step 2: Poll for token
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

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of compass cloud",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.Cloud = nil
		if err := config.Save(dataDir, cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Println("Logged out")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Cloud == nil || cfg.Cloud.APIKey == "" {
			fmt.Println("Not authenticated (local mode)")
			return nil
		}
		fmt.Printf("Authenticated to %s\n", store.CloudAPIBase)
		fmt.Printf("API key: %s...\n", cfg.Cloud.APIKey[:min(8, len(cfg.Cloud.APIKey))])
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}

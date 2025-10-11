package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/auth"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"github.com/spf13/cobra"
)

const (
	gitHubAPIHost = "github.com"
)

func NewDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "debug",
		Short:  "Debug GitHub App authentication",
		Hidden: true,
	}

	cmd.AddCommand(newListInstallationsCmd())
	cmd.AddCommand(newListInstallationReposCmd())
	return cmd
}

func newListInstallationsCmd() *cobra.Command {
	var appID int64

	cmd := &cobra.Command{
		Use:     "list-installations",
		Short:   "List all installations for a GitHub App",
		Long:    "Lists all installations for a GitHub App using the configured private key.",
		Example: "  gh app-auth debug list-installations --app-id 2083241",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			apps := make([]*config.GitHubApp, 0, len(cfg.GitHubApps))
			if cmd.Flags().Changed("app-id") {
				for i := range cfg.GitHubApps {
					if cfg.GitHubApps[i].AppID == appID {
						apps = append(apps, &cfg.GitHubApps[i])
						break
					}
				}
				if len(apps) == 0 {
					return fmt.Errorf("app with ID %d not found in configuration", appID)
				}
			} else {
				for i := range cfg.GitHubApps {
					apps = append(apps, &cfg.GitHubApps[i])
				}
				if len(apps) == 0 {
					fmt.Println("No GitHub Apps configured. Run 'gh app-auth setup' to add one.")
					return nil
				}
			}

			for idx, app := range apps {
				fmt.Printf("=== %s (App ID %d) ===\n", appDisplayName(app), app.AppID)

				authenticator := auth.NewAuthenticator()
				jwtToken, err := authenticator.GenerateJWTForApp(app)
				if err != nil {
					if cmd.Flags().Changed("app-id") {
						return fmt.Errorf("failed to generate JWT for app %d: %w", app.AppID, err)
					}
					fmt.Printf("  Error: %v\n\n", err)
					continue
				}

				fmt.Println("  JWT generated")

				installations, err := listInstallations(jwtToken)
				if err != nil {
					if cmd.Flags().Changed("app-id") {
						return fmt.Errorf("failed to list installations for app %d: %w", app.AppID, err)
					}
					fmt.Printf("  Error listing installations: %v\n\n", err)
					continue
				}

				if len(installations) == 0 {
					fmt.Println("  No installations found for this app")
				} else {
					fmt.Printf("  Found %d installation(s):\n", len(installations))
					for _, inst := range installations {
						fmt.Printf("    Installation ID: %d\n", inst.ID)
						fmt.Printf("      Account: %s (%s)\n", inst.Account.Login, inst.Account.Type)
						fmt.Printf("      Repository Selection: %s\n", inst.RepositorySelection)
						if inst.TargetType != "" {
							fmt.Printf("      Target Type: %s\n", inst.TargetType)
						}
					}
				}

				if idx < len(apps)-1 {
					fmt.Println()
				}
			}

			return nil
		},
	}

	cmd.Flags().Int64Var(&appID, "app-id", 0, "GitHub App ID (optional)")

	return cmd
}

func newListInstallationReposCmd() *cobra.Command {
	var appID int64

	cmd := &cobra.Command{
		Use:     "list-repositories",
		Short:   "List repositories accessible to an installation",
		Long:    "Lists repositories accessible to an installation using the configured GitHub App.",
		Example: "  gh app-auth debug list-repositories --app-id 2083241",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			apps := make([]*config.GitHubApp, 0, len(cfg.GitHubApps))
			if cmd.Flags().Changed("app-id") {
				for i := range cfg.GitHubApps {
					if cfg.GitHubApps[i].AppID == appID {
						apps = append(apps, &cfg.GitHubApps[i])
						break
					}
				}
				if len(apps) == 0 {
					return fmt.Errorf("app with ID %d not found in configuration", appID)
				}
			} else {
				for i := range cfg.GitHubApps {
					apps = append(apps, &cfg.GitHubApps[i])
				}
				if len(apps) == 0 {
					fmt.Println("No GitHub Apps configured. Run 'gh app-auth setup' to add one.")
					return nil
				}
			}

			for idx, app := range apps {
				fmt.Printf("=== %s (App ID %d) ===\n", appDisplayName(app), app.AppID)

				if app.InstallationID == 0 {
					err := fmt.Errorf("app %d does not have an installation_id configured", app.AppID)
					if cmd.Flags().Changed("app-id") {
						return err
					}
					fmt.Printf("  Error: %v\n\n", err)
					continue
				}

				if len(app.Patterns) == 0 {
					err := fmt.Errorf("app %d has no patterns configured", app.AppID)
					if cmd.Flags().Changed("app-id") {
						return err
					}
					fmt.Printf("  Error: %v\n\n", err)
					continue
				}

				host := extractHostFromPattern(app.Patterns[0])
				if host == "" {
					err := fmt.Errorf("unable to determine host from pattern %q", app.Patterns[0])
					if cmd.Flags().Changed("app-id") {
						return err
					}
					fmt.Printf("  Error: %v\n\n", err)
					continue
				}

				authenticator := auth.NewAuthenticator()
				jwtToken, err := authenticator.GenerateJWTForApp(app)
				if err != nil {
					if cmd.Flags().Changed("app-id") {
						return fmt.Errorf("failed to generate JWT for app %d: %w", app.AppID, err)
					}
					fmt.Printf("  Error: %v\n\n", err)
					continue
				}

				repoURL := fmt.Sprintf("https://%s", host)
				installationToken, err := authenticator.GetInstallationToken(jwtToken, app.InstallationID, repoURL)
				if err != nil {
					if cmd.Flags().Changed("app-id") {
						return fmt.Errorf("failed to obtain installation token for app %d: %w", app.AppID, err)
					}
					fmt.Printf("  Error obtaining installation token: %v\n\n", err)
					continue
				}

				repos, err := listInstallationRepositories(installationToken, host)
				if err != nil {
					if cmd.Flags().Changed("app-id") {
						return err
					}
					fmt.Printf("  Error listing repositories: %v\n\n", err)
					continue
				}

				if len(repos) == 0 {
					fmt.Println("  No repositories returned for this installation")
				} else {
					fmt.Printf("  Found %d repositories:\n", len(repos))
					for _, repo := range repos {
						privacy := "public"
						if repo.Private {
							privacy = "private"
						}
						fmt.Printf("    - %s (%s)\n", repo.FullName, privacy)
						if repo.Description != "" {
							fmt.Printf("      %s\n", repo.Description)
						}
						fmt.Printf("      URL: %s\n", repo.HTMLURL)
					}
				}

				if idx < len(apps)-1 {
					fmt.Println()
				}
			}

			return nil
		},
	}

	cmd.Flags().Int64Var(&appID, "app-id", 0, "GitHub App ID (optional)")

	return cmd
}

type installation struct {
	ID                  int64   `json:"id"`
	Account             account `json:"account"`
	RepositorySelection string  `json:"repository_selection"`
	TargetType          string  `json:"target_type"`
}

type account struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

func listInstallations(jwtToken string) ([]installation, error) {
	apiURL := "https://api.github.com/app/installations"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var installations []installation
	if err := json.NewDecoder(resp.Body).Decode(&installations); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return installations, nil
}

type installationRepository struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Private     bool   `json:"private"`
	Description string `json:"description"`
	HTMLURL     string `json:"html_url"`
}

func listInstallationRepositories(token, host string) ([]installationRepository, error) {
	apiURL := fmt.Sprintf("https://%s/api/v3/installation/repositories", host)
	if host == gitHubAPIHost {
		apiURL = "https://api.github.com/installation/repositories"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Repositories []installationRepository `json:"repositories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return payload.Repositories, nil
}

func extractHostFromPattern(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	pattern = strings.TrimPrefix(pattern, "https://")
	pattern = strings.TrimPrefix(pattern, "http://")
	if pattern == "" {
		return ""
	}
	parts := strings.Split(pattern, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func appDisplayName(app *config.GitHubApp) string {
	name := strings.TrimSpace(app.Name)
	if name == "" {
		return fmt.Sprintf("GitHub App %d", app.AppID)
	}
	return name
}

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
	var (
		appID    int64
		clientID string
	)

	cmd := &cobra.Command{
		Use:   "list-installations",
		Short: "List all installations for a GitHub App",
		Long:  "Lists all installations for a GitHub App using the configured private key.",
		Example: `  gh app-auth debug list-installations --app-id 2083241
  gh app-auth debug list-installations --client-id Iv1.your_client_id`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			appIDSet := cmd.Flags().Changed("app-id")
			clientIDSet := cmd.Flags().Changed("client-id")
			apps, err := selectDebugApps(cfg, appID, clientID, appIDSet, clientIDSet)
			if err != nil {
				return err
			}
			if len(apps) == 0 {
				fmt.Println("No GitHub Apps configured. Run 'gh app-auth setup' to add one.")
				return nil
			}

			filtered := appIDSet || clientIDSet

			for idx, app := range apps {
				fmt.Printf("=== %s (App ID: %s) ===\n", appDisplayName(app), app.GetIdentifier())

				authenticator := auth.NewAuthenticator()
				jwtToken, err := authenticator.GenerateJWTForApp(app)
				if err != nil {
					if filtered {
						return fmt.Errorf("failed to generate JWT for app %s: %w", app.GetIdentifier(), err)
					}
					fmt.Printf("  Error: %v\n\n", err)
					continue
				}

				fmt.Println("  JWT generated")

				installations, err := listInstallations(jwtToken)
				if err != nil {
					if filtered {
						return fmt.Errorf("failed to list installations for app %s: %w", app.GetIdentifier(), err)
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

	cmd.Flags().Int64Var(&appID, "app-id", 0, "GitHub App ID (optional, mutually exclusive with --client-id)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "GitHub App Client ID (optional, mutually exclusive with --app-id)")

	return cmd
}

func newListInstallationReposCmd() *cobra.Command {
	var (
		appID    int64
		clientID string
	)

	cmd := &cobra.Command{
		Use:   "list-repositories",
		Short: "List repositories accessible to an installation",
		Long:  "Lists repositories accessible to an installation using the configured GitHub App.",
		Example: `  gh app-auth debug list-repositories --app-id 2083241
  gh app-auth debug list-repositories --client-id Iv1.your_client_id`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			appIDSet := cmd.Flags().Changed("app-id")
			clientIDSet := cmd.Flags().Changed("client-id")
			apps, err := selectDebugApps(cfg, appID, clientID, appIDSet, clientIDSet)
			if err != nil {
				return err
			}
			if len(apps) == 0 {
				fmt.Println("No GitHub Apps configured. Run 'gh app-auth setup' to add one.")
				return nil
			}

			filtered := appIDSet || clientIDSet

			for idx, app := range apps {
				fmt.Printf("=== %s (App ID: %s) ===\n", appDisplayName(app), app.GetIdentifier())

				if app.InstallationID == 0 {
					err := fmt.Errorf("app %s does not have an installation_id configured", app.GetIdentifier())
					if filtered {
						return err
					}
					fmt.Printf("  Error: %v\n\n", err)
					continue
				}

				if len(app.Patterns) == 0 {
					err := fmt.Errorf("app %s has no patterns configured", app.GetIdentifier())
					if filtered {
						return err
					}
					fmt.Printf("  Error: %v\n\n", err)
					continue
				}

				host := extractHostFromPattern(app.Patterns[0])
				if host == "" {
					err := fmt.Errorf("unable to determine host from pattern %q", app.Patterns[0])
					if filtered {
						return err
					}
					fmt.Printf("  Error: %v\n\n", err)
					continue
				}

				authenticator := auth.NewAuthenticator()
				jwtToken, err := authenticator.GenerateJWTForApp(app)
				if err != nil {
					if filtered {
						return fmt.Errorf("failed to generate JWT for app %s: %w", app.GetIdentifier(), err)
					}
					fmt.Printf("  Error: %v\n\n", err)
					continue
				}

				repoURL := fmt.Sprintf("https://%s", host)
				installationToken, err := authenticator.GetInstallationToken(jwtToken, app.InstallationID, repoURL)
				if err != nil {
					if filtered {
						return fmt.Errorf("failed to obtain installation token for app %s: %w", app.GetIdentifier(), err)
					}
					fmt.Printf("  Error obtaining installation token: %v\n\n", err)
					continue
				}

				repos, err := listInstallationRepositories(installationToken, host)
				if err != nil {
					if filtered {
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

	cmd.Flags().Int64Var(&appID, "app-id", 0, "GitHub App ID (optional, mutually exclusive with --client-id)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "GitHub App Client ID (optional, mutually exclusive with --app-id)")

	return cmd
}

// selectDebugApps returns the apps to inspect based on the --app-id and --client-id flags.
// At most one of the flags may be set.
func selectDebugApps(
	cfg *config.Config,
	appID int64,
	clientID string,
	appIDSet, clientIDSet bool,
) ([]*config.GitHubApp, error) {
	if appIDSet && clientIDSet {
		return nil, fmt.Errorf("use only one of --app-id or --client-id")
	}

	if appIDSet && appID < 0 {
		return nil, fmt.Errorf("app-id must be positive")
	}

	if clientIDSet && clientID == "" {
		return nil, fmt.Errorf("client-id cannot be empty")
	}

	if !appIDSet && !clientIDSet {
		apps := make([]*config.GitHubApp, 0, len(cfg.GitHubApps))
		for i := range cfg.GitHubApps {
			apps = append(apps, &cfg.GitHubApps[i])
		}
		return apps, nil
	}

	for i := range cfg.GitHubApps {
		app := &cfg.GitHubApps[i]
		if clientIDSet && app.ClientID == clientID {
			return []*config.GitHubApp{app}, nil
		}
		if appIDSet && app.AppID == appID {
			return []*config.GitHubApp{app}, nil
		}
	}

	if clientIDSet {
		return nil, fmt.Errorf("app with client ID %s not found in configuration", clientID)
	}
	return nil, fmt.Errorf("app with ID %d not found in configuration", appID)
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
		return nil, auth.FormatAPIStatusError(resp.StatusCode, body)
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
		return nil, auth.FormatAPIStatusError(resp.StatusCode, body)
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
		return fmt.Sprintf("GitHub App %s", app.GetIdentifier())
	}
	return name
}

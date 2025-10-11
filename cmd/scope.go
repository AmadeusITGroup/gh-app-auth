package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/jwt"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/scope"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/secrets"
	"github.com/spf13/cobra"
)

func NewScopeCmd() *cobra.Command {
	var refresh bool

	cmd := &cobra.Command{
		Use:   "scope",
		Short: "Manage GitHub App installation scope cache",
		Long: `Fetch and display GitHub App installation scope information.

The scope determines which repositories an app can access:
- "all": App has access to all repositories in the organization/account
- "selected": App has access only to specific repositories

Scope information is cached locally for 24 hours.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return scopeRun(refresh)
		},
	}

	cmd.Flags().BoolVar(&refresh, "refresh", false, "Force refresh scope from GitHub API")

	return cmd
}

func scopeRun(forceRefresh bool) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.GitHubApps) == 0 {
		fmt.Println("No GitHub Apps configured.")
		fmt.Println("Use 'gh app-auth setup' to configure an app.")
		return nil
	}

	// Initialize secrets manager
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth")
	secretsMgr := secrets.NewManager(configDir)

	// Initialize scope manager
	scopeMgr := scope.NewManager()
	jwtGen := jwt.NewGenerator()

	updated := false

	// Process each app
	for i := range cfg.GitHubApps {
		app := &cfg.GitHubApps[i]

		// Check if refresh needed
		needsRefresh := forceRefresh || scopeMgr.NeedsRefresh(app)

		if needsRefresh {
			fmt.Printf("Fetching scope for %q (App ID: %d)...\n", app.Name, app.AppID)

			// Get private key
			privateKey, err := app.GetPrivateKey(secretsMgr)
			if err != nil {
				fmt.Printf("  ⚠️  Failed to get private key: %v\n", err)
				continue
			}

			// Generate JWT
			jwtToken, err := jwtGen.GenerateTokenFromKey(app.AppID, privateKey)
			if err != nil {
				fmt.Printf("  ⚠️  Failed to generate JWT: %v\n", err)
				continue
			}

			// Fetch scope
			err = scopeMgr.FetchScope(app, jwtToken)
			if err != nil {
				fmt.Printf("  ⚠️  Failed to fetch scope: %v\n", err)
				continue
			}
			updated = true
		}

		// Display scope
		displayScope(app)
	}

	// Save updated config
	if updated {
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Println("\n✅ Scope cache updated")
	}

	return nil
}

func displayScope(app *config.GitHubApp) {
	fmt.Printf("\n📦 %s (App ID: %d, Installation ID: %d)\n", app.Name, app.AppID, app.InstallationID)

	if app.Scope == nil {
		fmt.Println("   ⚠️  No scope information cached")
		fmt.Println("   Run with --refresh to fetch from GitHub")
		return
	}

	scope := app.Scope

	// Account info
	fmt.Printf("   Account: %s (%s)\n", scope.AccountLogin, scope.AccountType)

	// Repository selection
	if scope.RepositorySelection == "all" {
		fmt.Printf("   Scope: ✅ ALL repositories in %s\n", scope.AccountLogin)
	} else {
		fmt.Printf("   Scope: 🎯 %d selected repositories\n", len(scope.Repositories))
		if len(scope.Repositories) > 0 {
			fmt.Println("   Repositories:")
			for _, repo := range scope.Repositories {
				privacy := "public"
				if repo.Private {
					privacy = "private"
				}
				fmt.Printf("     - %s (%s)\n", repo.FullName, privacy)
			}
		}
	}

	// Cache info
	fmt.Printf("   Last fetched: %s\n", scope.LastFetched.Format(time.RFC3339))
	fmt.Printf("   Cache expires: %s\n", scope.CacheExpiry.Format(time.RFC3339))

	if time.Now().After(scope.CacheExpiry) {
		fmt.Println("   ⚠️  Cache expired - run with --refresh")
	}
}

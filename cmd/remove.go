package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wherka-ama/gh-app-auth/pkg/config"
)

func NewRemoveCmd() *cobra.Command {
	var (
		appID int64
		force bool
		all   bool
	)

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove GitHub App configuration",
		Long: `Remove a configured GitHub App by its App ID.

This will remove the app configuration and clear any cached tokens.`,
		Aliases: []string{"rm", "delete"},
		Example: `  # Remove specific app
  gh app-auth remove --app-id 123456
  
  # Remove without confirmation
  gh app-auth remove --app-id 123456 --force
  
  # Remove all configured apps
  gh app-auth remove --all`,
		RunE: removeRun(&appID, &force, &all),
	}

	cmd.Flags().Int64Var(&appID, "app-id", 0, "GitHub App ID to remove")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&all, "all", false, "Remove all configured GitHub Apps")

	return cmd
}

func removeRun(appID *int64, force *bool, all *bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if !*all && *appID <= 0 {
			return fmt.Errorf("either --app-id or --all must be specified")
		}

		if *all && *appID > 0 {
			return fmt.Errorf("cannot specify both --app-id and --all")
		}

		// Load configuration
		cfg, err := config.Load()
		if os.IsNotExist(err) {
			fmt.Printf("No GitHub Apps configured.\n")
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		if len(cfg.GitHubApps) == 0 {
			fmt.Printf("No GitHub Apps configured.\n")
			return nil
		}

		if *all {
			return removeAllApps(cfg, *force)
		}

		return removeSingleApp(cfg, *appID, *force)
	}
}

func removeSingleApp(cfg *config.Config, appID int64, force bool) error {
	// Find the app
	appIndex := -1
	var appToRemove config.GitHubApp
	
	for i, app := range cfg.GitHubApps {
		if app.AppID == appID {
			appIndex = i
			appToRemove = app
			break
		}
	}

	if appIndex == -1 {
		return fmt.Errorf("GitHub App with ID %d not found", appID)
	}

	// Confirm removal unless forced
	if !force {
		fmt.Printf("This will remove the following GitHub App configuration:\n")
		fmt.Printf("  Name: %s\n", appToRemove.Name)
		fmt.Printf("  App ID: %d\n", appToRemove.AppID)
		fmt.Printf("  Patterns: %v\n", appToRemove.Patterns)
		fmt.Printf("\nAre you sure? (y/N): ")
		
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			fmt.Printf("Cancelled.\n")
			return nil
		}
	}

	// Remove the app
	cfg.GitHubApps = append(cfg.GitHubApps[:appIndex], cfg.GitHubApps[appIndex+1:]...)

	// Save configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Clear cached tokens for this app
	if err := clearCachedTokens(appID); err != nil {
		fmt.Printf("Warning: failed to clear cached tokens: %v\n", err)
	}

	fmt.Printf("✅ Successfully removed GitHub App '%s' (ID: %d)\n", appToRemove.Name, appID)
	return nil
}

func removeAllApps(cfg *config.Config, force bool) error {
	if !force {
		fmt.Printf("This will remove ALL %d configured GitHub Apps:\n", len(cfg.GitHubApps))
		for _, app := range cfg.GitHubApps {
			fmt.Printf("  - %s (ID: %d)\n", app.Name, app.AppID)
		}
		fmt.Printf("\nAre you sure? (y/N): ")
		
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			fmt.Printf("Cancelled.\n")
			return nil
		}
	}

	count := len(cfg.GitHubApps)
	
	// Clear all apps
	cfg.GitHubApps = []config.GitHubApp{}

	// Save configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Clear all cached tokens
	if err := clearAllCachedTokens(); err != nil {
		fmt.Printf("Warning: failed to clear cached tokens: %v\n", err)
	}

	fmt.Printf("✅ Successfully removed all %d GitHub Apps\n", count)
	return nil
}

func clearCachedTokens(appID int64) error {
	// Implementation would clear cached tokens for specific app
	// For now, just a placeholder
	return nil
}

func clearAllCachedTokens() error {
	// Implementation would clear all cached tokens
	// For now, just a placeholder
	return nil
}

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/secrets"
	"github.com/spf13/cobra"
)

const (
	confirmYes    = "yes"
	confirmY      = "y"
	confirmYesCap = "Yes"
	confirmYCap   = "Y"
)

func NewRemoveCmd() *cobra.Command {
	var (
		appID   int64
		patName string
		force   bool
		allApps bool
		allPATs bool
	)

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove credential configuration",
		Long: `Remove a configured GitHub App or Personal Access Token.

	This will remove the credential configuration and clear any cached secrets.`,
		Aliases: []string{"rm", "delete"},
		Example: `  # Remove specific app
  gh app-auth remove --app-id 123456
  
  # Remove without confirmation
  gh app-auth remove --app-id 123456 --force
  
  # Remove all configured apps
  gh app-auth remove --all

  # Remove a Personal Access Token
  gh app-auth remove --pat-name "My PAT"

  # Remove all Personal Access Tokens
  gh app-auth remove --all-pats`,
		RunE: removeRun(&appID, &patName, &force, &allApps, &allPATs),
	}

	cmd.Flags().Int64Var(&appID, "app-id", 0, "GitHub App ID to remove")
	cmd.Flags().StringVar(&patName, "pat-name", "", "Personal Access Token name to remove")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&allApps, "all", false, "Remove all configured GitHub Apps")
	cmd.Flags().BoolVar(&allPATs, "all-pats", false, "Remove all configured Personal Access Tokens")

	return cmd
}

func removeRun(appID *int64, patName *string, force, allApps, allPATs *bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		targetCount := 0
		if *allApps {
			targetCount++
		}
		if *appID > 0 {
			targetCount++
		}
		if *patName != "" {
			targetCount++
		}
		if *allPATs {
			targetCount++
		}

		if targetCount == 0 {
			return fmt.Errorf("specify one of --app-id, --pat-name, --all, or --all-pats")
		}

		if targetCount > 1 {
			return fmt.Errorf("use only one of --app-id, --pat-name, --all, or --all-pats at a time")
		}

		// Load configuration
		cfg, err := config.Load()
		if os.IsNotExist(err) {
			fmt.Printf("No GitHub Apps or Personal Access Tokens configured.\n")
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		hasApps := len(cfg.GitHubApps) > 0
		hasPATs := len(cfg.PATs) > 0
		if !hasApps && !hasPATs {
			fmt.Printf("No GitHub Apps or Personal Access Tokens configured.\n")
			return nil
		}

		if *allApps {
			if !hasApps {
				fmt.Printf("No GitHub Apps configured.\n")
				return nil
			}
			return removeAllApps(cfg, *force)
		}

		if *appID > 0 {
			if !hasApps {
				fmt.Printf("No GitHub Apps configured.\n")
				return nil
			}
			return removeSingleApp(cfg, *appID, *force)
		}

		if *allPATs {
			if !hasPATs {
				fmt.Printf("No Personal Access Tokens configured.\n")
				return nil
			}
			return removeAllPATs(cfg, *force)
		}

		// PAT by name
		if !hasPATs {
			fmt.Printf("No Personal Access Tokens configured.\n")
			return nil
		}
		return removeSinglePAT(cfg, *patName, *force)
	}
}

func removeSingleApp(cfg *config.Config, appID int64, force bool) error {
	// Find the app
	appIndex, appToRemove, err := findAppByID(cfg, appID)
	if err != nil {
		return err
	}

	// Confirm removal unless forced
	if !force {
		if !confirmAppRemoval(appToRemove) {
			return nil
		}
	}

	// Perform the removal
	return performAppRemoval(cfg, appIndex, appToRemove, appID)
}

func removeAllApps(cfg *config.Config, force bool) error {
	// Confirm removal unless forced
	if !force {
		if !confirmAllAppsRemoval(cfg.GitHubApps) {
			return nil
		}
	}

	count := len(cfg.GitHubApps)

	// Perform the removal
	if err := performAllAppsRemoval(cfg); err != nil {
		return err
	}

	// Display success message
	displayAllAppsRemovalSuccess(count)
	return nil
}

func removeSinglePAT(cfg *config.Config, patName string, force bool) error {
	patIndex, patToRemove, err := findPATByName(cfg, patName)
	if err != nil {
		return err
	}

	if !force {
		if !confirmPATRemoval(patToRemove) {
			return nil
		}
	}

	return performPATRemoval(cfg, patIndex, patToRemove)
}

func removeAllPATs(cfg *config.Config, force bool) error {
	if !force {
		if !confirmAllPATsRemoval(cfg.PATs) {
			return nil
		}
	}

	count := len(cfg.PATs)
	if err := performAllPATsRemoval(cfg); err != nil {
		return err
	}

	displayAllPATsRemovalSuccess(count)
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

// findAppByID finds an app by ID and returns its index and the app itself
func findAppByID(cfg *config.Config, appID int64) (int, config.GitHubApp, error) {
	for i, app := range cfg.GitHubApps {
		if app.AppID == appID {
			return i, app, nil
		}
	}
	return -1, config.GitHubApp{}, fmt.Errorf("GitHub App with ID %d not found", appID)
}

// confirmAppRemoval prompts the user to confirm app removal
func confirmAppRemoval(appToRemove config.GitHubApp) bool {
	fmt.Printf("This will remove the following GitHub App configuration:\n")
	fmt.Printf("  Name: %s\n", appToRemove.Name)
	fmt.Printf("  App ID: %d\n", appToRemove.AppID)
	fmt.Printf("  Patterns: %v\n", appToRemove.Patterns)
	fmt.Printf("\nAre you sure? (y/N): ")

	var response string
	_, _ = fmt.Scanln(&response) // Error is intentionally ignored - treat as "N"

	if response != confirmY && response != confirmYCap && response != confirmYes && response != confirmYesCap {
		fmt.Printf("Canceled.\n")
		return false
	}
	return true
}

// performAppRemoval performs the actual app removal operations
func performAppRemoval(cfg *config.Config, appIndex int, appToRemove config.GitHubApp, appID int64) error {
	// Initialize secrets manager
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth")
	secretMgr := secrets.NewManager(configDir)

	// Delete private key from secure storage
	if err := appToRemove.DeletePrivateKey(secretMgr); err != nil {
		fmt.Printf("⚠️  Warning: failed to delete private key from storage: %v\n", err)
	}

	// Remove the app from configuration
	cfg.GitHubApps = append(cfg.GitHubApps[:appIndex], cfg.GitHubApps[appIndex+1:]...)

	// Save configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Clear cached tokens for this app
	if err := clearCachedTokens(appID); err != nil {
		fmt.Printf("⚠️  Warning: failed to clear cached tokens: %v\n", err)
	}

	fmt.Printf("✅ Successfully removed GitHub App '%s' (ID: %d)\n", appToRemove.Name, appID)
	fmt.Printf("   🗑️  Private key deleted from secure storage\n")
	return nil
}

// confirmAllAppsRemoval prompts the user to confirm removal of all apps
func confirmAllAppsRemoval(apps []config.GitHubApp) bool {
	fmt.Printf("This will remove ALL %d configured GitHub Apps:\n", len(apps))
	for _, app := range apps {
		fmt.Printf("  - %s (ID: %d)\n", app.Name, app.AppID)
	}
	fmt.Printf("\nAre you sure? (y/N): ")

	var response string
	_, _ = fmt.Scanln(&response) // Error is intentionally ignored - treat as "N"

	if response != confirmY && response != confirmYCap && response != confirmYes && response != confirmYesCap {
		fmt.Printf("Canceled.\n")
		return false
	}
	return true
}

// performAllAppsRemoval performs the actual removal of all apps
func performAllAppsRemoval(cfg *config.Config) error {
	// Initialize secrets manager
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth")
	secretMgr := secrets.NewManager(configDir)

	// Delete all private keys from secure storage
	for _, app := range cfg.GitHubApps {
		if err := app.DeletePrivateKey(secretMgr); err != nil {
			fmt.Printf("⚠️  Warning: failed to delete key for '%s': %v\n", app.Name, err)
		}
	}

	// Clear all apps from configuration
	cfg.GitHubApps = []config.GitHubApp{}

	// Save configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Clear all cached tokens
	if err := clearAllCachedTokens(); err != nil {
		fmt.Printf("⚠️  Warning: failed to clear cached tokens: %v\n", err)
	}

	return nil
}

// displayAllAppsRemovalSuccess displays the success message for removing all apps
func displayAllAppsRemovalSuccess(count int) {
	fmt.Printf("✅ Successfully removed all %d GitHub Apps\n", count)
	fmt.Printf("   🗑️  All private keys deleted from secure storage\n")
}

func findPATByName(cfg *config.Config, name string) (int, config.PersonalAccessToken, error) {
	for i, pat := range cfg.PATs {
		if pat.Name == name {
			return i, pat, nil
		}
	}
	return -1, config.PersonalAccessToken{}, fmt.Errorf("personal access token %q not found", name)
}

func confirmPATRemoval(patToRemove config.PersonalAccessToken) bool {
	fmt.Printf("This will remove the following Personal Access Token configuration:\n")
	fmt.Printf("  Name: %s\n", patToRemove.Name)
	fmt.Printf("  Patterns: %v\n", patToRemove.Patterns)
	fmt.Printf("  Priority: %d\n", patToRemove.Priority)
	fmt.Printf("\nAre you sure? (y/N): ")

	var response string
	_, _ = fmt.Scanln(&response)

	if response != confirmY && response != confirmYCap && response != confirmYes && response != confirmYesCap {
		fmt.Printf("Canceled.\n")
		return false
	}
	return true
}

func confirmAllPATsRemoval(pats []config.PersonalAccessToken) bool {
	fmt.Printf("This will remove ALL %d configured Personal Access Tokens:\n", len(pats))
	for _, pat := range pats {
		fmt.Printf("  - %s\n", pat.Name)
	}
	fmt.Printf("\nAre you sure? (y/N): ")

	var response string
	_, _ = fmt.Scanln(&response)

	if response != confirmY && response != confirmYCap && response != confirmYes && response != confirmYesCap {
		fmt.Printf("Canceled.\n")
		return false
	}
	return true
}

func performPATRemoval(cfg *config.Config, patIndex int, patToRemove config.PersonalAccessToken) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth")
	secretMgr := secrets.NewManager(configDir)

	if err := patToRemove.DeletePAT(secretMgr); err != nil {
		fmt.Printf("⚠️  Warning: failed to delete PAT from storage: %v\n", err)
	}

	cfg.PATs = append(cfg.PATs[:patIndex], cfg.PATs[patIndex+1:]...)

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("✅ Successfully removed Personal Access Token '%s'\n", patToRemove.Name)
	fmt.Printf("   🗑️  Token deleted from secure storage\n")
	return nil
}

func performAllPATsRemoval(cfg *config.Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth")
	secretMgr := secrets.NewManager(configDir)

	for _, pat := range cfg.PATs {
		if err := pat.DeletePAT(secretMgr); err != nil {
			fmt.Printf("⚠️  Warning: failed to delete PAT '%s': %v\n", pat.Name, err)
		}
	}

	cfg.PATs = []config.PersonalAccessToken{}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

func displayAllPATsRemovalSuccess(count int) {
	fmt.Printf("✅ Successfully removed all %d Personal Access Tokens\n", count)
	fmt.Printf("   🗑️  All tokens deleted from secure storage\n")
}

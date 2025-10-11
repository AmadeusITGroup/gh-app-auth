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
	storageKeyring    = "keyring"
	storageFilesystem = "filesystem"
)

func NewMigrateCmd() *cobra.Command {
	var (
		dryRun  bool
		storage string
		force   bool
	)

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate private keys to encrypted storage",
		Long: `Migrate private keys from filesystem to encrypted keyring storage.

This command helps you move your private keys from plain text files to 
the OS-native encrypted keyring (Keychain on macOS, Credential Manager 
on Windows, Secret Service on Linux).

The migration is safe and non-destructive - original key files are kept
as a fallback unless you specify --force.`,
		Example: `  # Preview migration (dry-run)
  gh app-auth migrate --dry-run
  
  # Migrate keys to encrypted keyring
  gh app-auth migrate
  
  # Migrate to filesystem storage
  gh app-auth migrate --storage filesystem
  
  # Migrate and remove original key files
  gh app-auth migrate --force`,
		RunE: migrateRun(&dryRun, &storage, &force),
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview migration without making changes")
	cmd.Flags().StringVar(&storage, "storage", storageKeyring, "Target storage: keyring or filesystem")
	cmd.Flags().BoolVar(&force, "force", false, "Remove original key files after successful migration")

	return cmd
}

func migrateRun(dryRun *bool, storage *string, force *bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Validate storage option
		if err := validateStorageOption(*storage); err != nil {
			return err
		}

		// Load configuration and initialize
		cfg, secretMgr, err := initializeMigration(storage)
		if err != nil {
			return err
		}
		if cfg == nil {
			return nil // Nothing to migrate
		}

		// Analyze apps to migrate
		appsToMigrate, appsUpToDate, appsNeedAttention := analyzeAppsForMigration(cfg.GitHubApps, *storage)

		// Display migration summary and handle dry-run
		if shouldExit := displayMigrationSummary(
			cfg.GitHubApps, appsToMigrate, appsUpToDate, appsNeedAttention, *storage, *dryRun,
		); shouldExit {
			return nil
		}

		// Perform migration
		migrated, failed := performMigration(cfg, appsToMigrate, secretMgr, *storage, *force)

		// Save configuration if needed
		if err := saveConfigurationIfNeeded(cfg, migrated); err != nil {
			return err
		}

		// Display results and next steps
		displayMigrationResults(migrated, failed, *force, *storage)

		if failed > 0 {
			return fmt.Errorf("migration completed with %d errors", failed)
		}

		return nil
	}
}

// initializeMigration loads configuration and sets up secrets manager
func initializeMigration(storage *string) (*config.Config, *secrets.Manager, error) {
	// Load configuration
	cfg, err := config.Load()
	if os.IsNotExist(err) {
		fmt.Printf("No configuration found. Nothing to migrate.\n")
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if len(cfg.GitHubApps) == 0 {
		fmt.Printf("No GitHub Apps configured. Nothing to migrate.\n")
		return nil, nil, nil
	}

	// Initialize secrets manager
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth")
	secretMgr := secrets.NewManager(configDir)

	// Check keyring availability
	keyringAvailable := secretMgr.IsAvailable()
	if *storage == storageKeyring && !keyringAvailable {
		fmt.Printf("⚠️  Warning: Keyring not available on this system.\n")
		fmt.Printf("   Migration will use filesystem storage instead.\n")
		*storage = storageFilesystem
	}

	return cfg, secretMgr, nil
}

// analyzeAppsForMigration categorizes apps based on their migration needs
func analyzeAppsForMigration(apps []config.GitHubApp, targetStorage string) (
	toMigrate, upToDate, needAttention []config.GitHubApp,
) {
	for _, app := range apps {
		// Check current state
		if app.PrivateKeySource == "" {
			// Legacy config
			toMigrate = append(toMigrate, app)
		} else if targetStorage == storageKeyring && app.PrivateKeySource != config.PrivateKeySourceKeyring {
			// Needs migration to keyring
			toMigrate = append(toMigrate, app)
		} else if targetStorage == storageFilesystem && app.PrivateKeySource != config.PrivateKeySourceFilesystem {
			// Needs migration to filesystem
			toMigrate = append(toMigrate, app)
		} else if app.PrivateKeySource == config.PrivateKeySourceInline {
			// Inline keys always need migration
			needAttention = append(needAttention, app)
		} else {
			// Already using target storage
			upToDate = append(upToDate, app)
		}
	}
	return toMigrate, upToDate, needAttention
}

// displayMigrationSummary shows migration summary and handles dry-run mode
// Returns true if the function should exit early
func displayMigrationSummary(
	allApps, appsToMigrate, appsUpToDate, appsNeedAttention []config.GitHubApp,
	storage string,
	dryRun bool,
) bool {
	// Display migration summary
	fmt.Printf("📊 Migration Summary\n")
	fmt.Printf("─────────────────────────────────────\n")
	fmt.Printf("Target Storage: %s\n", storage)
	fmt.Printf("Total Apps: %d\n", len(allApps))
	fmt.Printf("  • Need migration: %d\n", len(appsToMigrate))
	fmt.Printf("  • Already up-to-date: %d\n", len(appsUpToDate))
	if len(appsNeedAttention) > 0 {
		fmt.Printf("  • Need attention: %d\n", len(appsNeedAttention))
	}
	fmt.Printf("\n")

	if len(appsToMigrate) == 0 && len(appsNeedAttention) == 0 {
		fmt.Printf("✅ All apps are already using %s storage. No migration needed.\n", storage)
		return true
	}

	// Show apps needing attention
	if len(appsNeedAttention) > 0 {
		fmt.Printf("⚠️  Apps needing attention (inline keys):\n")
		for _, app := range appsNeedAttention {
			fmt.Printf("  • %s (ID: %d) - inline key detected\n", app.Name, app.AppID)
		}
		fmt.Printf("\n")
	}

	// Show apps to migrate
	if len(appsToMigrate) > 0 {
		fmt.Printf("🔄 Apps to migrate:\n")
		for _, app := range appsToMigrate {
			currentSource := string(app.PrivateKeySource)
			if currentSource == "" {
				currentSource = "legacy"
			}
			fmt.Printf("  • %s (ID: %d)\n", app.Name, app.AppID)
			fmt.Printf("    From: %s → To: %s\n", currentSource, storage)
		}
		fmt.Printf("\n")
	}

	if dryRun {
		fmt.Printf("🔍 Dry-run mode: No changes will be made.\n")
		fmt.Printf("\nTo perform the migration, run without --dry-run:\n")
		fmt.Printf("  gh app-auth migrate\n")
		return true
	}

	return false
}

// validateStorageOption validates the storage option parameter
func validateStorageOption(storage string) error {
	if storage != storageKeyring && storage != storageFilesystem {
		return fmt.Errorf("invalid storage option: %s (must be '%s' or '%s')", storage, storageKeyring, storageFilesystem)
	}
	return nil
}

// performMigration executes the migration process for all apps
func performMigration(
	cfg *config.Config, appsToMigrate []config.GitHubApp, secretMgr *secrets.Manager,
	storage string, force bool,
) (int, int) {
	fmt.Printf("🚀 Starting migration...\n\n")

	migrated := 0
	failed := 0

	for i := range cfg.GitHubApps {
		app := &cfg.GitHubApps[i]

		// Skip apps that don't need migration
		if !needsMigration(app, appsToMigrate) {
			continue
		}

		fmt.Printf("  Migrating '%s' (ID: %d)...\n", app.Name, app.AppID)

		// Get current private key
		privateKey, err := app.GetPrivateKey(secretMgr)
		if err != nil {
			fmt.Printf("    ❌ Failed to read current key: %v\n", err)
			failed++
			continue
		}

		// Migrate based on target storage
		if err := migrateAppToStorage(app, secretMgr, privateKey, storage, force); err != nil {
			fmt.Printf("    ❌ %v\n", err)
			failed++
			continue
		}

		migrated++
	}

	return migrated, failed
}

// needsMigration checks if an app needs migration
func needsMigration(app *config.GitHubApp, appsToMigrate []config.GitHubApp) bool {
	for _, migApp := range appsToMigrate {
		if migApp.Name == app.Name {
			return true
		}
	}
	return false
}

// migrateAppToStorage migrates a single app to the target storage
func migrateAppToStorage(
	app *config.GitHubApp, secretMgr *secrets.Manager, privateKey string,
	storage string, force bool,
) error {
	if storage == storageKeyring {
		return migrateToKeyring(app, secretMgr, privateKey, force)
	}
	return migrateToFilesystem(app)
}

// migrateToKeyring migrates an app to keyring storage
func migrateToKeyring(app *config.GitHubApp, secretMgr *secrets.Manager, privateKey string, force bool) error {
	backend, err := app.SetPrivateKey(secretMgr, privateKey)
	if err != nil {
		return fmt.Errorf("failed to store in keyring: %w", err)
	}

	if backend == secrets.StorageBackendKeyring {
		fmt.Printf("    ✅ Migrated to encrypted keyring\n")
		handleOriginalKeyFile(app, force)
	} else {
		fmt.Printf("    ⚠️  Fell back to filesystem storage\n")
	}

	return nil
}

// migrateToFilesystem migrates an app to filesystem storage
func migrateToFilesystem(app *config.GitHubApp) error {
	app.PrivateKeySource = config.PrivateKeySourceFilesystem
	if app.PrivateKeyPath == "" {
		return fmt.Errorf("no filesystem path available, skipping")
	}
	fmt.Printf("    ✅ Migrated to filesystem storage\n")
	return nil
}

// handleOriginalKeyFile handles the original key file based on force flag
func handleOriginalKeyFile(app *config.GitHubApp, force bool) {
	if force && app.PrivateKeyPath != "" {
		if err := os.Remove(app.PrivateKeyPath); err != nil {
			fmt.Printf("    ⚠️  Warning: Failed to remove original key file: %v\n", err)
		} else {
			fmt.Printf("    🗑️  Removed original key file\n")
		}
	} else if app.PrivateKeyPath != "" {
		fmt.Printf("    📝 Original key file kept as fallback: %s\n", app.PrivateKeyPath)
	}
}

// saveConfigurationIfNeeded saves the configuration if any apps were migrated
func saveConfigurationIfNeeded(cfg *config.Config, migrated int) error {
	if migrated > 0 {
		// Update config version
		if cfg.Version == "" {
			cfg.Version = config.CurrentConfigVersion
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}
	}
	return nil
}

// displayMigrationResults shows the migration results and next steps
func displayMigrationResults(migrated, failed int, force bool, storage string) {
	fmt.Printf("\n📊 Migration Results\n")
	fmt.Printf("─────────────────────────────────────\n")
	fmt.Printf("✅ Successfully migrated: %d\n", migrated)
	if failed > 0 {
		fmt.Printf("❌ Failed: %d\n", failed)
	}

	if migrated > 0 {
		fmt.Printf("\n💡 Next steps:\n")
		fmt.Printf("  1. Verify authentication: gh app-auth test --repo <repository-url>\n")
		fmt.Printf("  2. Check key status: gh app-auth list --verify-keys\n")
		if !force && storage == storageKeyring {
			fmt.Printf("  3. If everything works, re-run with --force to remove original key files\n")
		}
	}
}

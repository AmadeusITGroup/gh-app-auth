package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wherka-ama/gh-app-auth/pkg/config"
	"github.com/wherka-ama/gh-app-auth/pkg/jwt"
)

func NewSetupCmd() *cobra.Command {
	var (
		appID           int64
		keyFile         string
		patterns        []string
		name            string
		installationID  int64
		priority        int
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure GitHub App authentication",
		Long: `Configure GitHub App authentication for specific repository patterns.

This command sets up a GitHub App for authentication with git operations.
You'll need the App ID and private key file from your GitHub App settings.`,
		Example: `  # Basic setup
  gh app-auth setup --app-id 123456 --key-file ~/.ssh/my-app.pem --patterns "github.com/myorg/*"
  
  # Setup with custom name and priority
  gh app-auth setup \
    --app-id 123456 \
    --key-file ~/.ssh/my-app.pem \
    --patterns "github.com/myorg/*" \
    --name "Corporate App" \
    --priority 10
    
  # Multiple patterns
  gh app-auth setup \
    --app-id 123456 \
    --key-file ~/.ssh/my-app.pem \
    --patterns "github.com/myorg/*,github.example.com/corp/*"`,
		RunE: setupRun(&appID, &keyFile, &patterns, &name, &installationID, &priority),
	}

	// Required flags
	cmd.Flags().Int64Var(&appID, "app-id", 0, "GitHub App ID (required)")
	cmd.Flags().StringVar(&keyFile, "key-file", "", "Path to private key file (required)")
	cmd.Flags().StringSliceVar(&patterns, "patterns", nil, "Repository patterns to match (required)")
	
	// Optional flags
	cmd.Flags().StringVar(&name, "name", "", "Friendly name for the GitHub App")
	cmd.Flags().Int64Var(&installationID, "installation-id", 0, "Installation ID (auto-detected if not provided)")
	cmd.Flags().IntVar(&priority, "priority", 5, "Priority for pattern matching (higher = more priority)")

	// Mark required flags
	cmd.MarkFlagRequired("app-id")
	cmd.MarkFlagRequired("key-file") 
	cmd.MarkFlagRequired("patterns")

	return cmd
}

func setupRun(appID *int64, keyFile *string, patterns *[]string, name *string, installationID *int64, priority *int) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Validate inputs
		if *appID <= 0 {
			return fmt.Errorf("app-id must be a positive integer")
		}

		if *keyFile == "" {
			return fmt.Errorf("key-file is required")
		}

		if len(*patterns) == 0 {
			return fmt.Errorf("at least one pattern is required")
		}

		// Expand key file path
		expandedKeyFile, err := expandPath(*keyFile)
		if err != nil {
			return fmt.Errorf("invalid key file path: %w", err)
		}

		// Verify key file exists and has correct permissions
		if err := validateKeyFile(expandedKeyFile); err != nil {
			return fmt.Errorf("key file validation failed: %w", err)
		}

		// Test JWT generation to ensure key is valid
		if err := testJWTGeneration(*appID, expandedKeyFile); err != nil {
			return fmt.Errorf("JWT generation test failed: %w", err)
		}

		// Load or create configuration
		cfg, err := config.LoadOrCreate()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Set default name if not provided
		if *name == "" {
			*name = fmt.Sprintf("GitHub App %d", *appID)
		}

		// Create GitHub App configuration
		app := config.GitHubApp{
			Name:            *name,
			AppID:           *appID,
			InstallationID:  *installationID,
			PrivateKeyPath:  expandedKeyFile,
			Patterns:        *patterns,
			Priority:        *priority,
		}

		// Validate the app configuration
		if err := app.Validate(); err != nil {
			return fmt.Errorf("invalid app configuration: %w", err)
		}

		// Add or update the app in configuration
		cfg.AddOrUpdateApp(app)

		// Save configuration
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("✅ Successfully configured GitHub App '%s'\n", *name)
		fmt.Printf("   App ID: %d\n", *appID)
		fmt.Printf("   Patterns: %s\n", strings.Join(*patterns, ", "))
		fmt.Printf("   Priority: %d\n", *priority)
		fmt.Printf("   Key file: %s\n", expandedKeyFile)
		
		fmt.Printf("\n💡 Next steps:\n")
		fmt.Printf("   1. Test authentication: gh app-auth test --repo <repository-url>\n")
		fmt.Printf("   2. Configure git: git config credential.\"https://github.com/yourorg\".helper \"app-auth git-credential\"\n")

		return nil
	}
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to get home directory: %w", err)
		}
		return filepath.Join(homeDir, path[2:]), nil
	}
	return filepath.Abs(path)
}

func validateKeyFile(keyPath string) error {
	fileInfo, err := os.Stat(keyPath)
	if err != nil {
		return fmt.Errorf("failed to access key file: %w", err)
	}

	// Check permissions (should be 600 or 400)
	if fileInfo.Mode().Perm() & 0044 != 0 {
		return fmt.Errorf("private key file has overly permissive permissions %o (should be 600 or 400)", 
			fileInfo.Mode().Perm())
	}

	return nil
}

func testJWTGeneration(appID int64, keyPath string) error {
	generator := jwt.NewGenerator()
	_, err := generator.GenerateToken(appID, keyPath)
	if err != nil {
		return fmt.Errorf("failed to generate JWT token: %w", err)
	}
	return nil
}

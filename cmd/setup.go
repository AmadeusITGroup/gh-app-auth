package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/jwt"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/secrets"
	"github.com/spf13/cobra"
)

// Common errors returned by setup
var (
	ErrConflictingKeyOptions = errors.New("⚠️ Both --key-file and GH_APP_PRIVATE_KEY are provided, " +
		"choose one or the other, not both.\n")
)

func NewSetupCmd() *cobra.Command {
	var (
		appID          int64
		keyFile        string
		patterns       []string
		name           string
		installationID int64
		priority       int
		useKeyring     bool
		useFilesystem  bool
		pat            string
		username       string
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
    --patterns "github.com/myorg/*,github.example.com/corp/*"

  # Setup a Personal Access Token for GitHub
  gh app-auth setup \
    --pat gh_your_token_here \
    --patterns "github.com/myorg/" \
    --name "My PAT" \
    --priority 10

  # Setup a Personal Access Token for Bitbucket (with username)
  gh app-auth setup \
    --pat your_bitbucket_token \
    --patterns "bitbucket.example.com/" \
    --username your_username \
    --name "Bitbucket PAT" \
    --priority 10`,
		RunE: setupRun(
			&appID, &keyFile, &patterns, &name, &installationID,
			&priority, &useKeyring, &useFilesystem, &pat, &username,
		),
	}

	// GitHub App flags
	cmd.Flags().Int64Var(&appID, "app-id", 0, "GitHub App ID (required for app setup)")
	cmd.Flags().StringVar(&keyFile, "key-file", "", "Path to private key file (or use GH_APP_PRIVATE_KEY env var)")
	cmd.Flags().Int64Var(&installationID, "installation-id", 0, "Installation ID (auto-detected if not provided)")

	// PAT flags
	cmd.Flags().StringVar(&pat, "pat", "", "Personal Access Token (for PAT setup)")
	cmd.Flags().StringVar(
		&username, "username", "",
		"Username for HTTP basic auth (optional, defaults to 'x-access-token' for GitHub)",
	)

	// Common flags
	cmd.Flags().StringSliceVar(&patterns, "patterns", nil, "Repository patterns to match (required)")

	cmd.Flags().StringVar(&name, "name", "", "Friendly name for the GitHub App or PAT")
	cmd.Flags().IntVar(&priority, "priority", 5, "Priority for pattern matching (higher = more priority)")

	// Storage method flags
	cmd.Flags().BoolVar(&useKeyring, "use-keyring", true, "Store private key in OS keyring (default)")
	cmd.Flags().BoolVar(&useFilesystem, "use-filesystem", false, "Force filesystem storage instead of keyring")

	// Mark required flags
	_ = cmd.MarkFlagRequired("patterns")

	return cmd
}

func setupRun(
	appID *int64, keyFile *string, patterns *[]string,
	name *string, installationID *int64, priority *int,
	useKeyring *bool, useFilesystem *bool, pat *string, username *string,
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Determine if this is PAT or App setup
		isPATSetup := *pat != ""
		isAppSetup := *appID > 0

		// Validate mutually exclusive options
		if isPATSetup && isAppSetup {
			return fmt.Errorf("cannot use both --pat and --app-id; choose one authentication method")
		}
		if !isPATSetup && !isAppSetup {
			return fmt.Errorf("must specify either --pat for PAT setup or --app-id for GitHub App setup")
		}

		// Load or create configuration
		cfg, err := config.LoadOrCreate()

		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		if isPATSetup {
			return setupPAT(cfg, *pat, *name, *patterns, *priority, *username)
		}

		return setupGitHubApp(
			cfg, *appID, *keyFile, *name, *installationID,
			*patterns, *priority, *useKeyring, *useFilesystem,
		)
	}
}

func setupGitHubApp(
	cfg *config.Config, appID int64, keyFile, name string, installationID int64,
	patterns []string, priority int, useKeyring, useFilesystem bool,
) error {
	// Validate inputs
	if err := validateSetupInputs(appID, patterns, &useKeyring, &useFilesystem, keyFile); err != nil {
		return err
	}

	// Get and validate private key
	privateKeyContent, expandedKeyFile, err := getPrivateKey(keyFile)
	if err != nil {
		return err
	}

	// Test JWT generation to ensure key is valid
	jwtToken, err := generateJWTForSetup(appID, privateKeyContent)
	if err != nil {
		return err
	}

	// Auto-detect installation ID if not provided
	if installationID == 0 {
		detectedID, err := autoDetectInstallationID(jwtToken, patterns)
		if err != nil {
			return fmt.Errorf("failed to auto-detect installation ID: %w", err)
		}
		installationID = detectedID
		fmt.Printf("🔍 Auto-detected installation ID: %d\n", installationID)
	}

	// Create the GitHub App (validation happens after storage configuration)
	app := createGitHubApp(appID, name, installationID, patterns, priority)

	// Store private key and configure storage
	backend, err := configureAppStorage(&app, privateKeyContent, expandedKeyFile, useKeyring)
	if err != nil {
		return err
	}

	// Validate the complete app configuration after storage is configured
	if err := app.Validate(); err != nil {
		return fmt.Errorf("invalid app configuration: %w", err)
	}

	// Save configuration
	if err := saveAppConfiguration(cfg, &app); err != nil {
		return err
	}

	// Display success message and next steps
	displaySetupSuccess(name, appID, patterns, priority, backend, expandedKeyFile)

	return nil
}

func setupPAT(
	cfg *config.Config, token, name string, patterns []string,
	priority int, username string,
) error {
	// Set default name if not provided
	if name == "" {
		name = fmt.Sprintf("PAT for %s", patterns[0])
	}

	// Create PAT configuration
	pat := config.PersonalAccessToken{
		Name:     name,
		Patterns: patterns,
		Priority: priority,
		Username: username,
	}

	// Use XDG config directory for secrets
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth")
	secretMgr := secrets.NewManager(configDir)

	// Store PAT
	backend, err := pat.SetPAT(secretMgr, token)
	if err != nil {
		return fmt.Errorf("failed to store PAT: %w", err)
	}

	// Validate the PAT configuration
	if err := pat.Validate(); err != nil {
		return fmt.Errorf("invalid PAT configuration: %w", err)
	}

	// Save configuration
	cfg.AddOrUpdatePAT(&pat)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Display success message
	fmt.Printf("✅ Successfully configured PAT '%s'\n", name)
	fmt.Printf("   Patterns: %s\n", strings.Join(patterns, ", "))
	fmt.Printf("   Priority: %d\n", priority)
	if username != "" {
		fmt.Printf("   Username: %s\n", username)
	}
	if backend == secrets.StorageBackendKeyring {
		fmt.Printf("   🔐 Storage: OS Keyring (encrypted)\n")
	} else {
		fmt.Printf("   📁 Storage: Filesystem\n")
	}

	fmt.Printf("\n💡 Next steps:\n")
	fmt.Printf("   1. Sync git credential helper: gh app-auth gitconfig --sync --global\n")
	fmt.Printf("      # or run 'gh app-auth gitconfig --sync --local' inside a repository\n")

	return nil
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
	if fileInfo.Mode().Perm()&0044 != 0 {
		return fmt.Errorf("private key file has overly permissive permissions %o (should be 600 or 400)",
			fileInfo.Mode().Perm())
	}

	return nil
}

// validateSetupInputs validates the setup command inputs
func validateSetupInputs(appID int64, patterns []string, useKeyring *bool, useFilesystem *bool, keyFile string) error {
	if appID <= 0 {
		return fmt.Errorf("app-id must be a positive integer")
	}

	if len(patterns) == 0 {
		return fmt.Errorf("at least one pattern is required")
	}

	var envKey = os.Getenv("GH_APP_PRIVATE_KEY")
	if *useFilesystem && envKey != "" && keyFile != "" {
		return ErrConflictingKeyOptions
	}

	// Handle storage method flags
	if *useFilesystem {
		*useKeyring = false
	}

	return nil
}

// getPrivateKey retrieves and validates the private key from environment or file
func getPrivateKey(keyFile string) (string, string, error) {
	var privateKeyContent string
	var expandedKeyFile string

	var envKey = os.Getenv("GH_APP_PRIVATE_KEY")
	if envKey != "" {
		privateKeyContent = envKey
	} else if keyFile != "" {
		var err error
		expandedKeyFile, err = expandPath(keyFile)
		if err != nil {
			return "", "", fmt.Errorf("invalid key file path: %w", err)
		}

		// Verify key file exists
		if err := validateKeyFile(expandedKeyFile); err != nil {
			return "", "", fmt.Errorf("key file validation failed: %w", err)
		}

		// Read key content
		keyData, err := os.ReadFile(expandedKeyFile)
		if err != nil {
			return "", "", fmt.Errorf("failed to read key file: %w", err)
		}
		privateKeyContent = string(keyData)
	} else {
		return "", "", fmt.Errorf("private key required: use --key-file or set GH_APP_PRIVATE_KEY environment variable")
	}

	return privateKeyContent, expandedKeyFile, nil
}

// generateJWTForSetup generates a JWT token and returns it for use in setup
func generateJWTForSetup(appID int64, privateKeyContent string) (string, error) {
	generator := jwt.NewGenerator()
	token, err := generator.GenerateTokenFromKey(appID, privateKeyContent)
	if err != nil {
		return "", fmt.Errorf("JWT generation test failed: %w", err)
	}
	return token, nil
}

// autoDetectInstallationID finds the installation ID for the GitHub App using the patterns
func autoDetectInstallationID(jwtToken string, patterns []string) (int64, error) {
	if len(patterns) == 0 {
		return 0, fmt.Errorf("no patterns provided")
	}

	// Extract host and org from the first pattern
	// Pattern format: "github.com/org/*" or "github.example.com/org/*"
	pattern := patterns[0]
	host, org, err := parsePatternForInstallation(pattern)
	if err != nil {
		return 0, err
	}

	// Try to find installation for the org
	installationID, err := findInstallationForOrg(jwtToken, host, org)
	if err != nil {
		return 0, err
	}

	return installationID, nil
}

// parsePatternForInstallation extracts host and org from a pattern
func parsePatternForInstallation(pattern string) (host, org string, err error) {
	// Remove trailing wildcards and slashes
	pattern = strings.TrimSuffix(pattern, "/*")
	pattern = strings.TrimSuffix(pattern, "/")

	// Split by /
	parts := strings.Split(pattern, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid pattern format: %s (expected host/org or host/org/*)", pattern)
	}

	host = parts[0]
	org = parts[1]

	if host == "" || org == "" {
		return "", "", fmt.Errorf("invalid pattern: host and org are required")
	}

	return host, org, nil
}

// findInstallationForOrg finds the installation ID for a GitHub App in an organization
func findInstallationForOrg(jwtToken, host, org string) (int64, error) {
	// Construct API URL for listing installations
	var apiURL string
	if host == gitHubAPIHost {
		apiURL = "https://api.github.com/app/installations"
	} else {
		apiURL = fmt.Sprintf("https://%s/api/v3/app/installations", host)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to list installations: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var installations []struct {
		ID      int64 `json:"id"`
		Account struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"account"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&installations); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	// Find installation matching the org
	for _, inst := range installations {
		if strings.EqualFold(inst.Account.Login, org) {
			return inst.ID, nil
		}
	}

	// If no match found, provide helpful error
	if len(installations) == 0 {
		return 0, fmt.Errorf("no installations found for this GitHub App")
	}

	availableOrgs := make([]string, 0, len(installations))
	for _, inst := range installations {
		availableOrgs = append(availableOrgs, inst.Account.Login)
	}
	return 0, fmt.Errorf("no installation found for org '%s'. Available: %s", org, strings.Join(availableOrgs, ", "))
}

// createGitHubApp creates a GitHub App configuration
// Note: Validation is done later after storage configuration is complete
func createGitHubApp(
	appID int64, name string, installationID int64, patterns []string, priority int,
) config.GitHubApp {
	// Set default name if not provided
	if name == "" {
		name = fmt.Sprintf("GitHub App %d", appID)
	}

	// Create GitHub App configuration
	return config.GitHubApp{
		Name:           name,
		AppID:          appID,
		InstallationID: installationID,
		Patterns:       patterns,
		Priority:       priority,
	}
}

// configureAppStorage configures the storage for the GitHub App's private key
func configureAppStorage(
	app *config.GitHubApp, privateKeyContent, expandedKeyFile string, useKeyring bool,
) (secrets.StorageBackend, error) {
	// Use XDG config directory for secrets
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth")
	secretMgr := secrets.NewManager(configDir)

	var backend secrets.StorageBackend
	if useKeyring {
		// Try keyring storage
		backend, err = app.SetPrivateKey(secretMgr, privateKeyContent)
		if err != nil {
			return "", fmt.Errorf("failed to store private key: %w", err)
		}
	} else {
		// Use filesystem storage
		if expandedKeyFile != "" {
			app.PrivateKeyPath = expandedKeyFile
			app.PrivateKeySource = config.PrivateKeySourceFilesystem
			backend = secrets.StorageBackendFilesystem
		} else {
			return "", fmt.Errorf("filesystem storage requires --key-file")
		}
	}

	return backend, nil
}

// saveAppConfiguration saves the GitHub App configuration
func saveAppConfiguration(cfg *config.Config, app *config.GitHubApp) error {
	// Add or update the app in configuration
	cfg.AddOrUpdateApp(app)

	// Save configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

// displaySetupSuccess displays the success message and next steps
func displaySetupSuccess(
	name string, appID int64, patterns []string, priority int,
	backend secrets.StorageBackend, expandedKeyFile string,
) {
	fmt.Printf("✅ Successfully configured GitHub App '%s'\n", name)
	fmt.Printf("   App ID: %d\n", appID)
	fmt.Printf("   Patterns: %s\n", strings.Join(patterns, ", "))
	fmt.Printf("   Priority: %d\n", priority)

	// Display storage information
	if backend == secrets.StorageBackendKeyring {
		fmt.Printf("   🔐 Storage: OS Keyring (encrypted)\n")
		if expandedKeyFile != "" {
			fmt.Printf("   📝 Fallback: %s\n", expandedKeyFile)
		}
	} else {
		fmt.Printf("   📁 Storage: Filesystem\n")
		fmt.Printf("   📝 Key file: %s\n", expandedKeyFile)
		fmt.Printf("   ⚠️  Keyring unavailable - using filesystem storage\n")
	}

	fmt.Printf("\n💡 Next steps:\n")
	fmt.Printf("   1. Test authentication: gh app-auth test --repo <repository-url>\n")
	fmt.Printf("   2. Sync git credential helper: gh app-auth gitconfig --sync --global\n")
	fmt.Printf("      # or run 'gh app-auth gitconfig --sync --local' inside a repository\n")
}

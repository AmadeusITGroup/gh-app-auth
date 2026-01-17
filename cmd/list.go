package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/secrets"
	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Status messages for key/token verification
const (
	statusNotChecked = "⚠️  Not checked"
	statusAccessible = "✅ Accessible"
	statusNotFound   = "❌ Not found"
)

func NewListCmd() *cobra.Command {
	var (
		format     string
		quiet      bool
		verifyKeys bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured credentials",
		Long: `List all configured GitHub Apps and Personal Access Tokens with their settings.

Shows the configured credential sources, their patterns, priorities, and status.`,
		Aliases: []string{"ls"},
		Example: `  # List all configured apps
  gh app-auth list
  
  # List with JSON output
  gh app-auth list --format json
  
  # Quiet output (just app IDs)
  gh app-auth list --quiet`,
		RunE: listRun(&format, &quiet, &verifyKeys),
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json, yaml")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only show app IDs")
	cmd.Flags().BoolVar(&verifyKeys, "verify-keys", false, "Verify that private keys are accessible")

	return cmd
}

func listRun(format *string, quiet *bool, verifyKeys *bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Load and validate configuration
		cfg, err := loadListConfiguration()

		if err != nil {
			return err
		}
		if cfg == nil {
			return nil // No apps configured
		}

		// Handle quiet mode
		if *quiet {
			return outputQuietMode(cfg.GitHubApps, cfg.PATs)
		}

		// Initialize secrets manager if needed
		secretMgr, err := initializeSecretsManagerIfNeeded(*verifyKeys)
		if err != nil {
			return err
		}

		// Handle output format
		return handleOutputFormat(*format, cfg.GitHubApps, cfg.PATs, secretMgr, *verifyKeys)
	}
}

func outputTable(
	apps []config.GitHubApp, pats []config.PersonalAccessToken,
	secretMgr *secrets.Manager, verifyKeys bool,
) error {
	printedSection := false

	if len(apps) > 0 {
		fmt.Println("GitHub Apps")
		if err := outputAppsTable(apps, secretMgr, verifyKeys); err != nil {
			return err
		}
		printedSection = true
	}

	if len(pats) > 0 {
		if printedSection {
			fmt.Println()
		}
		fmt.Println("Personal Access Tokens")
		if err := outputPATTable(pats, secretMgr, verifyKeys); err != nil {
			return err
		}
		printedSection = true
	}

	if !printedSection {
		fmt.Println("No GitHub Apps or Personal Access Tokens configured.")
	}

	return nil
}

func outputAppsTable(apps []config.GitHubApp, secretMgr *secrets.Manager, verifyKeys bool) error {
	// Create table printer
	terminal := os.Stdout
	width := 120 // Default width
	tp := tableprinter.New(terminal, false, width)

	// Add headers
	tp.AddField("NAME", tableprinter.WithTruncate(nil))
	tp.AddField("APP ID", tableprinter.WithTruncate(nil))
	tp.AddField("INSTALLATION ID", tableprinter.WithTruncate(nil))
	tp.AddField("PATTERNS", tableprinter.WithTruncate(nil))
	tp.AddField("PRIORITY", tableprinter.WithTruncate(nil))
	tp.AddField("KEY SOURCE", tableprinter.WithTruncate(nil))
	if verifyKeys {
		tp.AddField("KEY STATUS", tableprinter.WithTruncate(nil))
	}
	tp.EndRow()

	// Add data rows
	for _, app := range apps {
		tp.AddField(app.Name, tableprinter.WithTruncate(nil))
		tp.AddField(fmt.Sprintf("%d", app.AppID), tableprinter.WithTruncate(nil))

		installationDisplay := "auto-detect"
		if app.InstallationID > 0 {
			installationDisplay = fmt.Sprintf("%d", app.InstallationID)
		}
		tp.AddField(installationDisplay, tableprinter.WithTruncate(nil))

		tp.AddField(strings.Join(app.Patterns, ", "), tableprinter.WithTruncate(nil))
		tp.AddField(fmt.Sprintf("%d", app.Priority), tableprinter.WithTruncate(nil))

		// Display key source
		keySource := getKeySourceDisplay(app)
		tp.AddField(keySource, tableprinter.WithTruncate(nil))

		// Verify key if requested
		if verifyKeys {
			keyStatus := verifyKeyAccess(app, secretMgr)
			tp.AddField(keyStatus, tableprinter.WithTruncate(nil))
		}

		tp.EndRow()
	}

	return tp.Render()
}

func outputPATTable(pats []config.PersonalAccessToken, secretMgr *secrets.Manager, verifyTokens bool) error {
	terminal := os.Stdout
	width := 120
	tb := tableprinter.New(terminal, false, width)

	tb.AddField("NAME", tableprinter.WithTruncate(nil))
	tb.AddField("PATTERNS", tableprinter.WithTruncate(nil))
	tb.AddField("PRIORITY", tableprinter.WithTruncate(nil))
	tb.AddField("USERNAME", tableprinter.WithTruncate(nil))
	tb.AddField("TOKEN SOURCE", tableprinter.WithTruncate(nil))
	if verifyTokens {
		tb.AddField("TOKEN STATUS", tableprinter.WithTruncate(nil))
	}
	tb.EndRow()

	for _, pat := range pats {
		tb.AddField(pat.Name, tableprinter.WithTruncate(nil))
		tb.AddField(strings.Join(pat.Patterns, ", "), tableprinter.WithTruncate(nil))
		tb.AddField(fmt.Sprintf("%d", pat.Priority), tableprinter.WithTruncate(nil))

		username := pat.Username
		if username == "" {
			username = "x-access-token"
		}
		tb.AddField(username, tableprinter.WithTruncate(nil))
		tb.AddField(getPATSourceDisplay(pat), tableprinter.WithTruncate(nil))

		if verifyTokens {
			tb.AddField(verifyPATAccess(pat, secretMgr), tableprinter.WithTruncate(nil))
		}

		tb.EndRow()
	}

	return tb.Render()
}

type listOutput struct {
	GitHubApps []config.GitHubApp           `json:"github_apps" yaml:"github_apps"`
	PATs       []config.PersonalAccessToken `json:"pats" yaml:"pats"`
}

func outputJSON(apps []config.GitHubApp, pats []config.PersonalAccessToken) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(listOutput{GitHubApps: apps, PATs: pats})
}

func outputYAML(apps []config.GitHubApp, pats []config.PersonalAccessToken) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(listOutput{GitHubApps: apps, PATs: pats})
}

// getKeySourceDisplay returns a human-readable display of the key source
func getKeySourceDisplay(app config.GitHubApp) string {
	switch app.PrivateKeySource {
	case config.PrivateKeySourceKeyring:
		return "🔐 Keyring (encrypted)"
	case config.PrivateKeySourceFilesystem:
		if app.PrivateKeyPath != "" {
			return fmt.Sprintf("📁 %s", app.PrivateKeyPath)
		}
		return "📁 Filesystem"
	case config.PrivateKeySourceInline:
		return "⚠️  Inline (migrate)"
	case "":
		// Legacy config - check if path exists
		if app.PrivateKeyPath != "" {
			return fmt.Sprintf("📁 %s (legacy)", app.PrivateKeyPath)
		}
		return "❓ Unknown"
	default:
		return fmt.Sprintf("❓ %s", app.PrivateKeySource)
	}
}

func getPATSourceDisplay(pat config.PersonalAccessToken) string {
	switch pat.TokenSource {
	case config.PrivateKeySourceKeyring, "":
		return "🔐 Keyring (encrypted)"
	case config.PrivateKeySourceFilesystem:
		return "📁 Filesystem"
	default:
		return fmt.Sprintf("❓ %s", pat.TokenSource)
	}
}

// verifyKeyAccess checks if the private key is accessible
func verifyKeyAccess(app config.GitHubApp, secretMgr *secrets.Manager) string {
	if secretMgr == nil {
		return statusNotChecked
	}

	if app.HasPrivateKey(secretMgr) {
		return statusAccessible
	}

	return statusNotFound
}

func verifyPATAccess(pat config.PersonalAccessToken, secretMgr *secrets.Manager) string {
	if secretMgr == nil {
		return statusNotChecked
	}

	if _, _, err := secretMgr.Get(pat.Name, secrets.SecretTypePAT); err == nil {
		return statusAccessible
	}

	return statusNotFound
}

// loadListConfiguration loads and validates the configuration for listing
func loadListConfiguration() (*config.Config, error) {
	cfg, err := config.Load()
	if errors.Is(err, config.ErrConfigNotExists) {
		fmt.Printf("No GitHub Apps configured. Run 'gh app-auth setup' to add one.\n")
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if len(cfg.GitHubApps) == 0 && len(cfg.PATs) == 0 {
		fmt.Printf("No GitHub Apps or Personal Access Tokens configured. Run 'gh app-auth setup' to add one.\n")
		return nil, nil
	}

	return cfg, nil
}

// outputQuietMode outputs app IDs in quiet mode
func outputQuietMode(apps []config.GitHubApp, pats []config.PersonalAccessToken) error {
	for _, app := range apps {
		fmt.Printf("app:%d\n", app.AppID)
	}
	for _, pat := range pats {
		fmt.Printf("pat:%s\n", pat.Name)
	}
	return nil
}

// initializeSecretsManagerIfNeeded initializes secrets manager if key verification is needed
func initializeSecretsManagerIfNeeded(verifyKeys bool) (*secrets.Manager, error) {
	if !verifyKeys {
		return nil, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth")
	return secrets.NewManager(configDir), nil
}

// handleOutputFormat handles different output formats
func handleOutputFormat(
	format string, apps []config.GitHubApp, pats []config.PersonalAccessToken,
	secretMgr *secrets.Manager, verifyKeys bool,
) error {
	switch format {
	case "json":
		return outputJSON(apps, pats)
	case "yaml":
		return outputYAML(apps, pats)
	case "table":
		return outputTable(apps, pats, secretMgr, verifyKeys)
	default:
		return fmt.Errorf("unsupported format: %s (supported: table, json, yaml)", format)
	}
}

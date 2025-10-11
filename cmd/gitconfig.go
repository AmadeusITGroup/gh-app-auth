package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"github.com/spf13/cobra"
)

func NewGitConfigCmd() *cobra.Command {
	var (
		sync   bool
		clean  bool
		global bool
		local  bool
	)

	cmd := &cobra.Command{
		Use:   "gitconfig",
		Short: "Manage git credential helper configuration",
		Long: `Manage git credential helper configuration for gh-app-auth.

This command automates the setup and cleanup of git credential helpers
based on your configured GitHub Apps and Personal Access Tokens (PATs).
It simplifies the manual process of configuring git to use gh-app-auth
for authentication.

The command can operate at two scopes:
  --global: Configure git globally (default)
  --local:  Configure git in the current repository only`,
		Example: `  # Sync git config with all configured apps
  gh app-auth gitconfig --sync

  # Clean up all gh-app-auth git configurations
  gh app-auth gitconfig --clean

  # Sync only for current repository
  gh app-auth gitconfig --sync --local

  # Clean global and check status
  gh app-auth gitconfig --clean --global`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate flags
			if !sync && !clean {
				return fmt.Errorf("must specify either --sync or --clean")
			}
			if sync && clean {
				return fmt.Errorf("cannot use --sync and --clean together")
			}
			if global && local {
				return fmt.Errorf("cannot use --global and --local together")
			}

			// Default to global if neither specified
			if !global && !local {
				global = true
			}

			scope := "--global"
			if local {
				scope = "--local"
			}

			if sync {
				return syncGitConfig(scope)
			}
			return cleanGitConfig(scope)
		},
	}

	cmd.Flags().BoolVar(&sync, "sync", false, "Sync git config with configured apps")
	cmd.Flags().BoolVar(&clean, "clean", false, "Remove all gh-app-auth git configurations")
	cmd.Flags().BoolVar(&global, "global", false, "Configure git globally (default)")
	cmd.Flags().BoolVar(&local, "local", false, "Configure git in current repository only")

	return cmd
}

func syncGitConfig(scope string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if len(cfg.GitHubApps) == 0 && len(cfg.PATs) == 0 {
		return fmt.Errorf("no GitHub Apps or Personal Access Tokens configured. Run 'gh app-auth setup' first")
	}

	// Get the path to gh-app-auth executable
	execPath, err := getExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to locate gh-app-auth executable: %w", err)
	}

	fmt.Printf("Configuring git credential helpers (%s)...\n\n", scope)

	// Track configured patterns to avoid duplicates
	configured := make(map[string]bool)

	configurePattern := func(pattern, source string) {
		if configured[pattern] {
			return
		}

		// Extract credential context from pattern
		context := extractCredentialContext(pattern)
		if context == "" {
			fmt.Printf("⚠️  Skipping invalid pattern: %s\n", pattern)
			return
		}

		// Clear existing helpers for this context
		credKey := fmt.Sprintf("credential.%s.helper", context)
		clearCmd := exec.Command("git", "config", scope, "--unset-all", credKey)
		_ = clearCmd.Run() // Ignore error if nothing to unset

		// Set gh-app-auth as the credential helper with pattern matching
		patternArg := pattern
		if strings.ContainsAny(pattern, "*?[] ") {
			patternArg = fmt.Sprintf("\"%s\"", pattern)
		}
		helperValue := fmt.Sprintf("!%s git-credential --pattern %s", execPath, patternArg)
		setCmd := exec.Command("git", "config", scope, "--add", credKey, helperValue)
		if err := setCmd.Run(); err != nil {
			fmt.Printf("❌ Failed to configure: %s\n", context)
			fmt.Printf("   Pattern: %s\n", pattern)
			fmt.Printf("   Source: %s\n", source)
			fmt.Printf("   Error: %v\n\n", err)
			return
		}

		fmt.Printf("✅ Configured: %s\n", context)
		fmt.Printf("   Source: %s\n", source)
		fmt.Printf("   Pattern: %s\n\n", pattern)

		configured[pattern] = true
	}

	for _, app := range cfg.GitHubApps {
		for _, pattern := range app.Patterns {
			source := fmt.Sprintf("GitHub App %s (ID: %d)", app.Name, app.AppID)
			configurePattern(pattern, source)
		}
	}

	for _, pat := range cfg.PATs {
		for _, pattern := range pat.Patterns {
			source := fmt.Sprintf("Personal Access Token %s", pat.Name)
			configurePattern(pattern, source)
		}
	}

	if len(configured) == 0 {
		return fmt.Errorf("no valid patterns found to configure")
	}

	fmt.Printf("✨ Successfully configured %d credential helper(s)\n\n", len(configured))
	fmt.Println("You can now use git commands and they will authenticate using gh-app-auth:")
	fmt.Println("  git clone https://github.com/org/repo")
	fmt.Println("  git submodule update --init --recursive")

	return nil
}

func cleanGitConfig(scope string) error {
	fmt.Printf("Cleaning gh-app-auth git configurations (%s)...\n\n", scope)

	// Get all git config entries
	listCmd := exec.Command("git", "config", scope, "--get-regexp", "^credential\\..*\\.helper$")
	output, err := listCmd.Output()
	if err != nil {
		// git config returns exit code 1 when no matches found - this is expected, not an error
		fmt.Println("✨ No gh-app-auth configurations found")
		//nolint:nilerr // No configurations found is success case
		return nil
	}

	lines := strings.Split(string(output), "\n")
	removed := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse: credential.https://github.com/org.helper !gh-app-auth git-credential...
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Check if this is a gh-app-auth helper
		if strings.Contains(value, "gh-app-auth") || strings.Contains(value, "gh app-auth") {
			// Extract the context from the key
			context := strings.TrimPrefix(key, "credential.")
			context = strings.TrimSuffix(context, ".helper")

			unsetCmd := exec.Command("git", "config", scope, "--unset-all", key)
			if err := unsetCmd.Run(); err != nil {
				fmt.Printf("⚠️  Failed to remove: %s\n", context)
				continue
			}

			fmt.Printf("🗑️  Removed: %s\n", context)
			removed++
		}
	}

	if removed == 0 {
		fmt.Println("✨ No gh-app-auth configurations found")
	} else {
		fmt.Printf("\n✨ Successfully removed %d credential helper(s)\n", removed)
	}

	return nil
}

func extractCredentialContext(pattern string) string {
	// Remove wildcards and extract base URL
	// Examples:
	//   github.com/myorg/* -> https://github.com/myorg
	//   github.enterprise.com/*/* -> https://github.enterprise.com
	//   github.com/org/repo -> https://github.com/org

	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return ""
	}

	// Remove protocol if present
	pattern = strings.TrimPrefix(pattern, "https://")
	pattern = strings.TrimPrefix(pattern, "http://")

	// Split by /
	parts := strings.Split(pattern, "/")
	if len(parts) == 0 {
		return ""
	}

	// Validate that this looks like a GitHub URL pattern
	// It should have at least a host, and if it has more parts, they should be valid
	host := parts[0]
	if host == "" || !strings.Contains(host, ".") {
		// Invalid: no dots in hostname (not a valid domain)
		return ""
	}

	// Build context based on pattern specificity
	// For github.com/org/* we want https://github.com/org
	// For github.com/org/repo we want https://github.com/org
	// For github.enterprise.com/*/* we want https://github.enterprise.com

	// Check if we have organization-level pattern
	if len(parts) >= 2 && parts[1] != "*" {
		// Include organization: github.com/org
		return fmt.Sprintf("https://%s/%s", host, parts[1])
	}

	// Host-level only (valid for patterns like "github.com" or "github.com/*")
	return fmt.Sprintf("https://%s", host)
}

func getExecutablePath() (string, error) {
	// Try to get the current executable path
	execPath, err := os.Executable()
	if err != nil {
		// Fallback to searching in PATH
		execPath, err = exec.LookPath("gh-app-auth")
		if err != nil {
			// Last resort: try gh extension path
			homeDir, _ := os.UserHomeDir()
			localPath := filepath.Join(homeDir, ".local", "share", "gh", "extensions", "gh-app-auth", "gh-app-auth")
			if _, err := os.Stat(localPath); err == nil {
				return localPath, nil
			}
			return "", fmt.Errorf("gh-app-auth executable not found in PATH or extension directory")
		}
	}

	// Resolve symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", err
	}

	return execPath, nil
}

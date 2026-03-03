package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"github.com/spf13/cobra"
)

const windowsOS = "windows"

func NewGitConfigCmd() *cobra.Command {
	var (
		sync   bool
		clean  bool
		global bool
		local  bool
		auto   bool
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
  --local:  Configure git in the current repository only
  --auto:  Configure git in auto-mode (globally). 
           A single GitHub App will be used to be configured automatically for each repository to clone`,
		Example: `  # Sync git config with all configured apps
  gh app-auth gitconfig --sync

  # Clean up all gh-app-auth git configurations
  gh app-auth gitconfig --clean

  # Sync only for current repository
  gh app-auth gitconfig --sync --local

  # Enable auto-mode, 2 environment variables need to be set: GH_APP_PRIVATE_KEY_PATH and GH_APP_ID
  gh app-auth gitconfig --sync --auto

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
			if (global && (local || auto)) || (auto && (global || local)) {
				return fmt.Errorf("cannot use --global, --local and --auto together")
			}

			// Default to global if neither specified
			if !global && !local && !auto {
				global = true
			}

			scope := "--global"
			if local {
				scope = "--local"
			}
			if auto {
				scope = "--global"
			}
			if sync {
				return syncGitConfig(scope, auto)
			}
			return cleanGitConfig(scope)
		},
	}

	cmd.Flags().BoolVar(&sync, "sync", false, "Sync git config with configured apps")
	cmd.Flags().BoolVar(&clean, "clean", false, "Remove all gh-app-auth git configurations")
	cmd.Flags().BoolVar(&global, "global", false, "Configure git globally (default)")
	cmd.Flags().BoolVar(&local, "local", false, "Configure git in current repository only")
	cmd.Flags().BoolVar(&auto, "auto", false, "Configure git in auto-mode")

	return cmd
}

func syncGitConfig(scope string, auto bool) error {
	// Load configuration
	cfg, err := config.LoadOrCreate()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if len(cfg.GitHubApps) == 0 && len(cfg.PATs) == 0 && !auto {
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
	// Track hosts that need useHttpPath enabled for path-based matching
	hostsNeedingHttpPath := make(map[string]bool)
	// Track existing generic host helpers that need to be re-added AFTER specific ones
	// This ensures path-specific helpers are checked first (git uses config order)
	genericHostHelpers := make(map[string][]string)

	// Save and remove existing generic host credential config (e.g., credential.https://github.com.*)
	// These will be re-added AFTER path-specific helpers to ensure correct precedence
	// Git uses config file order for credential matching, so we need generic hosts at the END
	saveAndRemoveGenericHostConfig := func(host string) {
		genericKey := fmt.Sprintf("credential.https://%s.helper", host)
		// Get all existing helpers for this host
		getCmd := exec.Command("git", "config", scope, "--get-all", genericKey)
		output, err := getCmd.Output()
		if err == nil && len(output) > 0 {
			helpers := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, h := range helpers {
				h = strings.TrimSpace(h)
				// Only save non-gh-app-auth helpers (we'll configure those ourselves)
				if h != "" && !strings.Contains(h, "gh-app-auth") {
					genericHostHelpers[host] = append(genericHostHelpers[host], h)
				}
			}
		}
		// Remove ALL config for this generic host section to force it to be re-created at end of file
		// This includes helper, useHttpPath, and any other settings
		unsetHelperCmd := exec.Command("git", "config", scope, "--remove-section", fmt.Sprintf("credential.https://%s", host))
		_ = unsetHelperCmd.Run() // Ignore error if section doesn't exist
	}

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

		// Build helper value - Windows needs a wrapper script for reliable execution
		var helperValue string
		if runtime.GOOS == windowsOS {
			// On Windows, create a batch file wrapper that git can execute directly
			helperValue = createWindowsCredentialWrapper(execPath, patternArg)
		} else {
			// Unix systems - Quote execPath to handle paths with spaces
			helperValue = fmt.Sprintf("!\"%s\" git-credential --pattern %s", execPath, patternArg)
		}
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

		// Track if this is a path-specific pattern (has org/repo in path)
		// If so, we need to enable useHttpPath for the host
		host := extractHost(pattern)
		if host != "" && context != fmt.Sprintf("https://%s", host) {
			// This is a path-specific context, we need useHttpPath on the base host
			hostsNeedingHttpPath[host] = true
		}
		if auto {
			hostsNeedingHttpPath[host] = true
		}
	}

	if auto {
		configurePattern("github.com", "Automatic mode")
		setUseHttpPath(scope, "github.com")
		return nil
	}

	// First pass: identify all hosts with path-specific patterns and save their generic helpers
	for _, app := range cfg.GitHubApps {
		for _, pattern := range app.Patterns {
			context := extractCredentialContext(pattern)
			host := extractHost(pattern)
			if host != "" && context != "" && context != fmt.Sprintf("https://%s", host) {
				// This is a path-specific pattern
				if _, saved := genericHostHelpers[host]; !saved {
					saveAndRemoveGenericHostConfig(host)
				}
			}
		}
	}
	for _, pat := range cfg.PATs {
		for _, pattern := range pat.Patterns {
			context := extractCredentialContext(pattern)
			host := extractHost(pattern)
			if host != "" && context != "" && context != fmt.Sprintf("https://%s", host) {
				if _, saved := genericHostHelpers[host]; !saved {
					saveAndRemoveGenericHostConfig(host)
				}
			}
		}
	}

	// Second pass: configure all patterns
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

	// Enable useHttpPath for hosts with path-specific patterns
	// This is critical for git to match path-specific credential helpers correctly
	for host := range hostsNeedingHttpPath {
		setUseHttpPath(scope, host)
	}

	// Restore generic host helpers AFTER path-specific ones
	// This ensures git checks path-specific helpers first (git uses config file order)
	for host, helpers := range genericHostHelpers {
		genericKey := fmt.Sprintf("credential.https://%s.helper", host)
		for _, helper := range helpers {
			addCmd := exec.Command("git", "config", scope, "--add", genericKey, helper)
			if err := addCmd.Run(); err != nil {
				fmt.Printf("⚠️  Warning: Failed to restore helper for %s: %v\n", host, err)
			}
		}
		if len(helpers) > 0 {
			fmt.Printf("🔄 Reordered credential helpers for %s (path-specific helpers now checked first)\n", host)
		}
	}

	if len(hostsNeedingHttpPath) > 0 || len(genericHostHelpers) > 0 {
		fmt.Println()
	}

	fmt.Printf("✨ Successfully configured %d credential helper(s)\n\n", len(configured))
	fmt.Println("You can now use git commands and they will authenticate using gh-app-auth:")
	fmt.Println("  git clone https://github.com/org/repo")
	fmt.Println("  git submodule update --init --recursive")

	return nil
}

func setUseHttpPath(scope string, host string) {
	useHttpPathKey := fmt.Sprintf("credential.https://%s.useHttpPath", host)
	setCmd := exec.Command("git", "config", scope, useHttpPathKey, "true")
	if err := setCmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Failed to enable useHttpPath for %s: %v\n", host, err)
	} else {
		fmt.Printf("🔧 Enabled useHttpPath for %s (required for path-based credential matching)\n", host)
	}
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

// createWindowsCredentialWrapper creates a batch file wrapper for Windows.
// Returns the path to the wrapper batch file which git can execute directly.
func createWindowsCredentialWrapper(execPath, patternArg string) string {
	// Create a unique wrapper name in user's temp directory
	hash := 0
	for _, c := range patternArg {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}

	tempDir := os.TempDir()
	wrapperName := fmt.Sprintf("gh-app-auth-cred-%d.bat", hash)
	wrapperPath := filepath.Join(tempDir, wrapperName)

	// Build batch file content
	// Use proper escaping for Windows batch files
	batchContent := fmt.Sprintf("@echo off\r\n\"%s\" git-credential --pattern %s %%*\r\n", execPath, patternArg)

	// Write the wrapper script (overwrite if exists)
	if err := os.WriteFile(wrapperPath, []byte(batchContent), 0600); err != nil {
		// Fallback: return direct command if we can't create wrapper
		return fmt.Sprintf("\"%s\" git-credential --pattern %s", execPath, patternArg)
	}

	// Convert to forward slashes for git config - Windows accepts both,
	// but backslashes get stripped by shell interpretation
	return strings.ReplaceAll(wrapperPath, "\\", "/")
}

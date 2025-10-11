package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/auth"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/secrets"
	"github.com/spf13/cobra"
)

func NewTestCmd() *cobra.Command {
	var (
		repo    string
		verbose bool
	)

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test gh-app-auth credentials",
		Long: `Test authentication for a specific repository using either a GitHub App or Personal Access Token.

For GitHub Apps this command verifies that:
1. A GitHub App is configured for the repository pattern
2. JWT token generation works with the private key
3. Installation token can be retrieved
4. GitHub API access works with the token

For Personal Access Tokens this command verifies that:
1. A PAT is configured for the repository pattern
2. The token can be retrieved from secure storage
3. GitHub API access works with the token`,
		Example: `  # Test authentication for a repository
  gh app-auth test --repo github.com/myorg/myrepo
  
  # Test with verbose output
  gh app-auth test --repo github.com/myorg/myrepo --verbose
  
  # Test current repository (if in git directory)
  gh app-auth test`,
		RunE: testRun(&repo, &verbose),
	}

	cmd.Flags().StringVarP(&repo, "repo", "R", "", "Repository to test (default: current repository)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed test output")

	return cmd
}

func newDefaultSecretsManager() (*secrets.Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth")
	return secrets.NewManager(configDir), nil
}

func testRun(repo *string, verbose *bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := loadTestConfiguration()
		if err != nil {
			return err
		}

		// Determine repository URL
		repoURL, err := determineRepositoryURL(*repo)
		if err != nil {
			return err
		}

		if *verbose {
			fmt.Printf("🔍 Testing authentication for repository: %s\n\n", repoURL)
		}

		// Run authentication tests
		if err := runAuthenticationTests(cfg, repoURL, *verbose); err != nil {
			return err
		}

		// Display success message and usage instructions
		displayTestResults(*verbose)

		return nil
	}
}

func getCurrentRepository() (string, error) {
	// This is a placeholder - in reality we'd use go-gh's repository detection
	return "", fmt.Errorf("current repository detection not implemented yet")
}

func testAPIAccess(token, repoURL string, verbose bool) error {
	// Extract host and owner/repo from URL
	host := extractHost(repoURL)
	owner, repo, err := extractOwnerRepo(repoURL)
	if err != nil {
		return fmt.Errorf("failed to parse repository URL: %w", err)
	}

	// Use raw HTTP instead of go-gh to avoid GitHub CLI auth requirement
	apiURL := fmt.Sprintf("https://%s/api/v3/repos/%s/%s", host, owner, repo)
	if host == "github.com" {
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	}

	// Make HTTP request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := &http.Client{}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "token "+token)
	httpReq.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var repoInfo struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Private  bool   `json:"private"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if verbose {
		fmt.Printf("   Repository: %s\n", repoInfo.FullName)
		fmt.Printf("   Private: %v\n", repoInfo.Private)
	}

	return nil
}

func extractHost(repoURL string) string {
	// Remove protocol and .git suffix
	repoURL = strings.TrimSuffix(repoURL, ".git")
	repoURL = strings.TrimPrefix(repoURL, "https://")
	repoURL = strings.TrimPrefix(repoURL, "http://")

	// Handle SSH format: git@github.com:owner/repo
	if strings.HasPrefix(repoURL, "git@") {
		repoURL = strings.TrimPrefix(repoURL, "git@")
		if colonIdx := strings.Index(repoURL, ":"); colonIdx > 0 {
			return repoURL[:colonIdx]
		}
	}

	// Handle HTTPS format: github.com/owner/repo
	parts := strings.Split(repoURL, "/")
	if len(parts) > 0 {
		return parts[0]
	}

	// Default to github.com
	return "github.com"
}

func extractOwnerRepo(repoURL string) (owner, repo string, err error) {
	// Parse repository URL to extract owner and repo
	// This is a simplified version - real implementation would handle various URL formats
	repoURL = strings.TrimSuffix(repoURL, ".git")
	repoURL = strings.TrimPrefix(repoURL, "https://")
	repoURL = strings.TrimPrefix(repoURL, "http://")
	repoURL = strings.TrimPrefix(repoURL, "git@")

	if strings.Contains(repoURL, ":") {
		// SSH format: git@github.com:owner/repo
		parts := strings.Split(repoURL, ":")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid repository URL format")
		}
		repoURL = parts[1]
	} else {
		// HTTPS format: github.com/owner/repo
		parts := strings.Split(repoURL, "/")
		if len(parts) < 3 {
			return "", "", fmt.Errorf("invalid repository URL format")
		}
		repoURL = strings.Join(parts[1:], "/")
	}

	parts := strings.Split(repoURL, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository URL format")
	}

	return parts[0], parts[1], nil
}

// loadTestConfiguration loads and validates the configuration for testing
func loadTestConfiguration() (*config.Config, error) {
	cfg, err := config.Load()
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("no GitHub Apps or Personal Access Tokens configured. Run 'gh app-auth setup' first")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if len(cfg.GitHubApps) == 0 && len(cfg.PATs) == 0 {
		return nil, fmt.Errorf("no GitHub Apps or Personal Access Tokens configured. Run 'gh app-auth setup' first")
	}

	return cfg, nil
}

// determineRepositoryURL determines the repository URL to test
func determineRepositoryURL(repo string) (string, error) {
	if repo != "" {
		return repo, nil
	}

	// Try to get current repository
	currentRepo, err := getCurrentRepository()
	if err != nil {
		return "", fmt.Errorf("no repository specified and not in a git repository. Use --repo flag")
	}
	return currentRepo, nil
}

// runAuthenticationTests runs all authentication tests

func runAuthenticationTests(cfg *config.Config, repoURL string, verbose bool) error {
	if verbose {
		fmt.Printf("Step 1: Finding matching credential...\n")
	}

	matchedApp, matchedPAT, err := findMatchingCredential(cfg, repoURL)
	if err != nil {
		return err
	}
	if matchedApp == nil && matchedPAT == nil {
		return fmt.Errorf("no matching GitHub App or Personal Access Token found for %s", repoURL)
	}

	if matchedPAT != nil {
		if verbose {
			fmt.Printf("✅ Found matching PAT: %s\n", matchedPAT.Name)
			fmt.Printf("   Patterns: %v\n", matchedPAT.Patterns)
			fmt.Printf("   Priority: %d\n\n", matchedPAT.Priority)
		} else {
			fmt.Printf("✅ Matched Personal Access Token: %s\n", matchedPAT.Name)
		}
		return runPATAuthenticationTests(matchedPAT, repoURL, verbose)
	}

	if verbose {
		fmt.Printf("✅ Found matching app: %s (ID: %d)\n", matchedApp.Name, matchedApp.AppID)
		fmt.Printf("   Patterns: %v\n", matchedApp.Patterns)
		fmt.Printf("   Priority: %d\n\n", matchedApp.Priority)
	} else {
		fmt.Printf("✅ Matched GitHub App: %s\n", matchedApp.Name)
	}

	return runGitHubAppAuthenticationTests(matchedApp, repoURL, verbose)
}

func runGitHubAppAuthenticationTests(matchedApp *config.GitHubApp, repoURL string, verbose bool) error {
	jwtToken, err := testJWTGeneration(matchedApp, verbose)
	if err != nil {
		return err
	}

	installationToken, err := testInstallationTokenGeneration(jwtToken, matchedApp, repoURL, verbose)
	if err != nil {
		return err
	}

	return testGitHubAPIAccess("Step 4: ", installationToken, repoURL, verbose)
}

func runPATAuthenticationTests(matchedPAT *config.PersonalAccessToken, repoURL string, verbose bool) error {
	secretMgr, err := newDefaultSecretsManager()
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("Step 2: Retrieving Personal Access Token...\n")
	}

	token, err := matchedPAT.GetPAT(secretMgr)
	if err != nil {
		return fmt.Errorf("failed to retrieve PAT: %w", err)
	}

	if verbose {
		fmt.Printf("✅ PAT retrieved successfully\n\n")
	} else {
		fmt.Printf("✅ PAT retrieved from secure storage\n")
	}

	return testGitHubAPIAccess("Step 3: ", token, repoURL, verbose)
}

// testJWTGeneration tests JWT token generation
func testJWTGeneration(matchedApp *config.GitHubApp, verbose bool) (string, error) {
	if verbose {
		fmt.Printf("Step 2: Testing JWT generation...\n")
	}

	authenticator := auth.NewAuthenticator()
	jwtToken, err := authenticator.GenerateJWTForApp(matchedApp)
	if err != nil {
		return "", fmt.Errorf("JWT generation failed: %w", err)
	}

	if verbose {
		fmt.Printf("✅ JWT token generated successfully\n")
		fmt.Printf("   Token length: %d characters\n\n", len(jwtToken))
	} else {
		fmt.Printf("✅ JWT generation successful\n")
	}

	return jwtToken, nil
}

// testInstallationTokenGeneration tests installation token generation
func testInstallationTokenGeneration(
	jwtToken string, matchedApp *config.GitHubApp, repoURL string, verbose bool,
) (string, error) {
	if verbose {
		fmt.Printf("Step 3: Testing installation token generation...\n")
	}

	authenticator := auth.NewAuthenticator()
	installationToken, err := authenticator.GetInstallationToken(jwtToken, matchedApp.InstallationID, repoURL)
	if err != nil {
		return "", fmt.Errorf("installation token generation failed: %w", err)
	}

	if verbose {
		fmt.Printf("✅ Installation token generated successfully\n")
		fmt.Printf("   Token length: %d characters\n\n", len(installationToken))
	} else {
		fmt.Printf("✅ Installation token generation successful\n")
	}

	return installationToken, nil
}

// testGitHubAPIAccess tests GitHub API access
func testGitHubAPIAccess(stepLabel string, token, repoURL string, verbose bool) error {
	if verbose {
		fmt.Printf("%sTesting GitHub API access...\n", stepLabel)
	}

	if err := testAPIAccess(token, repoURL, verbose); err != nil {
		return fmt.Errorf("GitHub API access test failed: %w", err)
	}

	if verbose {
		fmt.Printf("✅ GitHub API access successful\n\n")
	} else {
		fmt.Printf("✅ GitHub API access successful\n")
	}

	return nil
}

// displayTestResults shows the final test results and usage instructions
func displayTestResults(verbose bool) {
	fmt.Printf("\n🎉 All tests passed! Authentication is working correctly.\n")

	if !verbose {
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  gh app-auth gitconfig --sync --global\n")
		fmt.Printf("    # or run 'gh app-auth gitconfig --sync --local' inside a repository\n")
	}
}

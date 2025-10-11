package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/spf13/cobra"
	"github.com/wherka-ama/gh-app-auth/pkg/auth"
	"github.com/wherka-ama/gh-app-auth/pkg/config"
	"github.com/wherka-ama/gh-app-auth/pkg/matcher"
)

func NewTestCmd() *cobra.Command {
	var (
		repo    string
		verbose bool
	)

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test GitHub App authentication",
		Long: `Test GitHub App authentication for a specific repository.

This command verifies that:
1. A GitHub App is configured for the repository pattern
2. JWT token generation works with the private key
3. Installation token can be retrieved
4. GitHub API access works with the token`,
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

func testRun(repo *string, verbose *bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.Load()
		if os.IsNotExist(err) {
			return fmt.Errorf("no GitHub Apps configured. Run 'gh app-auth setup' first")
		}
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		if len(cfg.GitHubApps) == 0 {
			return fmt.Errorf("no GitHub Apps configured. Run 'gh app-auth setup' first")
		}

		// Determine repository URL
		repoURL := *repo
		if repoURL == "" {
			// Try to get current repository
			currentRepo, err := getCurrentRepository()
			if err != nil {
				return fmt.Errorf("no repository specified and not in a git repository. Use --repo flag")
			}
			repoURL = currentRepo
		}

		if *verbose {
			fmt.Printf("🔍 Testing authentication for repository: %s\n\n", repoURL)
		}

		// Step 1: Find matching GitHub App
		if *verbose {
			fmt.Printf("Step 1: Finding matching GitHub App...\n")
		}

		matcher := matcher.NewMatcher(cfg.GitHubApps)
		matchedApp, err := matcher.Match(repoURL)
		if err != nil {
			return fmt.Errorf("no matching GitHub App found for %s: %w", repoURL, err)
		}

		if *verbose {
			fmt.Printf("✅ Found matching app: %s (ID: %d)\n", matchedApp.Name, matchedApp.AppID)
			fmt.Printf("   Patterns: %v\n", matchedApp.Patterns)
			fmt.Printf("   Priority: %d\n\n", matchedApp.Priority)
		} else {
			fmt.Printf("✅ Matched GitHub App: %s\n", matchedApp.Name)
		}

		// Step 2: Test JWT generation
		if *verbose {
			fmt.Printf("Step 2: Testing JWT generation...\n")
		}

		authenticator := auth.NewAuthenticator()
		jwtToken, err := authenticator.GenerateJWT(matchedApp.AppID, matchedApp.PrivateKeyPath)
		if err != nil {
			return fmt.Errorf("JWT generation failed: %w", err)
		}

		if *verbose {
			fmt.Printf("✅ JWT token generated successfully\n")
			fmt.Printf("   Token length: %d characters\n\n", len(jwtToken))
		} else {
			fmt.Printf("✅ JWT generation successful\n")
		}

		// Step 3: Test installation token generation
		if *verbose {
			fmt.Printf("Step 3: Testing installation token generation...\n")
		}

		installationToken, err := authenticator.GetInstallationToken(jwtToken, matchedApp.InstallationID, repoURL)
		if err != nil {
			return fmt.Errorf("installation token generation failed: %w", err)
		}

		if *verbose {
			fmt.Printf("✅ Installation token generated successfully\n")
			fmt.Printf("   Token length: %d characters\n\n", len(installationToken))
		} else {
			fmt.Printf("✅ Installation token generation successful\n")
		}

		// Step 4: Test GitHub API access
		if *verbose {
			fmt.Printf("Step 4: Testing GitHub API access...\n")
		}

		if err := testAPIAccess(installationToken, repoURL, *verbose); err != nil {
			return fmt.Errorf("GitHub API access test failed: %w", err)
		}

		if *verbose {
			fmt.Printf("✅ GitHub API access successful\n\n")
		} else {
			fmt.Printf("✅ GitHub API access successful\n")
		}

		fmt.Printf("\n🎉 All tests passed! GitHub App authentication is working correctly.\n")
		
		if !*verbose {
			fmt.Printf("\nTo use this authentication:\n")
			fmt.Printf("  git config credential.\"https://%s\".helper \"app-auth git-credential\"\n", extractHost(repoURL))
		}

		return nil
	}
}

func getCurrentRepository() (string, error) {
	// This is a placeholder - in reality we'd use go-gh's repository detection
	return "", fmt.Errorf("current repository detection not implemented yet")
}

func testAPIAccess(token, repoURL string, verbose bool) error {
	// Create a REST client with the installation token
	client, err := api.NewRESTClient(api.ClientOptions{
		Headers: map[string]string{
			"Authorization": "token " + token,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Extract owner/repo from URL
	owner, repo, err := extractOwnerRepo(repoURL)
	if err != nil {
		return fmt.Errorf("failed to parse repository URL: %w", err)
	}

	// Test API access by getting repository information
	var repoInfo struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Private  bool   `json:"private"`
	}

	endpoint := fmt.Sprintf("repos/%s/%s", owner, repo)
	if err := client.Get(endpoint, &repoInfo); err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}

	if verbose {
		fmt.Printf("   Repository: %s\n", repoInfo.FullName)
		fmt.Printf("   Private: %v\n", repoInfo.Private)
	}

	return nil
}

func extractHost(repoURL string) string {
	// Simple host extraction - in practice this would be more robust
	if strings.Contains(repoURL, "github.com") {
		return "github.com"
	}
	// Handle enterprise GitHub instances
	parts := strings.Split(repoURL, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return repoURL
}

func extractOwnerRepo(repoURL string) (string, string, error) {
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

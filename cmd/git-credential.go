package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wherka-ama/gh-app-auth/pkg/auth"
	"github.com/wherka-ama/gh-app-auth/pkg/config"
	"github.com/wherka-ama/gh-app-auth/pkg/matcher"
)

func NewGitCredentialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "git-credential",
		Short: "Git credential helper for GitHub App authentication",
		Long: `Git credential helper that provides GitHub App authentication.

This command implements the git credential helper protocol and should not
be called directly. Instead, configure git to use it:

  git config credential.helper "app-auth git-credential"

Or for specific hosts:

  git config credential."https://github.com/myorg".helper "app-auth git-credential"`,
		Hidden: true, // Hide from general help as it's internal
		Args:   cobra.ExactArgs(1),
		RunE:   gitCredentialRun,
	}

	return cmd
}

func gitCredentialRun(cmd *cobra.Command, args []string) error {
	operation := args[0]

	switch operation {
	case "get":
		return handleCredentialGet()
	case "store":
		return handleCredentialStore()
	case "erase":
		return handleCredentialErase()
	default:
		return fmt.Errorf("unsupported git credential operation: %s", operation)
	}
}

func handleCredentialGet() error {
	// Read input from git
	input, err := readCredentialInput(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read credential input: %w", err)
	}

	// Build repository URL from input
	repoURL := buildRepositoryURL(input)
	if repoURL == "" {
		return fmt.Errorf("unable to determine repository URL from input")
	}

	// Load configuration
	cfg, err := config.Load()
	if os.IsNotExist(err) {
		// No configuration - exit silently to allow fallback to other credential helpers
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if len(cfg.GitHubApps) == 0 {
		// No apps configured - exit silently
		return nil
	}

	// Find matching GitHub App
	matcher := matcher.NewMatcher(cfg.GitHubApps)
	matchedApp, err := matcher.Match(repoURL)
	if err != nil {
		// No matching app - exit silently to allow fallback
		return nil
	}

	// Generate authentication token
	authenticator := auth.NewAuthenticator()
	token, username, err := authenticator.GetCredentials(matchedApp, repoURL)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Output credentials in git credential format
	fmt.Printf("username=%s\n", username)
	fmt.Printf("password=%s\n", token)

	return nil
}

func handleCredentialStore() error {
	// Read and ignore input - we don't need to store anything
	// since we generate tokens dynamically
	_, err := readCredentialInput(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read credential input: %w", err)
	}

	// Nothing to store for GitHub App authentication
	return nil
}

func handleCredentialErase() error {
	// Read input to get the repository information
	input, err := readCredentialInput(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read credential input: %w", err)
	}

	// Build repository URL from input
	repoURL := buildRepositoryURL(input)
	if repoURL == "" {
		return nil // Nothing to erase if we can't determine the repo
	}

	// Clear any cached tokens for this repository
	// This is a placeholder - actual implementation would clear cache
	return nil
}

func readCredentialInput(reader io.Reader) (map[string]string, error) {
	input := make(map[string]string)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break // Empty line marks end of input
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			input[parts[0]] = parts[1]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return input, nil
}

func buildRepositoryURL(input map[string]string) string {
	protocol := input["protocol"]
	host := input["host"]
	path := input["path"]

	if host == "" {
		return ""
	}

	// Default to https if no protocol specified
	if protocol == "" {
		protocol = "https"
	}

	if path == "" {
		return fmt.Sprintf("%s://%s", protocol, host)
	}

	// Clean up path
	path = strings.Trim(path, "/")
	path = strings.TrimSuffix(path, ".git")

	return fmt.Sprintf("%s://%s/%s", protocol, host, path)
}

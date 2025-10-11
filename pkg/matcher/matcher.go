package matcher

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

// ErrNoMatchingApp is returned when no GitHub App matches the repository URL
var ErrNoMatchingApp = errors.New("no matching GitHub App found")

// Matcher handles pattern matching for GitHub App configurations
type Matcher struct {
	apps []config.GitHubApp
}

// NewMatcher creates a new pattern matcher with the given GitHub App configurations
func NewMatcher(apps []config.GitHubApp) *Matcher {
	return &Matcher{
		apps: apps,
	}
}

// RepositoryInfo contains parsed repository information
type RepositoryInfo struct {
	Host       string
	Owner      string
	Repository string
	FullPath   string // host/owner/repo
	URL        string // original URL
}

// Match finds the best matching GitHub App for the given repository URL
// Uses longest prefix matching - the app with the longest matching path prefix wins
// If scope information is available, validates that the repo is within the app's installation scope
func (m *Matcher) Match(repositoryURL string) (*config.GitHubApp, error) {
	if len(m.apps) == 0 {
		return nil, nil
	}

	// Parse repository information to get the path
	repoInfo, err := parseRepositoryURL(repositoryURL)
	if err != nil {
		// If we can't parse the full URL (e.g., just "github.com"), try host-only matching
		// This is intentional - we silently fall back to host matching for partial inputs
		app := m.matchByHost(repositoryURL)
		if app == nil {
			return nil, fmt.Errorf("failed to parse repository URL: %w", err)
		}
		return app, nil
	}

	repoPath := repoInfo.FullPath // e.g., "github.com/org/repo"

	// Find the app with the longest matching prefix
	var bestMatch *config.GitHubApp
	longestPrefixLen := 0

	for i := range m.apps {
		app := &m.apps[i]

		for _, pattern := range app.Patterns {
			// Strip trailing /* for backward compatibility with old configs
			prefix := strings.TrimSuffix(pattern, "/*")
			prefix = strings.TrimSpace(prefix)

			if prefix == "" {
				continue
			}

			// Check if this prefix matches
			if strings.HasPrefix(repoPath, prefix) {
				// If scope info is available, validate repo is in scope
				if app.Scope != nil && !isInScope(repoPath, app.Scope) {
					continue // Skip - repo not in installation scope
				}

				// Use longest prefix for best match
				if len(prefix) > longestPrefixLen {
					longestPrefixLen = len(prefix)
					bestMatch = app
				}
			}
		}
	}

	return bestMatch, nil
}

// matchByHost matches apps when only a host is provided (e.g., "github.com")
// Returns the first app that has a pattern matching the host
func (m *Matcher) matchByHost(host string) *config.GitHubApp {
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimSuffix(host, "/")

	for i := range m.apps {
		app := &m.apps[i]
		for _, pattern := range app.Patterns {
			// Check if pattern starts with this host
			if strings.HasPrefix(pattern, host+"/") || pattern == host {
				return app
			}
		}
	}

	return nil
}

// isInScope checks if a repository is within the app's installation scope
func isInScope(repoPath string, scope *config.InstallationScope) bool {
	if scope.RepositorySelection == "all" {
		// Check if repo is under the account
		// e.g., "github.com/myorg/repo" should match account "myorg"
		parts := strings.Split(repoPath, "/")
		if len(parts) < 2 {
			return false
		}
		// parts[0] = host, parts[1] = owner
		return parts[1] == scope.AccountLogin
	}

	// repository_selection == "selected"
	// Extract owner/repo from path
	parts := strings.Split(repoPath, "/")
	if len(parts) < 3 {
		return false
	}
	fullName := fmt.Sprintf("%s/%s", parts[1], parts[2])

	// Check if repo is in the list
	for _, repo := range scope.Repositories {
		if repo.FullName == fullName {
			return true
		}
	}

	return false
}

// Legacy functions removed - no longer needed with simple prefix matching
// Old logic used complex glob patterns, wildcards, and manual priority
// New logic: longest matching prefix wins automatically

// parseRepositoryURL parses a Git repository URL and extracts relevant information
func parseRepositoryURL(repoURL string) (*RepositoryInfo, error) {
	if repoURL == "" {
		return nil, fmt.Errorf("repository URL cannot be empty")
	}

	// Handle different URL formats
	repoURL = normalizeURL(repoURL)

	// Parse as URL
	u, err := url.Parse(repoURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL format: %w", err)
	}

	// Extract host
	host := u.Host
	if host == "" {
		return nil, fmt.Errorf("no host found in URL")
	}

	// Extract path and parse owner/repo
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return nil, fmt.Errorf("no path found in URL")
	}

	// Remove .git suffix if present
	path = strings.TrimSuffix(path, ".git")

	// Split path into components
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid repository path: expected owner/repo format")
	}

	owner := parts[0]
	repo := parts[1]

	// Handle case where there are more path components (e.g., subgroups)
	if len(parts) > 2 {
		// For patterns like "host/group/subgroup/repo", we want to preserve the full path
		repo = strings.Join(parts[1:], "/")
	}

	fullPath := fmt.Sprintf("%s/%s/%s", host, owner, repo)

	return &RepositoryInfo{
		Host:       host,
		Owner:      owner,
		Repository: repo,
		FullPath:   fullPath,
		URL:        repoURL,
	}, nil
}

// normalizeURL normalizes different Git URL formats to a standard HTTPS format
func normalizeURL(repoURL string) string {
	// Handle SSH URLs like git@github.com:owner/repo.git
	if strings.HasPrefix(repoURL, "git@") {
		// Extract the part after @
		parts := strings.SplitN(repoURL, "@", 2)
		if len(parts) == 2 {
			// Replace : with / and prepend https://
			hostPath := strings.Replace(parts[1], ":", "/", 1)
			return "https://" + hostPath
		}
	}

	// Handle URLs that don't have a scheme
	if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
		// Assume it's a github.com repository in owner/repo format
		if !strings.Contains(repoURL, "/") {
			return repoURL // Return as-is if it doesn't look like a path
		}
		return "https://" + repoURL
	}

	return repoURL
}

// GetRepositoryInfo parses repository URL and returns the info (for testing/debugging)
func GetRepositoryInfo(repoURL string) (*RepositoryInfo, error) {
	return parseRepositoryURL(repoURL)
}

// No complex pattern matching needed - simple prefix comparison handles all cases

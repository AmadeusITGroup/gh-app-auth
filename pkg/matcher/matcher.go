package matcher

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/wherka-ama/gh-app-auth/pkg/config"
)

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
func (m *Matcher) Match(repositoryURL string) (*config.GitHubApp, error) {
	if len(m.apps) == 0 {
		return nil, nil
	}

	// Parse repository information
	repoInfo, err := parseRepositoryURL(repositoryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository URL: %w", err)
	}

	// Find all matching apps
	var matches []config.GitHubApp
	for _, app := range m.apps {
		if m.appMatches(app, repoInfo) {
			matches = append(matches, app)
		}
	}

	if len(matches) == 0 {
		return nil, nil
	}

	// Sort matches by priority and return the best one
	best := m.findBestMatch(matches)
	return &best, nil
}

// appMatches checks if a GitHub App's patterns match the repository info
func (m *Matcher) appMatches(app config.GitHubApp, repoInfo *RepositoryInfo) bool {
	for _, pattern := range app.Patterns {
		if m.patternMatches(pattern, repoInfo) {
			return true
		}
	}
	return false
}

// patternMatches checks if a pattern matches the repository info
func (m *Matcher) patternMatches(pattern string, repoInfo *RepositoryInfo) bool {
	// Normalize pattern and repository path
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}

	// Global wildcard matches everything
	if pattern == "*" {
		return true
	}

	// Test against full path (host/owner/repo)
	if matched, _ := filepath.Match(pattern, repoInfo.FullPath); matched {
		return true
	}

	// If pattern contains slashes, it might be a partial path pattern
	if strings.Contains(pattern, "/") {
		// Count slashes to determine what level we're matching
		slashCount := strings.Count(pattern, "/")
		
		switch slashCount {
		case 1:
			// Could be "host/*" or "owner/repo" format
			if strings.HasSuffix(pattern, "/*") {
				// Host-based pattern like "github.com/*"
				hostPattern := strings.TrimSuffix(pattern, "/*")
				if matched, _ := filepath.Match(hostPattern, repoInfo.Host); matched {
					return true
				}
			} else {
				// Owner/repo pattern like "myorg/myrepo"
				ownerRepo := fmt.Sprintf("%s/%s", repoInfo.Owner, repoInfo.Repository)
				if matched, _ := filepath.Match(pattern, ownerRepo); matched {
					return true
				}
			}
		case 2:
			// Full path pattern like "github.com/owner/*"
			if matched, _ := filepath.Match(pattern, repoInfo.FullPath); matched {
				return true
			}
		}
	} else {
		// No slashes - could be host only, repo only, or wildcard
		// Test against host
		if matched, _ := filepath.Match(pattern, repoInfo.Host); matched {
			return true
		}
		// Test against repo name only
		if matched, _ := filepath.Match(pattern, repoInfo.Repository); matched {
			return true
		}
	}

	return false
}

// findBestMatch returns the GitHub App with the highest priority
func (m *Matcher) findBestMatch(matches []config.GitHubApp) config.GitHubApp {
	if len(matches) == 1 {
		return matches[0]
	}

	best := matches[0]
	for i := 1; i < len(matches); i++ {
		if matches[i].Priority > best.Priority ||
			(matches[i].Priority == best.Priority && matches[i].Name < best.Name) {
			best = matches[i]
		}
	}

	return best
}

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
	if strings.HasSuffix(path, ".git") {
		path = strings.TrimSuffix(path, ".git")
	}

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

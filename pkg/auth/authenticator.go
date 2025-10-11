package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/wherka-ama/gh-app-auth/pkg/cache"
	"github.com/wherka-ama/gh-app-auth/pkg/config"
	"github.com/wherka-ama/gh-app-auth/pkg/jwt"
)

// Authenticator handles GitHub App authentication
type Authenticator struct {
	jwtGenerator *jwt.Generator
	tokenCache   *cache.TokenCache
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator() *Authenticator {
	return &Authenticator{
		jwtGenerator: jwt.NewGenerator(),
		tokenCache:   cache.NewTokenCache(),
	}
}

// GetCredentials returns username and token for git credential helper
func (a *Authenticator) GetCredentials(app *config.GitHubApp, repoURL string) (token, username string, err error) {
	// Generate cache key
	cacheKey := cache.CreateCacheKey(app.AppID, app.InstallationID)
	
	// Check cache first
	if cachedToken, found := a.tokenCache.Get(cacheKey); found {
		username = fmt.Sprintf("%s[bot]", app.Name)
		return cachedToken, username, nil
	}

	// Generate JWT token
	jwtToken, err := a.GenerateJWT(app.AppID, app.PrivateKeyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Get installation token
	installationToken, err := a.GetInstallationToken(jwtToken, app.InstallationID, repoURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to get installation token: %w", err)
	}

	// Cache the token (GitHub installation tokens are valid for 1 hour)
	a.tokenCache.Set(cacheKey, installationToken, 55*time.Minute)

	// Return credentials
	username = fmt.Sprintf("%s[bot]", app.Name)
	return installationToken, username, nil
}

// GenerateJWT generates a JWT token for the GitHub App
func (a *Authenticator) GenerateJWT(appID int64, privateKeyPath string) (string, error) {
	return a.jwtGenerator.GenerateToken(appID, privateKeyPath)
}

// GetInstallationToken exchanges JWT for an installation access token
func (a *Authenticator) GetInstallationToken(jwtToken string, installationID int64, repoURL string) (string, error) {
	// Create API client with JWT token
	client, err := api.NewRESTClient(api.ClientOptions{
		Headers: map[string]string{
			"Authorization": "Bearer " + jwtToken,
			"Accept":        "application/vnd.github.v3+json",
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create API client: %w", err)
	}

	// If installation ID is not provided, try to find it
	if installationID == 0 {
		installationID, err = a.findInstallationID(client, repoURL)
		if err != nil {
			return "", fmt.Errorf("failed to find installation ID: %w", err)
		}
	}

	// Request installation access token
	var tokenResponse struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	endpoint := fmt.Sprintf("app/installations/%d/access_tokens", installationID)
	if err := client.Post(endpoint, nil, &tokenResponse); err != nil {
		return "", fmt.Errorf("failed to get installation token: %w", err)
	}

	return tokenResponse.Token, nil
}

// findInstallationID finds the installation ID for a repository
func (a *Authenticator) findInstallationID(client *api.RESTClient, repoURL string) (int64, error) {
	// Extract owner and repo from URL
	owner, repo, err := parseRepoURL(repoURL)
	if err != nil {
		return 0, fmt.Errorf("failed to parse repository URL: %w", err)
	}

	// Get installation for the repository
	var installation struct {
		ID int64 `json:"id"`
	}

	endpoint := fmt.Sprintf("repos/%s/%s/installation", owner, repo)
	if err := client.Get(endpoint, &installation); err != nil {
		return 0, fmt.Errorf("failed to get installation for repository: %w", err)
	}

	return installation.ID, nil
}

// parseRepoURL parses a repository URL to extract owner and repo
func parseRepoURL(repoURL string) (owner, repo string, err error) {
	// Remove protocol and .git suffix
	repoURL = strings.TrimSuffix(repoURL, ".git")
	repoURL = strings.TrimPrefix(repoURL, "https://")
	repoURL = strings.TrimPrefix(repoURL, "http://")
	repoURL = strings.TrimPrefix(repoURL, "git@")

	// Handle SSH format: git@github.com:owner/repo
	if strings.Contains(repoURL, ":") {
		parts := strings.Split(repoURL, ":")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH repository URL format")
		}
		repoURL = parts[1]
	} else {
		// Handle HTTPS format: github.com/owner/repo
		parts := strings.Split(repoURL, "/")
		if len(parts) < 3 {
			return "", "", fmt.Errorf("invalid HTTPS repository URL format")
		}
		repoURL = strings.Join(parts[1:], "/")
	}

	// Extract owner and repo
	parts := strings.Split(repoURL, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository URL format")
	}

	return parts[0], parts[1], nil
}

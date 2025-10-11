package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/cache"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/jwt"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/secrets"
	"github.com/cli/go-gh/v2/pkg/api"
)

const (
	gitHubAPIHost = "github.com"
)

// Authenticator handles GitHub App authentication.
type Authenticator struct {
	jwtGenerator   *jwt.Generator
	tokenCache     *cache.TokenCache
	secretsManager *secrets.Manager
	// clientFactory creates API clients (can be overridden for testing)
	clientFactory func(api.ClientOptions) (*api.RESTClient, error)
}

// NewAuthenticator creates a new authenticator.
func NewAuthenticator() *Authenticator {
	// Initialize secrets manager
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth")

	return &Authenticator{
		jwtGenerator:   jwt.NewGenerator(),
		tokenCache:     cache.NewTokenCache(),
		secretsManager: secrets.NewManager(configDir),
		clientFactory:  api.NewRESTClient,
	}
}

// GetCredentials returns username and token for git credential helper.
func (a *Authenticator) GetCredentials(app *config.GitHubApp, repoURL string) (token, username string, err error) {
	// Generate cache key
	cacheKey := cache.CreateCacheKey(app.AppID, app.InstallationID)

	// Check cache first
	if cachedToken, found := a.tokenCache.Get(cacheKey); found {
		username = fmt.Sprintf("%s[bot]", app.Name)
		return cachedToken, username, nil
	}

	// Get private key from secure storage
	privateKey, err := app.GetPrivateKey(a.secretsManager)
	if err != nil {
		return "", "", fmt.Errorf("failed to get private key: %w", err)
	}

	// Generate JWT token
	jwtToken, err := a.jwtGenerator.GenerateTokenFromKey(app.AppID, privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Get installation token from GitHub API
	installationToken, err := a.GetInstallationToken(jwtToken, app.InstallationID, repoURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to get installation token: %w", err)
	}

	// Cache the token for 55 minutes (GitHub tokens valid for 60 minutes, 5-min buffer)
	// SECURITY: Token stored in memory only, not persisted to disk. See docs/TOKEN_CACHING.md
	a.tokenCache.Set(cacheKey, installationToken, 55*time.Minute)

	// Return credentials
	username = fmt.Sprintf("%s[bot]", app.Name)
	return installationToken, username, nil
}

// GenerateJWT generates a JWT token for the GitHub App (legacy file-based method).
func (a *Authenticator) GenerateJWT(appID int64, privateKeyPath string) (string, error) {
	return a.jwtGenerator.GenerateToken(appID, privateKeyPath)
}

// GenerateJWTForApp generates a JWT token using the app's configured private key source.
func (a *Authenticator) GenerateJWTForApp(app *config.GitHubApp) (string, error) {
	// Get private key from secure storage
	privateKey, err := app.GetPrivateKey(a.secretsManager)
	if err != nil {
		return "", fmt.Errorf("failed to get private key: %w", err)
	}

	// Generate JWT token
	return a.jwtGenerator.GenerateTokenFromKey(app.AppID, privateKey)
}

// GetInstallationToken exchanges JWT for an installation access token.
func (a *Authenticator) GetInstallationToken(jwtToken string, installationID int64, repoURL string) (string, error) {
	// Extract host from repository URL (default to github.com)
	host := extractHostFromURL(repoURL)

	// If installation ID is not provided, try to find it
	if installationID == 0 {
		var err error
		installationID, err = a.findInstallationIDHTTP(jwtToken, host, repoURL)
		if err != nil {
			return "", fmt.Errorf("failed to find installation ID: %w", err)
		}
	}

	// Request installation access token using raw HTTP
	apiURL := fmt.Sprintf("https://%s/api/v3/app/installations/%d/access_tokens", host, installationID)
	if host == gitHubAPIHost {
		apiURL = fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get installation token: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResponse struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return tokenResponse.Token, nil
}

// findInstallationIDHTTP finds the installation ID for a repository using raw HTTP.
func (a *Authenticator) findInstallationIDHTTP(jwtToken, host, repoURL string) (int64, error) {
	// Extract owner and repo from URL
	owner, repo, err := parseRepoURL(repoURL)
	if err != nil {
		return 0, fmt.Errorf("failed to parse repository URL: %w", err)
	}

	// Construct API URL
	apiURL := fmt.Sprintf("https://%s/api/v3/repos/%s/%s/installation", host, owner, repo)
	if host == gitHubAPIHost {
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/installation", owner, repo)
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
		return 0, fmt.Errorf("failed to get installation: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var installation struct {
		ID int64 `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&installation); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return installation.ID, nil
}

// extractHostFromURL extracts the host from a repository URL.
func extractHostFromURL(repoURL string) string {
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

// parseRepoURL parses a repository URL to extract owner and repo.
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

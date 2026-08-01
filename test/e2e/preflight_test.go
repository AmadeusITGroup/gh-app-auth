//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/jwt"
)

// Expected test repositories — must be pre-created per docs/E2E_INFRASTRUCTURE.md.
const (
	mainRepo      = "main-repo"      // In TestOrg1: private, contains submodule from TestOrg2
	submoduleRepo = "submodule-repo" // In TestOrg2: private, referenced as submodule

	// Marker files must exist in cloned repos for content-verification assertions.
	mainMarkerFile         = "data/main-marker.txt"
	submoduleMarkerFile    = "data/submodule-marker.txt"
	mainMarkerContent      = "gh-app-auth-e2e-main-marker"
	submoduleMarkerContent = "gh-app-auth-e2e-submodule-marker"
)

// TestPreflight validates that the E2E infrastructure is correctly set up.
// This is a hard pre-flight gate: all sub-tests must pass before any workflow
// tests can be meaningful. Failures here indicate infrastructure issues, not
// product bugs.
func TestPreflight(t *testing.T) {
	t.Run("binary_accessible", testPreflightBinary)
	t.Run("app_token_valid", testPreflightAppToken)
	t.Run("org1_main_repo_is_private", testPreflightRepoPrivate(globalConfig.TestOrg1, mainRepo))
	t.Run("org2_submodule_repo_is_private", testPreflightRepoPrivate(globalConfig.TestOrg2, submoduleRepo))
}

func testPreflightBinary(t *testing.T) {
	env, _ := isolatedAppConfig(t)
	stdout, stderr, err := RunCmd(t, env, "--version")
	if err != nil {
		t.Fatalf("binary not accessible or returned error: %v\nstderr: %s", err, stderr)
	}
	combined := stdout + stderr
	if combined == "" {
		t.Error("expected version output, got nothing")
	}
	t.Logf("binary version: %s", combined)
}

func testPreflightAppToken(t *testing.T) {
	// Test token generation for both orgs to verify installation discovery works
	for _, org := range []string{globalConfig.TestOrg1, globalConfig.TestOrg2} {
		token, err := generateAppInstallationTokenForOrg(org)
		if err != nil {
			t.Fatalf("failed to generate App installation token for %s: %v", org, err)
		}

		// Validate installation token by accessing a repo it should have access to
		// (installation tokens cannot access /user endpoint - that's for PATs only)
		repo := mainRepo
		if org == globalConfig.TestOrg2 {
			repo = submoduleRepo
		}
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s", org, repo), nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to call GitHub API: %v", err)
		}
		resp.Body.Close() //nolint:errcheck

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("App installation token for %s is invalid (HTTP %d): %s", org, resp.StatusCode, body)
		}
		t.Logf("App installation token for %s is valid", org)
	}
}

// testPreflightRepoPrivate returns a test function that verifies a repo exists
// and is private. Fails fast with a clear, actionable error if not.
func testPreflightRepoPrivate(org, repo string) func(*testing.T) {
	return func(t *testing.T) {
		token, err := generateAppInstallationTokenForOrg(org)
		if err != nil {
			t.Fatalf("failed to generate App installation token for %s: %v", org, err)
		}

		info, err := fetchRepoInfo(token, org, repo)
		if err != nil {
			t.Fatalf(
				"repo %s/%s not accessible: %v\n\nAction required: create the repository per docs/E2E_INFRASTRUCTURE.md",
				org, repo, err,
			)
		}
		if !info.Private {
			t.Fatalf(
				"SECURITY GATE FAILED: %s/%s is PUBLIC\n"+
					"E2E tests authenticate against private repos to validate real auth flows.\n"+
					"Set the repository to private before running E2E tests.",
				org, repo,
			)
		}
		t.Logf("✓ %s/%s exists and is private", org, repo)
	}
}

// repoMeta holds the relevant fields from the GitHub repos API response.
type repoMeta struct {
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
}

// fetchRepoInfo calls the GitHub repos API and returns repository metadata.
func fetchRepoInfo(token, org, repo string) (*repoMeta, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", org, repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("repository %s/%s not found (404)", org, repo)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API HTTP %d: %s", resp.StatusCode, body)
	}

	var meta repoMeta
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &meta, nil
}

// generateAppInstallationTokenForOrg generates a GitHub App installation access token
// for the specified organization. It discovers the installation ID dynamically
// using the GitHub API if not already cached.
func generateAppInstallationTokenForOrg(org string) (string, error) {
	// Parse App ID
	appID, err := strconv.ParseInt(globalConfig.AppID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid E2E_APP_ID: %w", err)
	}

	// Check cache first
	var installationID int64
	if cachedID, ok := globalConfig.installationIDCache[org]; ok {
		installationID = cachedID
	} else {
		// Discover installation ID dynamically
		discoveredID, err := discoverInstallationID(org)
		if err != nil {
			return "", fmt.Errorf("failed to discover installation ID for %s: %w", org, err)
		}
		installationID = discoveredID
		globalConfig.installationIDCache[org] = installationID
	}

	// Generate JWT for the App
	jwtGen := jwt.NewGenerator()
	jwtToken, err := jwtGen.GenerateTokenFromKey(appID, globalConfig.PrivateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Exchange JWT for installation token
	return getInstallationToken(jwtToken, installationID)
}

// discoverInstallationID discovers the installation ID for an organization
// by listing installations accessible to the GitHub App.
func discoverInstallationID(org string) (int64, error) {
	// Parse App ID
	appID, err := strconv.ParseInt(globalConfig.AppID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid E2E_APP_ID: %w", err)
	}

	// Generate JWT for the App
	jwtGen := jwt.NewGenerator()
	jwtToken, err := jwtGen.GenerateTokenFromKey(appID, globalConfig.PrivateKeyPEM)
	if err != nil {
		return 0, fmt.Errorf("failed to generate JWT: %w", err)
	}

	// List installations
	apiURL := "https://api.github.com/app/installations"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to list installations: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var installations []struct {
		ID      int64 `json:"id"`
		Account struct {
			Login string `json:"login"`
		} `json:"account"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&installations); err != nil {
		return 0, fmt.Errorf("failed to decode installations: %w", err)
	}

	// Find installation for the organization
	for _, inst := range installations {
		if inst.Account.Login == org {
			return inst.ID, nil
		}
	}

	return 0, fmt.Errorf("no installation found for organization %s", org)
}

// getInstallationToken exchanges a JWT for an installation access token.
func getInstallationToken(jwtToken string, installationID int64) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)

	req, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get installation token: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResponse struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return tokenResponse.Token, nil
}

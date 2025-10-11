package auth

import (
	"fmt"
	"testing"
	"time"
)

func TestNewAuthenticator(t *testing.T) {
	auth := NewAuthenticator()

	if auth == nil {
		t.Fatal("Expected authenticator to be created")
	}

	if auth.jwtGenerator == nil {
		t.Error("Expected JWT generator to be initialized")
	}

	if auth.tokenCache == nil {
		t.Error("Expected token cache to be initialized")
	}

	if auth.secretsManager == nil {
		t.Error("Expected secrets manager to be initialized")
	}
}

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name      string
		repoURL   string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "HTTPS URL",
			repoURL:   "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "HTTPS URL with .git",
			repoURL:   "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "SSH URL",
			repoURL:   "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "SSH URL without .git",
			repoURL:   "git@github.com:owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "HTTP URL",
			repoURL:   "http://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:    "Invalid URL - no owner/repo",
			repoURL: "https://github.com",
			wantErr: true,
		},
		{
			name:    "Invalid URL - only owner",
			repoURL: "https://github.com/owner",
			wantErr: true,
		},
		{
			name:    "Invalid SSH format",
			repoURL: "git@github.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseRepoURL(tt.repoURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for URL %s, got none", tt.repoURL)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("Owner = %s, want %s", owner, tt.wantOwner)
			}

			if repo != tt.wantRepo {
				t.Errorf("Repo = %s, want %s", repo, tt.wantRepo)
			}
		})
	}
}

func TestGenerateJWT(t *testing.T) {
	t.Skip("Requires properly formatted RSA key - tracked in TESTING_IMPROVEMENTS_TODO.md")
	// testutil.SkipIfShort(t, "requires file I/O")

	// // Create test private key
	// keyPath := testutil.CreateTestPrivateKey(t)

	// auth := NewAuthenticator()

	// // Test with invalid app ID
	// t.Run("invalid app ID", func(t *testing.T) {
	// 	_, err := auth.GenerateJWT(0, keyPath)
	// 	if err == nil {
	// 		t.Error("Expected error for invalid app ID")
	// 	}
	// })

	// // Test with non-existent key file
	// t.Run("non-existent key file", func(t *testing.T) {
	// 	_, err := auth.GenerateJWT(123456, "/nonexistent/key.pem")
	// 	if err == nil {
	// 		t.Error("Expected error for non-existent key file")
	// 	}
	// })

	// // Test with valid inputs
	// t.Run("valid inputs", func(t *testing.T) {
	// 	token, err := auth.GenerateJWT(123456, keyPath)
	// 	if err != nil {
	// 		t.Fatalf("Unexpected error: %v", err)
	// 	}

	// 	if token == "" {
	// 		t.Error("Expected token to be generated")
	// 	}

	// 	// JWT should have 3 parts separated by dots
	// 	parts := strings.Split(token, ".")
	// 	if len(parts) != 3 {
	// 		t.Errorf("Expected JWT to have 3 parts, got %d", len(parts))
	// 	}
	// })
}

func TestGenerateJWTForApp(t *testing.T) {
	t.Skip("Requires properly formatted RSA key - tracked in TESTING_IMPROVEMENTS_TODO.md")
}

// TestTokenCaching validates that tokens are properly cached
func TestTokenCaching(t *testing.T) {
	auth := NewAuthenticator()

	// Manually test cache
	cacheKey := "test-app-123-install-456"
	testToken := "ghs_test_token"

	// Set in cache
	auth.tokenCache.Set(cacheKey, testToken, 1*time.Minute)

	// Retrieve from cache
	cached, found := auth.tokenCache.Get(cacheKey)
	if !found {
		t.Error("Expected token to be in cache")
	}

	if cached != testToken {
		t.Errorf("Cached token = %s, want %s", cached, testToken)
	}

	// Wait for expiration
	auth.tokenCache.Set(cacheKey, testToken, 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	_, found = auth.tokenCache.Get(cacheKey)
	if found {
		t.Error("Expected token to be expired from cache")
	}
}

// TestAuthenticatorConcurrency tests concurrent access to the authenticator
func TestAuthenticatorConcurrency(t *testing.T) {
	t.Skip("Requires properly formatted RSA key - tracked in TESTING_IMPROVEMENTS_TODO.md")
}

// Benchmark tests
func BenchmarkParseRepoURL(b *testing.B) {
	testURL := "https://github.com/owner/repo.git"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = parseRepoURL(testURL)
	}
}

// Example showing the authentication flow
func ExampleAuthenticator_GetCredentials() {
	// This example shows the expected flow for GetCredentials
	// Real implementation requires GitHub API mock server

	fmt.Println("Authentication flow:")
	fmt.Println("1. Check cache for existing token")
	fmt.Println("2. If not cached, generate JWT")
	fmt.Println("3. Exchange JWT for installation token")
	fmt.Println("4. Cache the installation token")
	fmt.Println("5. Return token and username")

	// Output:
	// Authentication flow:
	// 1. Check cache for existing token
	// 2. If not cached, generate JWT
	// 3. Exchange JWT for installation token
	// 4. Cache the installation token
	// 5. Return token and username
}

// NOTE: Full integration tests for GetCredentials and GetInstallationToken
// require a mock GitHub API server. See test/testutil/mock_github.go
// Implementation tracked in TESTING_IMPROVEMENTS_TODO.md Phase 1

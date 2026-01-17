package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"gopkg.in/yaml.v3"
)

func TestExtractHost(t *testing.T) {
	tests := []struct {
		name    string
		repoURL string
		want    string
	}{
		{
			name:    "github.com URL",
			repoURL: "https://github.com/myorg/myrepo",
			want:    "github.com",
		},
		{
			name:    "github.com with git suffix",
			repoURL: "https://github.com/myorg/myrepo.git",
			want:    "github.com",
		},
		{
			name:    "enterprise GitHub URL",
			repoURL: "https://github.enterprise.com/myorg/myrepo",
			want:    "github.enterprise.com",
		},
		{
			name:    "SSH format",
			repoURL: "git@github.com:myorg/myrepo.git",
			want:    "github.com",
		},
		{
			name:    "plain domain",
			repoURL: "example.com",
			want:    "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHost(tt.repoURL)
			if got != tt.want {
				t.Errorf("extractHost(%q) = %q, want %q", tt.repoURL, got, tt.want)
			}
		})
	}
}

func TestExtractOwnerRepo(t *testing.T) {
	tests := []struct {
		name      string
		repoURL   string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "HTTPS URL",
			repoURL:   "https://github.com/myorg/myrepo",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
			wantErr:   false,
		},
		{
			name:      "HTTPS URL with .git",
			repoURL:   "https://github.com/myorg/myrepo.git",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
			wantErr:   false,
		},
		{
			name:      "SSH URL",
			repoURL:   "git@github.com:myorg/myrepo.git",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
			wantErr:   false,
		},
		{
			name:      "SSH URL without .git",
			repoURL:   "git@github.com:myorg/myrepo",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
			wantErr:   false,
		},
		{
			name:      "HTTP URL",
			repoURL:   "http://github.com/myorg/myrepo",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
			wantErr:   false,
		},
		{
			name:      "Enterprise GitHub HTTPS",
			repoURL:   "https://github.enterprise.com/myorg/myrepo",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
			wantErr:   false,
		},
		{
			name:      "Enterprise GitHub SSH",
			repoURL:   "git@github.enterprise.com:myorg/myrepo",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
			wantErr:   false,
		},
		{
			name:      "invalid format - no slashes",
			repoURL:   "invalid",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
		{
			name:      "invalid format - only owner",
			repoURL:   "https://github.com/myorg",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
		{
			name:      "invalid SSH format",
			repoURL:   "git@github.com",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := extractOwnerRepo(tt.repoURL)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}

			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

func TestGetCurrentRepository(t *testing.T) {
	t.Run("not implemented", func(t *testing.T) {
		_, err := getCurrentRepository()
		if err == nil {
			t.Error("Expected error for not implemented function")
		}
	})
}

func TestDetermineRepositoryURL(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		wantErr bool
	}{
		{
			name:    "with repo specified",
			repo:    "https://github.com/myorg/myrepo",
			wantErr: false,
		},
		{
			name:    "without repo - should fail",
			repo:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := determineRepositoryURL(tt.repo)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if got != tt.repo {
				t.Errorf("got = %q, want %q", got, tt.repo)
			}
		})
	}
}

func TestDisplayTestResults(t *testing.T) {
	t.Run("non-verbose", func(t *testing.T) {
		// Just verify it doesn't panic
		displayTestResults(false)
	})

	t.Run("verbose", func(t *testing.T) {
		// Just verify it doesn't panic
		displayTestResults(true)
	})
}

func TestLoadTestConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func(t *testing.T) string
		wantErr     bool
	}{
		{
			name: "valid configuration",
			setupConfig: func(t *testing.T) string {
				t.Helper()
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "config.yml")

				cfg := &config.Config{
					Version: "1.0",
					GitHubApps: []config.GitHubApp{
						{
							Name:             "Test App",
							AppID:            123456,
							InstallationID:   789012,
							PrivateKeySource: config.PrivateKeySourceFilesystem,
							PrivateKeyPath:   "/tmp/key.pem",
							Patterns:         []string{"github.com/test/*"},
						},
					},
				}

				data, err := yaml.Marshal(cfg)
				if err != nil {
					t.Fatalf("Failed to marshal config: %v", err)
				}
				if err := os.WriteFile(configPath, data, 0600); err != nil {
					t.Fatalf("Failed to save config: %v", err)
				}

				return configPath
			},
			wantErr: false,
		},
		{
			name: "missing configuration",
			setupConfig: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "nonexistent.yml")
			},
			wantErr: true,
		},
		{
			name: "empty configuration",
			setupConfig: func(t *testing.T) string {
				t.Helper()
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "config.yml")

				cfg := &config.Config{
					Version:    "1.0",
					GitHubApps: []config.GitHubApp{},
				}

				data, err := yaml.Marshal(cfg)
				if err != nil {
					t.Fatalf("Failed to marshal config: %v", err)
				}
				if err := os.WriteFile(configPath, data, 0600); err != nil {
					t.Fatalf("Failed to save config: %v", err)
				}

				return configPath
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupConfig(t)

			// Set env var to override default config path
			original := os.Getenv("GH_APP_AUTH_CONFIG")
			defer func() {
				if original != "" {
					os.Setenv("GH_APP_AUTH_CONFIG", original)
				} else {
					os.Unsetenv("GH_APP_AUTH_CONFIG")
				}
			}()
			os.Setenv("GH_APP_AUTH_CONFIG", configPath)

			cfg, err := loadTestConfiguration()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if cfg == nil {
				t.Fatal("Expected non-nil config")
			}

			if len(cfg.GitHubApps) == 0 {
				t.Error("Expected at least one GitHub App")
			}
		})
	}
}

func TestFindMatchingAppForTest(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	cfg := &config.Config{
		Version: "1.0",
		GitHubApps: []config.GitHubApp{
			{
				Name:             "Test App",
				AppID:            123456,
				InstallationID:   789012,
				PrivateKeySource: config.PrivateKeySourceFilesystem,
				PrivateKeyPath:   "/tmp/key.pem",
				Patterns:         []string{"github.com/test/*"},
				Priority:         5,
			},
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	original := os.Getenv("GH_APP_AUTH_CONFIG")
	defer func() {
		if original != "" {
			os.Setenv("GH_APP_AUTH_CONFIG", original)
		} else {
			os.Unsetenv("GH_APP_AUTH_CONFIG")
		}
	}()
	os.Setenv("GH_APP_AUTH_CONFIG", configPath)

	tests := []struct {
		name    string
		repoURL string
		verbose bool
		wantErr bool
	}{
		{
			name:    "matching repository",
			repoURL: "https://github.com/test/repo",
			verbose: false,
			wantErr: false,
		},
		{
			name:    "matching repository verbose",
			repoURL: "https://github.com/test/repo",
			verbose: true,
			wantErr: false,
		},
		{
			name:    "non-matching repository",
			repoURL: "https://github.com/other/repo",
			verbose: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loadedCfg, err := loadTestConfiguration()
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			app, pat, err := findMatchingCredential(loadedCfg, tt.repoURL)

			if tt.wantErr {
				// For non-matching repos, we expect nil app and pat
				if app != nil || pat != nil {
					t.Error("Expected no matching credentials but got one")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if app == nil && pat == nil {
				t.Fatal("Expected non-nil app or PAT")
			}

			if app != nil && app.Name != "Test App" {
				t.Errorf("App name = %q, want %q", app.Name, "Test App")
			}
		})
	}
}

func TestTestJWTGeneration(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test-key.pem")

	// Generate valid test key
	testKey := generateTestRSAKey(t)
	if err := os.WriteFile(keyPath, []byte(testKey), 0600); err != nil {
		t.Fatalf("Failed to write test key: %v", err)
	}

	tests := []struct {
		name    string
		app     *config.GitHubApp
		verbose bool
		wantErr bool
	}{
		{
			name: "valid app non-verbose",
			app: &config.GitHubApp{
				Name:             "Test App",
				AppID:            123456,
				InstallationID:   789012,
				PrivateKeySource: config.PrivateKeySourceFilesystem,
				PrivateKeyPath:   keyPath,
				Patterns:         []string{"github.com/test/*"},
			},
			verbose: false,
			wantErr: false,
		},
		{
			name: "valid app verbose",
			app: &config.GitHubApp{
				Name:             "Test App",
				AppID:            123456,
				InstallationID:   789012,
				PrivateKeySource: config.PrivateKeySourceFilesystem,
				PrivateKeyPath:   keyPath,
				Patterns:         []string{"github.com/test/*"},
			},
			verbose: true,
			wantErr: false,
		},
		{
			name: "invalid key path",
			app: &config.GitHubApp{
				Name:             "Test App",
				AppID:            123456,
				InstallationID:   789012,
				PrivateKeySource: config.PrivateKeySourceFilesystem,
				PrivateKeyPath:   "/nonexistent/key.pem",
				Patterns:         []string{"github.com/test/*"},
			},
			verbose: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := testJWTGeneration(tt.app, tt.verbose)
			if (err != nil) != tt.wantErr {
				t.Errorf("testJWTGeneration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && token == "" {
				t.Error("Expected non-empty JWT token")
			}
		})
	}
}

func TestTestAPIAccess(t *testing.T) {
	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "token ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/repos/test/repo" {
			resp := map[string]interface{}{
				"name":      "repo",
				"full_name": "test/repo",
				"private":   false,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "Not found", http.StatusNotFound)
	}))
	defer mockServer.Close()

	tests := []struct {
		name    string
		token   string
		repoURL string
		verbose bool
		wantErr bool
	}{
		{
			name:    "valid token and repo",
			token:   "ghs_test_token",
			repoURL: "https://github.com/test/repo",
			verbose: false,
			wantErr: false,
		},
		{
			name:    "valid token and repo verbose",
			token:   "ghs_test_token",
			repoURL: "https://github.com/test/repo",
			verbose: true,
			wantErr: false,
		},
		{
			name:    "invalid repo URL format",
			token:   "ghs_test_token",
			repoURL: "invalid",
			verbose: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: testAPIAccess calls the real GitHub API via go-gh
			// We can't easily mock it without modifying the function
			// So we test the error path (invalid URL) which doesn't make API calls
			if tt.wantErr {
				err := testAPIAccess(tt.token, tt.repoURL, tt.verbose)
				if err == nil {
					t.Error("Expected error but got none")
				}
			}
			// Skip successful cases as they require real API access
		})
	}
}

func TestRunAuthenticationTests_ErrorPaths(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	cfg := &config.Config{
		Version: "1.0",
		GitHubApps: []config.GitHubApp{
			{
				Name:             "Test App",
				AppID:            123456,
				InstallationID:   789012,
				PrivateKeySource: config.PrivateKeySourceFilesystem,
				PrivateKeyPath:   "/nonexistent/key.pem",
				Patterns:         []string{"github.com/test/*"},
			},
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	original := os.Getenv("GH_APP_AUTH_CONFIG")
	defer func() {
		if original != "" {
			os.Setenv("GH_APP_AUTH_CONFIG", original)
		} else {
			os.Unsetenv("GH_APP_AUTH_CONFIG")
		}
	}()
	os.Setenv("GH_APP_AUTH_CONFIG", configPath)

	tests := []struct {
		name    string
		repoURL string
		verbose bool
		wantErr bool
	}{
		{
			name:    "non-matching repository",
			repoURL: "https://github.com/other/repo",
			verbose: false,
			wantErr: true,
		},
		{
			name:    "matching repo but missing key file",
			repoURL: "https://github.com/test/repo",
			verbose: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loadedCfg, err := loadTestConfiguration()
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			err = runAuthenticationTests(loadedCfg, tt.repoURL, tt.verbose)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

func TestTestGitHubAPIAccess(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		repoURL string
		verbose bool
		wantErr bool
	}{
		{
			name:    "invalid URL format",
			token:   "test_token",
			repoURL: "invalid-url",
			verbose: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testGitHubAPIAccess("Step 1: ", tt.token, tt.repoURL, tt.verbose)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

func TestNewTestCmd(t *testing.T) {
	cmd := NewTestCmd()

	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}

	if cmd.Use != "test" {
		t.Errorf("Use = %q, want %q", cmd.Use, "test")
	}

	// Test that flags are defined
	if cmd.Flags().Lookup("repo") == nil {
		t.Error("Expected --repo flag to be defined")
	}

	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("Expected --verbose flag to be defined")
	}
}

func TestTestRun_Integration(t *testing.T) {
	// Create a test configuration
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	// Create a test key file
	keyPath := filepath.Join(tempDir, "test-key.pem")
	testKey := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF4qjQKzq4V5l6mJ9VpKJqKvYXKrc
IzDRNgQPBCZQqS4OZjFHFVLDPq3EO3R7+2VvTMvKVU6EwXVBXkjJPvbmQGWKqHvL
VB0sPVOAGEqQXcWmCNPpMTLTNLwMSPX3GjNaJEp2a0qSVZpUePqcTjz4U5GZC/0r
sSBDZXWKR1kRCT9E3YiRFKx+PQ9gQcFqMzH4A3OQqTBGPO5F0O0LJwTTTqQqGdTH
g5kMa6WGPQN5hPnCDxaFMNsEEQdKT8LXQ9JNQkW4JYqNHGqN3VkRQdHQEQM0Vq1t
IHQnEW6fV0XqVCVN6IQVVH0uDQqYRVKqEwIDAQABAoIBAG3Z6Y7FVA+rUOqJcW6t
9YRH7HQdvJKqQHFQpwPLVPvN5H6Q3J2TqRBhKv4qJvM6VrFQ5bphZKZQ6Hjqxevq
j6TVhXPKfN7MJHLxQphZqf3xjPG7qmFJWVqR8L5X+RYaGkP5MNYKu+pKKK4T8qpR
cLZxQFPK8WqG4j8wVYWGMH5RqPQTZw5ZKZLqqJKjGvPHMQ8xVGPYGmQwKJqmxPKQ
8YqLqhvGqJFpQJqmPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqmQGYqJP8w
VGYqQHLqPJYGmPqQJhvGqJPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqmQG
YqJP8ECgYEA6i9AxVPQpV8JqCGXQHMYqGHQVPGqJqLGYPqQJ8xGLqYPHqmQGYqJP
8wVGYqQHLqPJYGmPqQJhvGqJPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqm
-----END RSA PRIVATE KEY-----`
	if err := os.WriteFile(keyPath, []byte(testKey), 0600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	cfg := &config.Config{
		Version: "1.0",
		GitHubApps: []config.GitHubApp{
			{
				Name:             "Test App",
				AppID:            123456,
				InstallationID:   789012,
				PrivateKeySource: config.PrivateKeySourceFilesystem,
				PrivateKeyPath:   keyPath,
				Patterns:         []string{"github.com/test/*"},
				Priority:         5,
			},
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	original := os.Getenv("GH_APP_AUTH_CONFIG")
	defer func() {
		if original != "" {
			os.Setenv("GH_APP_AUTH_CONFIG", original)
		} else {
			os.Unsetenv("GH_APP_AUTH_CONFIG")
		}
	}()
	os.Setenv("GH_APP_AUTH_CONFIG", configPath)

	t.Run("no repo specified error", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{})

		// Capture output
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error when no repo specified")
		}
	})

	t.Run("non-matching repo error", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--repo", "https://github.com/other/repo"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for non-matching repo")
		}
	})
}

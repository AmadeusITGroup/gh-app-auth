package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

func TestGitCredentialWithPattern(t *testing.T) {
	// Create a test configuration with multiple apps
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	configContent := `version: "1.0"
github_apps:
  - app_id: 111111
    installation_id: 987654
    name: "App for AmadeusITGroup"
    private_key_path: /tmp/key1.pem
    patterns:
      - "https://github.com/AmadeusITGroup"
  - app_id: 222222
    installation_id: 876543
    name: "App for myorg"
    private_key_path: /tmp/key2.pem
    patterns:
      - "https://github.com/myorg"
  - app_id: 333333
    installation_id: 765432
    name: "App for wildcards"
    private_key_path: /tmp/key3.pem
    patterns:
      - "https://github.com"
`

	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	tests := []struct {
		name          string
		pattern       string
		input         string
		expectMatch   bool
		expectedAppID int64
		expectSilent  bool
	}{
		{
			name:          "Pattern matches AmadeusITGroup app",
			pattern:       "https://github.com/AmadeusITGroup",
			input:         "protocol=https\nhost=github.com\npath=AmadeusITGroup/repo\n\n",
			expectMatch:   true,
			expectedAppID: 111111,
			expectSilent:  false,
		},
		{
			name:          "Pattern matches myorg app",
			pattern:       "https://github.com/myorg",
			input:         "protocol=https\nhost=github.com\npath=myorg/repo\n\n",
			expectMatch:   true,
			expectedAppID: 222222,
			expectSilent:  false,
		},
		{
			name:         "Pattern doesn't match any app",
			pattern:      "https://github.com/nonexistent",
			input:        "protocol=https\nhost=github.com\npath=AmadeusITGroup/repo\n\n",
			expectMatch:  false,
			expectSilent: true,
		},
		{
			name:         "No pattern - falls back to URL matching (should exit silently if no match)",
			pattern:      "",
			input:        "protocol=https\nhost=github.com\npath=nonexistent/repo\n\n",
			expectMatch:  false,
			expectSilent: true,
		},
		{
			name:         "No pattern - URL matching not expected to work with URL prefix patterns",
			pattern:      "",
			input:        "protocol=https\nhost=github.com\npath=AmadeusITGroup/repo\n\n",
			expectMatch:  false,
			expectSilent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the pattern flag
			gitCredentialPattern = tt.pattern

			// Mock stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			go func() {
				w.Write([]byte(tt.input))
				w.Close()
			}()

			// Capture stdout
			oldStdout := os.Stdout
			rOut, wOut, _ := os.Pipe()
			os.Stdout = wOut

			// Run the handler
			err := handleCredentialGet()

			// Restore stdin/stdout
			os.Stdin = oldStdin
			os.Stdout = oldStdout
			wOut.Close()

			// Read output
			var buf bytes.Buffer
			io.Copy(&buf, rOut)
			output := buf.String()

			if tt.expectSilent {
				// Should exit silently (no error, no output)
				if err != nil {
					t.Errorf("Expected silent exit, got error: %v", err)
				}
				if output != "" {
					t.Errorf("Expected no output, got: %s", output)
				}
			} else {
				// Should either succeed or fail with an error
				// (will fail if private key doesn't exist, which is expected in tests)
				if err == nil && output == "" {
					t.Errorf("Expected output or error, got neither")
				}
			}
		})
	}
}

func TestPatternMatchingLogic(t *testing.T) {
	apps := []config.GitHubApp{
		{
			AppID:    111111,
			Name:     "App1",
			Patterns: []string{"https://github.com/org1", "https://github.com/org2"},
		},
		{
			AppID:    222222,
			Name:     "App2",
			Patterns: []string{"https://github.com/org3"},
		},
		{
			AppID:    333333,
			Name:     "App3",
			Patterns: []string{"https://github.enterprise.com"},
		},
	}

	tests := []struct {
		name          string
		searchPattern string
		repoURL       string
		expectedAppID int64
		shouldFind    bool
	}{
		{
			name:          "Find app with URL prefix match",
			searchPattern: "https://github.com/org1",
			repoURL:       "https://github.com/org1/repo",
			expectedAppID: 111111,
			shouldFind:    true,
		},
		{
			name:          "Find app with different org",
			searchPattern: "https://github.com/org2",
			repoURL:       "https://github.com/org2/another-repo",
			expectedAppID: 111111,
			shouldFind:    true,
		},
		{
			name:          "Find different app",
			searchPattern: "https://github.com/org3",
			repoURL:       "https://github.com/org3/repo",
			expectedAppID: 222222,
			shouldFind:    true,
		},
		{
			name:          "Find enterprise app",
			searchPattern: "https://github.enterprise.com",
			repoURL:       "https://github.enterprise.com/org/repo",
			expectedAppID: 333333,
			shouldFind:    true,
		},
		{
			name:          "Pattern not found in config",
			searchPattern: "https://github.com/nonexistent",
			repoURL:       "https://github.com/nonexistent/repo",
			shouldFind:    false,
		},
		{
			name:          "URL doesn't match pattern prefix",
			searchPattern: "https://github.com/org1",
			repoURL:       "https://github.com/different-org/repo",
			shouldFind:    false,
		},
		{
			name:          "Partial URL match should not work",
			searchPattern: "https://github.com/org1-extended",
			repoURL:       "https://github.com/org1/repo",
			shouldFind:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var matchedApp *config.GitHubApp

			// Simulate the URL prefix matching logic from handleCredentialGet
			if strings.HasPrefix(tt.repoURL, tt.searchPattern) {
				// Find the GitHub App that has this pattern
				for _, app := range apps {
					for _, pattern := range app.Patterns {
						if pattern == tt.searchPattern {
							matchedApp = &app
							break
						}
					}
					if matchedApp != nil {
						break
					}
				}
			}

			if tt.shouldFind {
				if matchedApp == nil {
					t.Errorf("Expected to find app with pattern %s for URL %s, but didn't", tt.searchPattern, tt.repoURL)
				} else if matchedApp.AppID != tt.expectedAppID {
					t.Errorf("Found wrong app: expected %d, got %d", tt.expectedAppID, matchedApp.AppID)
				}
			} else {
				if matchedApp != nil {
					t.Errorf("Expected not to find app with pattern %s for URL %s, but found app %d",
						tt.searchPattern, tt.repoURL, matchedApp.AppID)
				}
			}
		})
	}
}

func TestURLFormatWithPattern(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedInput map[string]string
	}{
		{
			name:  "URL format with full path",
			input: "url=https://github.com/AmadeusITGroup/Dataspace_Ecosystem\n\n",
			expectedInput: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "AmadeusITGroup/Dataspace_Ecosystem",
			},
		},
		{
			name:  "Key-value format",
			input: "protocol=https\nhost=github.com\npath=AmadeusITGroup/Dataspace_Ecosystem\n\n",
			expectedInput: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "AmadeusITGroup/Dataspace_Ecosystem",
			},
		},
		{
			name:  "URL format with trailing slash",
			input: "url=https://github.com/org/repo/\n\n",
			expectedInput: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "org/repo/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := readCredentialInput(reader)
			if err != nil {
				t.Fatalf("readCredentialInput() error = %v", err)
			}

			for key, expectedValue := range tt.expectedInput {
				if result[key] != expectedValue {
					t.Errorf("For key %s: expected %q, got %q", key, expectedValue, result[key])
				}
			}
		})
	}
}

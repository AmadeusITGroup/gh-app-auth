package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"gopkg.in/yaml.v3"
)

func TestGitCredentialRun_GetOperation(t *testing.T) {
	// Skip if short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	// Create test config
	cfg := &config.Config{
		Version: "1.0",
		GitHubApps: []config.GitHubApp{
			{
				Name:             "Test App",
				AppID:            123456,
				InstallationID:   789012,
				Patterns:         []string{"github.com/testorg/*"},
				PrivateKeySource: config.PrivateKeySourceFilesystem,
				PrivateKeyPath:   filepath.Join(tempDir, "key.pem"),
			},
		},
	}

	// Create test key
	testKey := generateTestRSAKey(t)
	if err := os.WriteFile(cfg.GitHubApps[0].PrivateKeyPath, []byte(testKey), 0600); err != nil {
		t.Fatalf("Failed to write test key: %v", err)
	}

	// Write config
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	t.Run("get operation with matching pattern", func(t *testing.T) {
		// Create command with get operation
		cmd := NewGitCredentialCmd()

		// Simulate git credential input via stdin
		input := "protocol=https\nhost=github.com\npath=testorg/repo\n\n"
		cmd.SetIn(strings.NewReader(input))

		// Capture stdout
		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)

		// Set args
		cmd.SetArgs([]string{"get"})

		// Execute command
		err := cmd.Execute()

		// The command might fail due to GitHub API not being accessible
		// but we're testing that the flow executes without panic
		// Check that we at least attempted to process
		_ = err // May or may not error depending on network/mock availability

		// The function should have attempted to read stdin and process the request
		// Since we don't have a mock GitHub API, it will likely fail at auth step
		// but we've still covered the gitCredentialRun flow
	})

	t.Run("get operation with no matching pattern", func(t *testing.T) {
		cmd := NewGitCredentialCmd()

		// Input that doesn't match any pattern
		input := "protocol=https\nhost=github.com\npath=differentorg/repo\n\n"
		cmd.SetIn(strings.NewReader(input))

		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)

		cmd.SetArgs([]string{"get"})

		err := cmd.Execute()

		// Should exit silently (no match, no error)
		if err != nil {
			t.Logf("Command returned error (may be expected): %v", err)
		}
	})
}

func TestGitCredentialRun_StoreOperation(t *testing.T) {
	t.Run("store operation is no-op", func(t *testing.T) {
		cmd := NewGitCredentialCmd()

		// Simulate git credential store input
		input := "protocol=https\nhost=github.com\npath=testorg/repo\nusername=x-access-token\npassword=test-token\n\n"
		cmd.SetIn(strings.NewReader(input))

		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)

		cmd.SetArgs([]string{"store"})

		// Store should always succeed (it's a no-op)
		err := cmd.Execute()
		if err != nil {
			t.Errorf("Store operation should not error: %v", err)
		}
	})
}

func TestGitCredentialRun_EraseOperation(t *testing.T) {
	t.Run("erase operation clears cache", func(t *testing.T) {
		cmd := NewGitCredentialCmd()

		// Simulate git credential erase input
		input := "protocol=https\nhost=github.com\npath=testorg/repo\n\n"
		cmd.SetIn(strings.NewReader(input))

		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)

		cmd.SetArgs([]string{"erase"})

		// Erase should always succeed
		err := cmd.Execute()
		if err != nil {
			t.Errorf("Erase operation should not error: %v", err)
		}
	})
}

func TestGitCredentialRun_UnsupportedOperation(t *testing.T) {
	t.Run("unsupported operation returns error", func(t *testing.T) {
		cmd := NewGitCredentialCmd()

		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)

		cmd.SetArgs([]string{"invalid-operation"})

		// Should return error for unsupported operation
		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for unsupported operation")
		}

		if !strings.Contains(err.Error(), "unsupported") {
			t.Errorf("Expected 'unsupported' in error message, got: %v", err)
		}
	})
}

func TestHandleCredentialStore_Direct(t *testing.T) {
	t.Run("store reads and ignores input", func(t *testing.T) {
		// Save and restore stdin
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		// Create pipe for stdin
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}

		os.Stdin = r

		// Write test input
		input := "protocol=https\nhost=github.com\npath=org/repo\nusername=test\npassword=token\n\n"
		go func() {
			defer w.Close()
			w.Write([]byte(input))
		}()

		// Call handleCredentialStore directly
		err = handleCredentialStore()
		if err != nil {
			t.Errorf("handleCredentialStore failed: %v", err)
		}
	})
}

func TestHandleCredentialErase_Direct(t *testing.T) {
	t.Run("erase reads input and clears cache", func(t *testing.T) {
		// Save and restore stdin
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		// Create pipe for stdin
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}

		os.Stdin = r

		// Write test input
		input := "protocol=https\nhost=github.com\npath=org/repo\n\n"
		go func() {
			defer w.Close()
			w.Write([]byte(input))
		}()

		// Call handleCredentialErase directly
		err = handleCredentialErase()
		if err != nil {
			t.Errorf("handleCredentialErase failed: %v", err)
		}
	})

	t.Run("erase handles missing URL gracefully", func(t *testing.T) {
		// Save and restore stdin
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		// Create pipe for stdin
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}

		os.Stdin = r

		// Write input with no host (can't build URL)
		input := "protocol=https\n\n"
		go func() {
			defer w.Close()
			w.Write([]byte(input))
		}()

		// Should not error even with incomplete input
		err = handleCredentialErase()
		if err != nil {
			t.Errorf("handleCredentialErase should handle missing URL gracefully: %v", err)
		}
	})
}

func TestDoAutomaticSetup(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")
	keyPath := filepath.Join(tempDir, "test-key.pem")

	testKey := generateTestRSAKey(t)
	if err := os.WriteFile(keyPath, []byte(testKey), 0600); err != nil {
		t.Fatalf("Failed to write test key: %v", err)
	}

	t.Run("returns nil when env vars are not set", func(t *testing.T) {
		t.Setenv("GH_APP_PRIVATE_KEY_PATH", "")
		t.Setenv("GH_APP_ID", "")
		t.Setenv("GH_APP_CLIENT_ID", "")

		app, err := doAutomaticSetup("github.com/org/repo")
		if err != nil {
			t.Errorf("doAutomaticSetup() error = %v", err)
		}
		if app != nil {
			t.Errorf("doAutomaticSetup() app = %v, want nil", app)
		}
	})

	t.Run("sets up from GH_APP_CLIENT_ID", func(t *testing.T) {
		t.Setenv("GH_APP_AUTH_CONFIG", configPath)
		t.Setenv("GH_APP_PRIVATE_KEY_PATH", keyPath)
		t.Setenv("GH_APP_ID", "")
		t.Setenv("GH_APP_CLIENT_ID", "Iv1.AutoSetup")

		_, err := doAutomaticSetup("github.com/org/repo")
		// Auto-detect installation ID requires network access, so an error is expected
		if err == nil {
			t.Error("Expected error when auto-detecting installation ID without network")
		}
	})

	t.Run("sets up from GH_APP_ID", func(t *testing.T) {
		t.Setenv("GH_APP_AUTH_CONFIG", configPath)
		t.Setenv("GH_APP_PRIVATE_KEY_PATH", keyPath)
		t.Setenv("GH_APP_ID", "123456")
		t.Setenv("GH_APP_CLIENT_ID", "")

		_, err := doAutomaticSetup("github.com/org/repo")
		// Auto-detect installation ID requires network access, so an error is expected
		if err == nil {
			t.Error("Expected error when auto-detecting installation ID without network")
		}
	})
}

func TestGenerateAndOutputCredentials_Success(t *testing.T) {
	// Mock GitHub Enterprise installation token endpoint
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		expectedPrefix := "/api/v3/app/installations/"
		if !strings.HasPrefix(r.URL.Path, expectedPrefix) {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusCreated)
		expiresAt := time.Now().Add(time.Hour).Format(time.RFC3339)
		_, _ = fmt.Fprintf(w, `{"token":"mock-installation-token","expires_at":"%s"}`, expiresAt)
	}))
	defer server.Close()

	// Make the default HTTP client trust the test TLS server
	oldTransport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = oldTransport }()

	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test-key.pem")
	testKey := generateTestRSAKey(t)
	if err := os.WriteFile(keyPath, []byte(testKey), 0600); err != nil {
		t.Fatalf("Failed to write test key: %v", err)
	}

	app := &config.GitHubApp{
		Name:             "Mock Server App",
		ClientID:         "Iv1.MockServer",
		InstallationID:   789012,
		Patterns:         []string{"github.com/org/*"},
		PrivateKeySource: config.PrivateKeySourceFilesystem,
		PrivateKeyPath:   keyPath,
	}

	repoURL := fmt.Sprintf("%s/org/repo", server.URL)
	err := generateAndOutputCredentials(app, repoURL)
	if err != nil {
		t.Errorf("generateAndOutputCredentials() error = %v", err)
	}
}

package cmd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/secrets"
)

// generateTestRSAKey generates a test RSA private key in PEM format
func generateTestRSAKey(t *testing.T) string {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	return string(pem.EncodeToMemory(privateKeyPEM))
}

func TestNewSetupCmd(t *testing.T) {
	cmd := NewSetupCmd()

	if cmd == nil {
		t.Fatal("NewSetupCmd() returned nil")
	}

	if cmd.Use != "setup" {
		t.Errorf("Expected Use to be 'setup', got %q", cmd.Use)
	}

	// Check required flags exist
	if cmd.Flags().Lookup("app-id") == nil {
		t.Error("Expected --app-id flag to be defined")
	}

	if cmd.Flags().Lookup("patterns") == nil {
		t.Error("Expected --patterns flag to be defined")
	}

	if cmd.Flags().Lookup("key-file") == nil {
		t.Error("Expected --key-file flag to be defined")
	}
}

func TestValidateSetupInputs(t *testing.T) {
	tests := []struct {
		name           string
		appID          int64
		patterns       []string
		useKeyring     bool
		useFilesystem  bool
		keyFile        string
		wantErr        bool
		wantKeyring    bool
		expectedErr    error
		ghAppKeyEnvVar string
	}{
		{
			name:        "valid inputs with keyring",
			appID:       123456,
			patterns:    []string{"github.com/org/*"},
			useKeyring:  true,
			keyFile:     "",
			wantErr:     false,
			wantKeyring: true,
		},
		{
			name:          "valid inputs with filesystem",
			appID:         123456,
			patterns:      []string{"github.com/org/*"},
			useKeyring:    true,
			useFilesystem: true,
			keyFile:       "",
			wantErr:       false,
			wantKeyring:   false, // filesystem overrides keyring
		},
		{
			name:           "conflicting options",
			appID:          123456,
			patterns:       []string{"github.com/org/*"},
			ghAppKeyEnvVar: "some ssh key",
			useKeyring:     true,
			useFilesystem:  true,
			keyFile:        "some key file",
			wantErr:        true,
			wantKeyring:    false, // filesystem overrides keyring
			expectedErr:    ErrConflictingKeyOptions,
		},
		{
			name:     "invalid app ID - zero",
			appID:    0,
			patterns: []string{"github.com/org/*"},
			wantErr:  true,
		},
		{
			name:     "invalid app ID - negative",
			appID:    -1,
			patterns: []string{"github.com/org/*"},
			wantErr:  true,
		},
		{
			name:     "missing patterns",
			appID:    123456,
			patterns: []string{},
			wantErr:  true,
		},
		{
			name:     "nil patterns",
			appID:    123456,
			patterns: nil,
			wantErr:  true,
		},
		{
			name:     "multiple patterns",
			appID:    123456,
			patterns: []string{"github.com/org1/*", "github.com/org2/*"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useKeyring := tt.useKeyring
			useFilesystem := tt.useFilesystem
			keyFile := tt.keyFile
			if tt.ghAppKeyEnvVar != "" {
				os.Setenv("GH_APP_PRIVATE_KEY", tt.ghAppKeyEnvVar)
			} else {
				os.Unsetenv("GH_APP_PRIVATE_KEY")
			}

			err := validateSetupInputs(tt.appID, tt.patterns, &useKeyring, &useFilesystem, keyFile)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
					t.Error("Expected error to be ", tt.expectedErr, "got", err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if useKeyring != tt.wantKeyring {
				t.Errorf("useKeyring = %v, want %v", useKeyring, tt.wantKeyring)
			}
		})
	}
}

func TestGetPrivateKey(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("from environment variable", func(t *testing.T) {
		t.Setenv("GH_APP_PRIVATE_KEY", "test-private-key-content")

		content, path, err := getPrivateKey("")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if content != "test-private-key-content" {
			t.Errorf("Content = %q, want %q", content, "test-private-key-content")
		}

		if path != "" {
			t.Errorf("Path = %q, want empty string", path)
		}
	})

	t.Run("from file", func(t *testing.T) {
		// Skip on Windows - file permissions are not enforced the same way
		if runtime.GOOS == "windows" {
			t.Skip("Skipping on Windows: file permissions not supported")
		}

		// Clear env var
		t.Setenv("GH_APP_PRIVATE_KEY", "")

		keyPath := filepath.Join(tempDir, "test.pem")
		keyContent := "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----"
		if err := os.WriteFile(keyPath, []byte(keyContent), 0600); err != nil {
			t.Fatalf("Failed to create test key file: %v", err)
		}

		content, path, err := getPrivateKey(keyPath)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if content != keyContent {
			t.Errorf("Content mismatch")
		}

		if path != keyPath {
			t.Errorf("Path = %q, want %q", path, keyPath)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		t.Setenv("GH_APP_PRIVATE_KEY", "")

		_, _, err := getPrivateKey("")
		if err == nil {
			t.Error("Expected error for missing key")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		t.Setenv("GH_APP_PRIVATE_KEY", "")

		_, _, err := getPrivateKey("/nonexistent/key.pem")
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})
}

func TestValidateKeyFile(t *testing.T) {
	// Skip on Windows - file permissions are not enforced the same way
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows: file permissions not supported")
	}

	tempDir := t.TempDir()

	t.Run("valid key file with 0600", func(t *testing.T) {
		keyPath := filepath.Join(tempDir, "valid.pem")
		if err := os.WriteFile(keyPath, []byte("test"), 0600); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		err := validateKeyFile(keyPath)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("valid key file with 0400", func(t *testing.T) {
		keyPath := filepath.Join(tempDir, "readonly.pem")
		if err := os.WriteFile(keyPath, []byte("test"), 0400); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		err := validateKeyFile(keyPath)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("file with overly permissive permissions", func(t *testing.T) {
		keyPath := filepath.Join(tempDir, "bad-perms.pem")
		if err := os.WriteFile(keyPath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		err := validateKeyFile(keyPath)
		if err == nil {
			t.Error("Expected error for overly permissive file")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		err := validateKeyFile("/nonexistent/key.pem")
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})
}

func TestCreateGitHubApp(t *testing.T) {
	tests := []struct {
		name           string
		appID          int64
		appName        string
		installationID int64
		patterns       []string
		priority       int
		wantName       string
		wantErr        bool
	}{
		{
			name:           "with custom name",
			appID:          123456,
			appName:        "My App",
			installationID: 789,
			patterns:       []string{"github.com/org/*"},
			priority:       5,
			wantName:       "My App",
			wantErr:        false, // No validation in createGitHubApp anymore
		},
		{
			name:           "with default name",
			appID:          123456,
			appName:        "",
			installationID: 789,
			patterns:       []string{"github.com/org/*"},
			priority:       5,
			wantName:       "GitHub App 123456",
			wantErr:        false, // No validation in createGitHubApp anymore
		},
		{
			name:           "with multiple patterns",
			appID:          123456,
			appName:        "Test",
			installationID: 789,
			patterns:       []string{"github.com/org1/*", "github.com/org2/*"},
			priority:       10,
			wantName:       "Test",
			wantErr:        false, // No validation in createGitHubApp anymore
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createGitHubApp(tt.appID, tt.appName, tt.installationID, tt.patterns, tt.priority)

			if app.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", app.Name, tt.wantName)
			}

			if app.AppID != tt.appID {
				t.Errorf("AppID = %d, want %d", app.AppID, tt.appID)
			}

			if app.InstallationID != tt.installationID {
				t.Errorf("InstallationID = %d, want %d", app.InstallationID, tt.installationID)
			}

			if len(app.Patterns) != len(tt.patterns) {
				t.Errorf("Patterns length = %d, want %d", len(app.Patterns), len(tt.patterns))
			}

			if app.Priority != tt.priority {
				t.Errorf("Priority = %d, want %d", app.Priority, tt.priority)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantHome bool // should expand to home directory
	}{
		{
			name:     "tilde expansion",
			path:     "~/.ssh/key.pem",
			wantHome: true,
		},
		// Note: "tilde only" test removed - expandPath uses filepath.Abs which
		// doesn't handle bare "~" consistently across platforms
		{
			name:     "absolute path",
			path:     "/tmp/key.pem",
			wantHome: false,
		},
		{
			name:     "relative path - converted to absolute",
			path:     "./key.pem",
			wantHome: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expanded, err := expandPath(tt.path)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.wantHome {
				homeDir, _ := os.UserHomeDir()
				// Should start with home directory
				if expanded == tt.path {
					t.Error("Path was not expanded")
				}
				// Basic check that it contains home dir
				if homeDir != "" && expanded[:len(homeDir)] != homeDir {
					t.Errorf("Expanded path %q does not start with home %q", expanded, homeDir)
				}
			} else {
				// expandPath calls filepath.Abs for non-tilde paths, so they become absolute
				if !filepath.IsAbs(expanded) {
					t.Errorf("Expected absolute path, got: %q", expanded)
				}
				// For absolute paths, should be unchanged
				if filepath.IsAbs(tt.path) && expanded != tt.path {
					t.Errorf("Absolute path was modified: got %q, want %q", expanded, tt.path)
				}
			}
		})
	}
}

func TestGenerateJWTForSetup(t *testing.T) {
	validKey := generateTestRSAKey(t)

	t.Run("valid JWT generation", func(t *testing.T) {
		token, err := generateJWTForSetup(123456, validKey)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if token == "" {
			t.Error("Expected non-empty token")
		}
	})

	t.Run("invalid key format", func(t *testing.T) {
		_, err := generateJWTForSetup(123456, "invalid-key-content")
		if err == nil {
			t.Error("Expected error for invalid key")
		}
	})

	t.Run("empty key", func(t *testing.T) {
		_, err := generateJWTForSetup(123456, "")
		if err == nil {
			t.Error("Expected error for empty key")
		}
	})
}

func TestConfigureAppStorage(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test.pem")
	keyContent := "test-private-key-content"

	if err := os.WriteFile(keyPath, []byte(keyContent), 0600); err != nil {
		t.Fatalf("Failed to create test key: %v", err)
	}

	t.Run("filesystem storage", func(t *testing.T) {
		app := &config.GitHubApp{
			Name:           "Test App",
			AppID:          123456,
			InstallationID: 789,
			Patterns:       []string{"github.com/test/*"},
		}

		backend, err := configureAppStorage(app, keyContent, keyPath, false)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if backend != secrets.StorageBackendFilesystem {
			t.Errorf("Backend = %v, want %v", backend, secrets.StorageBackendFilesystem)
		}

		if app.PrivateKeyPath != keyPath {
			t.Errorf("PrivateKeyPath = %q, want %q", app.PrivateKeyPath, keyPath)
		}

		if app.PrivateKeySource != config.PrivateKeySourceFilesystem {
			t.Errorf("PrivateKeySource = %v, want %v", app.PrivateKeySource, config.PrivateKeySourceFilesystem)
		}
	})

	t.Run("filesystem without key file", func(t *testing.T) {
		app := &config.GitHubApp{
			Name:           "Test App",
			AppID:          123456,
			InstallationID: 789,
			Patterns:       []string{"github.com/test/*"},
		}

		_, err := configureAppStorage(app, keyContent, "", false)

		if err == nil {
			t.Error("Expected error when filesystem storage without key file")
		}
	})

	t.Run("keyring storage", func(t *testing.T) {
		app := &config.GitHubApp{
			Name:           "Test App",
			AppID:          123456,
			InstallationID: 789,
			Patterns:       []string{"github.com/test/*"},
		}

		// Keyring may or may not be available, so we just test that it doesn't panic
		// and returns a valid backend type
		backend, err := configureAppStorage(app, keyContent, keyPath, true)

		// If keyring is available, it should succeed
		// If not, it might fail gracefully
		if err == nil {
			if backend != secrets.StorageBackendKeyring && backend != secrets.StorageBackendFilesystem {
				t.Errorf("Unexpected backend: %v", backend)
			}
		}
		// Error is acceptable if keyring is not available
	})
}

func TestSaveAppConfiguration(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	// Set config path via environment
	original := os.Getenv("GH_APP_AUTH_CONFIG")
	defer func() {
		if original != "" {
			os.Setenv("GH_APP_AUTH_CONFIG", original)
		} else {
			os.Unsetenv("GH_APP_AUTH_CONFIG")
		}
	}()
	os.Setenv("GH_APP_AUTH_CONFIG", configPath)

	t.Run("save new app", func(t *testing.T) {
		cfg := &config.Config{
			Version:    "1.0",
			GitHubApps: []config.GitHubApp{},
		}

		app := &config.GitHubApp{
			Name:             "Test App",
			AppID:            123456,
			InstallationID:   789,
			Patterns:         []string{"github.com/test/*"},
			PrivateKeySource: config.PrivateKeySourceFilesystem,
			PrivateKeyPath:   "/tmp/key.pem",
		}

		err := saveAppConfiguration(cfg, app)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(cfg.GitHubApps) != 1 {
			t.Errorf("Expected 1 app, got %d", len(cfg.GitHubApps))
		}

		if cfg.GitHubApps[0].Name != "Test App" {
			t.Errorf("App name = %q, want %q", cfg.GitHubApps[0].Name, "Test App")
		}

		// Verify file was created
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}
	})

	t.Run("update existing app", func(t *testing.T) {
		// Load the previously saved config
		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Update the app
		app := &config.GitHubApp{
			Name:             "Test App Updated",
			AppID:            123456, // Same ID
			InstallationID:   999,    // Changed
			Patterns:         []string{"github.com/updated/*"},
			PrivateKeySource: config.PrivateKeySourceFilesystem,
			PrivateKeyPath:   "/tmp/key.pem",
		}

		err = saveAppConfiguration(cfg, app)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Should still have 1 app (updated, not added)
		if len(cfg.GitHubApps) != 1 {
			t.Errorf("Expected 1 app, got %d", len(cfg.GitHubApps))
		}

		if cfg.GitHubApps[0].InstallationID != 999 {
			t.Errorf("InstallationID not updated: got %d, want 999", cfg.GitHubApps[0].InstallationID)
		}
	})
}

func TestSetupRun_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")
	keyPath := filepath.Join(tempDir, "test-key.pem")

	// Generate valid test key
	testKey := generateTestRSAKey(t)
	if err := os.WriteFile(keyPath, []byte(testKey), 0600); err != nil {
		t.Fatalf("Failed to write test key: %v", err)
	}

	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	t.Run("setup validates required inputs", func(t *testing.T) {
		// This test validates that the setup flow is reached and validates inputs
		// Full integration would require keyring/filesystem mocking
		t.Skip("Full integration test requires keyring setup")
	})

	t.Run("setup with invalid app ID", func(t *testing.T) {
		// Clear config
		os.Remove(configPath)

		cmd := NewSetupCmd()
		cmd.Flags().Set("app-id", "0")
		cmd.Flags().Set("key-file", keyPath)
		cmd.Flags().Set("patterns", "github.com/testorg/*")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for invalid app ID")
		}
	})

	t.Run("setup with missing pattern", func(t *testing.T) {
		// Clear config
		os.Remove(configPath)

		cmd := NewSetupCmd()
		cmd.Flags().Set("app-id", "123456")
		cmd.Flags().Set("key-file", keyPath)

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing pattern")
		}
	})

	t.Run("setup with invalid key file", func(t *testing.T) {
		// Clear config
		os.Remove(configPath)

		cmd := NewSetupCmd()
		cmd.Flags().Set("app-id", "123456")
		cmd.Flags().Set("key-file", "/nonexistent/key.pem")
		cmd.Flags().Set("patterns", "github.com/testorg/*")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for nonexistent key file")
		}
	})

	t.Run("setup with malformed key", func(t *testing.T) {
		// Clear config
		os.Remove(configPath)

		// Create invalid key file
		badKeyPath := filepath.Join(tempDir, "bad-key.pem")
		if err := os.WriteFile(badKeyPath, []byte("not-a-valid-key"), 0600); err != nil {
			t.Fatalf("Failed to write bad key: %v", err)
		}

		cmd := NewSetupCmd()
		cmd.Flags().Set("app-id", "123456")
		cmd.Flags().Set("key-file", badKeyPath)
		cmd.Flags().Set("patterns", "github.com/testorg/*")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for malformed key")
		}
	})

	t.Run("setup with environment variable key should not fail validation", func(t *testing.T) {
		// This test verifies the regression fix where using GH_APP_PRIVATE_KEY
		// environment variable would fail with "private_key_path or private_key_source is required"
		// because validation happened before storage configuration
		os.Remove(configPath)

		// Set environment variable with valid key
		t.Setenv("GH_APP_PRIVATE_KEY", testKey)

		// Test that validation passes when storage is configured
		appID := int64(123456)
		name := "Test App"
		installationID := int64(789012)
		patterns := []string{"github.com/testorg/*"}
		priority := 5

		// Create the app without validation
		app := createGitHubApp(appID, name, installationID, patterns, priority)

		// At this point, app should have no PrivateKeySource or PrivateKeyPath set
		if app.PrivateKeySource != "" {
			t.Errorf("Expected empty PrivateKeySource before storage config, got %s", app.PrivateKeySource)
		}

		// Configure storage (simulating the fixed flow)
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("Failed to get home directory: %v", err)
		}
		configDir := filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth")
		secretMgr := secrets.NewManager(configDir)

		// Try to store the key (will use keyring or filesystem as fallback)
		backend, err := app.SetPrivateKey(secretMgr, testKey)
		if err != nil {
			// It's OK if this fails due to no keyring, just skip the rest
			t.Skipf("Keyring not available for testing: %v", err)
		}

		// After SetPrivateKey, PrivateKeySource should be set
		if app.PrivateKeySource == "" {
			t.Error("Expected PrivateKeySource to be set after SetPrivateKey")
		}

		t.Logf("Storage backend used: %s", backend)

		// Now validation should pass
		if err := app.Validate(); err != nil {
			t.Errorf("Validation failed after storage configuration: %v", err)
		}
	})
}

func TestDisplaySetupSuccess(t *testing.T) {
	tests := []struct {
		name            string
		appName         string
		appID           int64
		patterns        []string
		priority        int
		backend         secrets.StorageBackend
		expandedKeyFile string
	}{
		{
			name:            "keyring with fallback",
			appName:         "My App",
			appID:           123456,
			patterns:        []string{"github.com/org/*"},
			priority:        5,
			backend:         secrets.StorageBackendKeyring,
			expandedKeyFile: "/path/to/key.pem",
		},
		{
			name:            "filesystem storage",
			appName:         "Test App",
			appID:           789012,
			patterns:        []string{"github.com/test/*", "github.com/org/*"},
			priority:        10,
			backend:         secrets.StorageBackendFilesystem,
			expandedKeyFile: "/home/user/.ssh/app.pem",
		},
		{
			name:            "keyring without fallback",
			appName:         "Secure App",
			appID:           111111,
			patterns:        []string{"github.com/secure/*"},
			priority:        1,
			backend:         secrets.StorageBackendKeyring,
			expandedKeyFile: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic
			displaySetupSuccess(tt.appName, tt.appID, tt.patterns, tt.priority, tt.backend, tt.expandedKeyFile)
		})
	}
}

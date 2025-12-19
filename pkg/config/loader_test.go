package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoader_Load(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-app-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test configuration
	testConfig := Config{
		Version: "1.0",
		GitHubApps: []GitHubApp{
			{
				Name:           "test-app",
				AppID:          12345,
				InstallationID: 67890,
				PrivateKeyPath: "/tmp/key.pem",
				Patterns:       []string{"github.com/org/*"},
				Priority:       100,
			},
		},
	}

	tests := []struct {
		name       string
		setupFile  func(string) error
		configPath string
		wantErr    bool
		errMsg     string
	}{
		{
			name: "valid YAML file",
			setupFile: func(path string) error {
				data, err := yaml.Marshal(testConfig)
				if err != nil {
					return err
				}
				return os.WriteFile(path, data, 0644)
			},
			configPath: filepath.Join(tmpDir, "config.yml"),
			wantErr:    false,
		},
		{
			name: "valid JSON file",
			setupFile: func(path string) error {
				data, err := json.MarshalIndent(testConfig, "", "  ")
				if err != nil {
					return err
				}
				return os.WriteFile(path, data, 0644)
			},
			configPath: filepath.Join(tmpDir, "config.json"),
			wantErr:    false,
		},
		{
			name: "file does not exist",
			setupFile: func(path string) error {
				return nil // Don't create file
			},
			configPath: filepath.Join(tmpDir, "nonexistent.yml"),
			wantErr:    true,
			errMsg:     "configuration file not found",
		},
		{
			name: "invalid YAML",
			setupFile: func(path string) error {
				return os.WriteFile(path, []byte("invalid: yaml: content: ["), 0644)
			},
			configPath: filepath.Join(tmpDir, "invalid.yml"),
			wantErr:    true,
			errMsg:     "failed to parse configuration",
		},
		{
			name: "invalid JSON",
			setupFile: func(path string) error {
				return os.WriteFile(path, []byte(`{"invalid": json}`), 0644)
			},
			configPath: filepath.Join(tmpDir, "invalid.json"),
			wantErr:    true,
			errMsg:     "failed to parse configuration",
		},
		{
			name: "invalid configuration content",
			setupFile: func(path string) error {
				invalidConfig := Config{
					Version:    "", // Missing version
					GitHubApps: []GitHubApp{},
				}
				data, err := yaml.Marshal(invalidConfig)
				if err != nil {
					return err
				}
				return os.WriteFile(path, data, 0644)
			},
			configPath: filepath.Join(tmpDir, "invalid-content.yml"),
			wantErr:    true,
			errMsg:     "version is required",
		},
		{
			name: "no extension - tries YAML first",
			setupFile: func(path string) error {
				data, err := yaml.Marshal(testConfig)
				if err != nil {
					return err
				}
				return os.WriteFile(path, data, 0644)
			},
			configPath: filepath.Join(tmpDir, "config"),
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test file
			if err := tt.setupFile(tt.configPath); err != nil {
				t.Fatalf("Failed to setup test file: %v", err)
			}

			loader := NewLoader(tt.configPath)
			config, err := loader.Load()

			if (err != nil) != tt.wantErr {
				t.Errorf("Loader.Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Loader.Load() error = %v, want error containing %v", err.Error(), tt.errMsg)
				}
			} else {
				if config == nil {
					t.Error("Loader.Load() returned nil config without error")
					return
				}
				if config.Version != testConfig.Version {
					t.Errorf("Loader.Load() config.Version = %v, want %v", config.Version, testConfig.Version)
				}
				if len(config.GitHubApps) != len(testConfig.GitHubApps) {
					t.Errorf("Loader.Load() len(config.GitHubApps) = %v, want %v", len(config.GitHubApps), len(testConfig.GitHubApps))
				}
			}
		})
	}
}

func TestLoader_LoadWithFallback(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-app-config-fallback-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name       string
		setupFile  func(string) error
		configPath string
		wantConfig bool
		wantErr    bool
	}{
		{
			name: "file exists and valid",
			setupFile: func(path string) error {
				config := Config{
					Version: "1.0",
					GitHubApps: []GitHubApp{
						{
							Name:           "test-app",
							AppID:          12345,
							InstallationID: 67890,
							PrivateKeyPath: "/tmp/key.pem",
							Patterns:       []string{"github.com/org/*"},
							Priority:       100,
						},
					},
				}
				data, err := yaml.Marshal(config)
				if err != nil {
					return err
				}
				return os.WriteFile(path, data, 0644)
			},
			configPath: filepath.Join(tmpDir, "exists.yml"),
			wantConfig: true,
			wantErr:    false,
		},
		{
			name: "file does not exist",
			setupFile: func(path string) error {
				return nil // Don't create file
			},
			configPath: filepath.Join(tmpDir, "nonexistent.yml"),
			wantConfig: false,
			wantErr:    false,
		},
		{
			name: "file exists but invalid",
			setupFile: func(path string) error {
				return os.WriteFile(path, []byte("invalid yaml ["), 0644)
			},
			configPath: filepath.Join(tmpDir, "invalid.yml"),
			wantConfig: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test file
			if err := tt.setupFile(tt.configPath); err != nil {
				t.Fatalf("Failed to setup test file: %v", err)
			}

			loader := NewLoader(tt.configPath)
			config, err := loader.LoadWithFallback()

			if (err != nil) != tt.wantErr {
				t.Errorf("Loader.LoadWithFallback() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantConfig && config == nil {
				t.Error("Loader.LoadWithFallback() returned nil config when config expected")
			}

			if !tt.wantConfig && config != nil {
				t.Error("Loader.LoadWithFallback() returned config when nil expected")
			}
		})
	}
}

func TestNewDefaultLoader(t *testing.T) {
	loader := NewDefaultLoader()
	if loader == nil {
		t.Fatal("NewDefaultLoader() returned nil")
	}

	configPath := loader.GetConfigPath()
	if configPath == "" {
		t.Error("NewDefaultLoader() returned loader with empty config path")
	}

	// Should contain the extension config path (handle both Unix / and Windows \ separators)
	expectedParts := []string{".config", "gh", "extensions", "gh-app-auth", "config.yml"}
	containsAll := true
	for _, part := range expectedParts {
		if !containsString(configPath, part) {
			containsAll = false
			break
		}
	}
	if !containsAll {
		t.Errorf("NewDefaultLoader() config path = %v, want path containing extension config parts", configPath)
	}
}

func TestConfigExists(t *testing.T) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "config-exists-test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	tests := []struct {
		name       string
		configPath string
		want       bool
	}{
		{
			name:       "file exists",
			configPath: tmpFile.Name(),
			want:       true,
		},
		{
			name:       "file does not exist",
			configPath: "/nonexistent/path/config.yml",
			want:       false,
		},
		{
			name:       "empty path",
			configPath: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConfigExists(tt.configPath); got != tt.want {
				t.Errorf("ConfigExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDefaultConfigPath_WithEnvVar(t *testing.T) {
	// Save original env var
	originalEnv := os.Getenv("GH_APP_AUTH_CONFIG")
	defer func() {
		if originalEnv != "" {
			os.Setenv("GH_APP_AUTH_CONFIG", originalEnv)
		} else {
			os.Unsetenv("GH_APP_AUTH_CONFIG")
		}
	}()

	// Test with environment variable
	// Use platform-agnostic path
	expectedPath := filepath.Join("custom", "config", "path.yml")
	os.Setenv("GH_APP_AUTH_CONFIG", expectedPath)

	path := getDefaultConfigPath()
	// Check that the path contains the expected parts
	if !containsString(path, "custom") || !containsString(path, "config") || !containsString(path, "path.yml") {
		t.Errorf("getDefaultConfigPath() with env var = %v, want path containing custom/config/path.yml", path)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

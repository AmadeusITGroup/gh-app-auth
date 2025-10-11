package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
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
			},
			wantErr: false,
		},
		{
			name: "missing version",
			config: Config{
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
			},
			wantErr: true,
			errMsg:  "version is required",
		},
		{
			name: "no github apps",
			config: Config{
				Version:    "1.0",
				GitHubApps: []GitHubApp{},
			},
			wantErr: true,
			errMsg:  "at least one github_app or pat is required",
		},
		{
			name: "invalid github app",
			config: Config{
				Version: "1.0",
				GitHubApps: []GitHubApp{
					{
						Name:           "",
						AppID:          12345,
						InstallationID: 67890,
						PrivateKeyPath: "/tmp/key.pem",
						Patterns:       []string{"github.com/org/*"},
						Priority:       100,
					},
				},
			},
			wantErr: true,
			errMsg:  "github_apps[0]: name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Config.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestGitHubApp_Validate(t *testing.T) {
	tests := []struct {
		name    string
		app     GitHubApp
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid app",
			app: GitHubApp{
				Name:           "test-app",
				AppID:          12345,
				InstallationID: 67890,
				PrivateKeyPath: "/tmp/key.pem",
				Patterns:       []string{"github.com/org/*"},
				Priority:       100,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			app: GitHubApp{
				AppID:          12345,
				InstallationID: 67890,
				PrivateKeyPath: "/tmp/key.pem",
				Patterns:       []string{"github.com/org/*"},
				Priority:       100,
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "invalid app_id",
			app: GitHubApp{
				Name:           "test-app",
				AppID:          0,
				InstallationID: 67890,
				PrivateKeyPath: "/tmp/key.pem",
				Patterns:       []string{"github.com/org/*"},
				Priority:       100,
			},
			wantErr: true,
			errMsg:  "app_id must be positive",
		},
		{
			name: "invalid installation_id",
			app: GitHubApp{
				Name:           "test-app",
				AppID:          12345,
				InstallationID: -1,
				PrivateKeyPath: "/tmp/key.pem",
				Patterns:       []string{"github.com/org/*"},
				Priority:       100,
			},
			wantErr: true,
			errMsg:  "installation_id must be positive",
		},
		{
			name: "missing private_key_path",
			app: GitHubApp{
				Name:           "test-app",
				AppID:          12345,
				InstallationID: 67890,
				Patterns:       []string{"github.com/org/*"},
				Priority:       100,
			},
			wantErr: true,
			errMsg:  "private_key_path or private_key_source is required",
		},
		{
			name: "no patterns",
			app: GitHubApp{
				Name:           "test-app",
				AppID:          12345,
				InstallationID: 67890,
				PrivateKeyPath: "/tmp/key.pem",
				Patterns:       []string{},
				Priority:       100,
			},
			wantErr: true,
			errMsg:  "at least one pattern is required",
		},
		{
			name: "empty pattern",
			app: GitHubApp{
				Name:           "test-app",
				AppID:          12345,
				InstallationID: 67890,
				PrivateKeyPath: "/tmp/key.pem",
				Patterns:       []string{"github.com/org/*", "", "github.com/other/*"},
				Priority:       100,
			},
			wantErr: true,
			errMsg:  "patterns[1] cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.app.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("GitHubApp.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("GitHubApp.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Unable to get home directory")
	}

	tests := []struct {
		name     string
		path     string
		expected string
		wantErr  bool
	}{
		{
			name:     "absolute path",
			path:     "/tmp/key.pem",
			expected: "", // Will be platform-specific
			wantErr:  false,
		},
		{
			name:     "home directory only",
			path:     "~",
			expected: homeDir,
			wantErr:  false,
		},
		{
			name:     "home directory with path",
			path:     "~/.config/gh/key.pem",
			expected: filepath.Join(homeDir, ".config/gh/key.pem"),
			wantErr:  false,
		},
		{
			name:     "relative path",
			path:     "key.pem",
			expected: "", // Will be set to absolute path
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.name == "relative path" || tt.name == "absolute path" {
				// For relative and absolute paths, just check that result is absolute
				if !filepath.IsAbs(result) {
					t.Errorf("expandPath() = %v, expected absolute path", result)
				}
			} else if result != tt.expected {
				t.Errorf("expandPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_GetByPriority(t *testing.T) {
	config := Config{
		Version: "1.0",
		GitHubApps: []GitHubApp{
			{
				Name:     "low-priority",
				Priority: 10,
			},
			{
				Name:     "high-priority",
				Priority: 100,
			},
			{
				Name:     "medium-priority",
				Priority: 50,
			},
			{
				Name:     "same-priority-a",
				Priority: 25,
			},
			{
				Name:     "same-priority-b",
				Priority: 25,
			},
		},
	}

	sorted := config.GetByPriority()

	// Check that sorting is correct
	expectedOrder := []string{
		"high-priority",   // Priority 100
		"medium-priority", // Priority 50
		"same-priority-a", // Priority 25 (alphabetically first)
		"same-priority-b", // Priority 25 (alphabetically second)
		"low-priority",    // Priority 10
	}

	if len(sorted) != len(expectedOrder) {
		t.Fatalf("GetByPriority() returned %d apps, want %d", len(sorted), len(expectedOrder))
	}

	for i, expected := range expectedOrder {
		if sorted[i].Name != expected {
			t.Errorf("GetByPriority()[%d].Name = %v, want %v", i, sorted[i].Name, expected)
		}
	}

	// Verify original config is not modified
	if config.GitHubApps[0].Name != "low-priority" {
		t.Error("GetByPriority() modified the original config")
	}
}

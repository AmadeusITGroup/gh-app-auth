package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOrCreate(t *testing.T) {
	tests := []struct {
		name           string
		existingConfig string
		wantErr        bool
		wantAppsCount  int
	}{
		{
			name:           "create new config",
			existingConfig: "",
			wantErr:        false,
			wantAppsCount:  0,
		},
		{
			name: "load existing config",
			existingConfig: `version: "1.0"
github_apps:
  - name: "Test App"
    app_id: 123456
    installation_id: 67890
    patterns:
      - "github.com/test/*"
    priority: 5
    private_key_source: "keyring"
`,
			wantErr:       false,
			wantAppsCount: 1,
		},
		{
			name: "invalid YAML",
			existingConfig: `invalid: yaml: content:
  this is: [not, valid]
    malformed`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yml")

			// Write existing config if provided
			if tt.existingConfig != "" {
				if err := os.WriteFile(configPath, []byte(tt.existingConfig), 0600); err != nil {
					t.Fatalf("Failed to write test config: %v", err)
				}
			}

			t.Setenv("GH_APP_AUTH_CONFIG", configPath)

			cfg, err := LoadOrCreate()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if cfg == nil {
				t.Fatal("Expected config to be created")
			}

			if len(cfg.GitHubApps) != tt.wantAppsCount {
				t.Errorf("Apps count = %d, want %d", len(cfg.GitHubApps), tt.wantAppsCount)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("load existing file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yml")

		configContent := `version: "1.0"
github_apps:
  - name: "App1"
    app_id: 111111
    installation_id: 11111
    patterns:
      - "github.com/org1/*"
    priority: 5
    private_key_source: "keyring"
  - name: "App2"
    app_id: 222222
    installation_id: 22222
    patterns:
      - "github.com/org2/*"
    priority: 5
    private_key_source: "keyring"
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		t.Setenv("GH_APP_AUTH_CONFIG", configPath)

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(cfg.GitHubApps) != 2 {
			t.Errorf("Expected 2 apps, got %d", len(cfg.GitHubApps))
		}
	})

	t.Run("file not found", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "nonexistent.yml")
		t.Setenv("GH_APP_AUTH_CONFIG", configPath)

		_, err := Load()
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})
}

func TestSave(t *testing.T) {
	t.Run("save to new file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yml")

		cfg := &Config{
			Version: "1.0",
			GitHubApps: []GitHubApp{
				{
					Name:     "Test App",
					AppID:    123456,
					Patterns: []string{"github.com/test/*"},
				},
			},
		}

		t.Setenv("GH_APP_AUTH_CONFIG", configPath)

		err := cfg.Save()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}

		// Verify content
		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read config: %v", err)
		}

		if !strings.Contains(string(content), "Test App") {
			t.Error("Config does not contain expected content")
		}
	})
}

func TestAddOrUpdateApp(t *testing.T) {
	t.Run("add new app", func(t *testing.T) {
		cfg := &Config{
			Version:    "1.0",
			GitHubApps: []GitHubApp{},
		}

		newApp := &GitHubApp{
			Name:     "New App",
			AppID:    123456,
			Patterns: []string{"github.com/test/*"},
		}

		cfg.AddOrUpdateApp(newApp)

		if len(cfg.GitHubApps) != 1 {
			t.Errorf("Expected 1 app, got %d", len(cfg.GitHubApps))
		}

		if cfg.GitHubApps[0].Name != "New App" {
			t.Errorf("App name = %s, want %s", cfg.GitHubApps[0].Name, "New App")
		}
	})

	t.Run("update existing app by app_id", func(t *testing.T) {
		cfg := &Config{
			Version: "1.0",
			GitHubApps: []GitHubApp{
				{
					Name:     "Existing App",
					AppID:    123456,
					Patterns: []string{"github.com/old/*"},
				},
			},
		}

		updatedApp := &GitHubApp{
			Name:     "Updated App",
			AppID:    123456,
			Patterns: []string{"github.com/new/*"},
		}

		cfg.AddOrUpdateApp(updatedApp)

		if len(cfg.GitHubApps) != 1 {
			t.Errorf("Expected 1 app, got %d", len(cfg.GitHubApps))
		}

		if cfg.GitHubApps[0].Name != "Updated App" {
			t.Errorf("App name = %s, want %s", cfg.GitHubApps[0].Name, "Updated App")
		}

		if cfg.GitHubApps[0].Patterns[0] != "github.com/new/*" {
			t.Errorf("Pattern = %s, want %s", cfg.GitHubApps[0].Patterns[0], "github.com/new/*")
		}
	})
}

func TestRemoveApp(t *testing.T) {
	t.Run("remove by app_id", func(t *testing.T) {
		cfg := &Config{
			Version: "1.0",
			GitHubApps: []GitHubApp{
				{Name: "App1", AppID: 111111},
				{Name: "App2", AppID: 222222},
				{Name: "App3", AppID: 333333},
			},
		}

		removed := cfg.RemoveApp(222222)

		if !removed {
			t.Error("Expected app to be removed")
		}

		if len(cfg.GitHubApps) != 2 {
			t.Errorf("Expected 2 apps remaining, got %d", len(cfg.GitHubApps))
		}

		// Verify App2 is gone
		for _, app := range cfg.GitHubApps {
			if app.AppID == 222222 {
				t.Error("App2 should have been removed")
			}
		}
	})

	t.Run("remove non-existent app", func(t *testing.T) {
		cfg := &Config{
			Version: "1.0",
			GitHubApps: []GitHubApp{
				{Name: "App1", AppID: 111111},
			},
		}

		removed := cfg.RemoveApp(999999)

		if removed {
			t.Error("Should not have removed any app")
		}

		if len(cfg.GitHubApps) != 1 {
			t.Error("App count should not have changed")
		}
	})
}

func TestGetApp(t *testing.T) {
	cfg := &Config{
		Version: "1.0",
		GitHubApps: []GitHubApp{
			{Name: "App1", AppID: 111111},
			{Name: "App2", AppID: 222222},
		},
	}

	t.Run("get existing app", func(t *testing.T) {
		app, err := cfg.GetApp(111111)
		if err != nil {
			t.Fatalf("Expected to find app: %v", err)
		}

		if app.Name != "App1" {
			t.Errorf("App name = %s, want %s", app.Name, "App1")
		}
	})

	t.Run("get non-existent app", func(t *testing.T) {
		_, err := cfg.GetApp(999999)
		if err == nil {
			t.Error("Should not have found app")
		}
	})
}

func TestOutputJSON(t *testing.T) {
	apps := []GitHubApp{
		{
			Name:     "Test App",
			AppID:    123456,
			Patterns: []string{"github.com/test/*"},
		},
	}

	var buf bytes.Buffer
	err := OutputJSON(&buf, apps)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	json := buf.String()
	if json == "" {
		t.Error("Expected JSON output")
	}

	// Verify it contains expected content
	if !strings.Contains(json, "Test App") {
		t.Error("JSON does not contain app name")
	}

	if !strings.Contains(json, "123456") {
		t.Error("JSON does not contain app ID")
	}
}

func TestOutputYAML(t *testing.T) {
	apps := []GitHubApp{
		{
			Name:     "Test App",
			AppID:    123456,
			Patterns: []string{"github.com/test/*"},
		},
	}

	var buf bytes.Buffer
	err := OutputYAML(&buf, apps)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	yaml := buf.String()
	if yaml == "" {
		t.Error("Expected YAML output")
	}

	// Verify it contains expected content
	if !strings.Contains(yaml, "Test App") {
		t.Error("YAML does not contain app name")
	}

	if !strings.Contains(yaml, "123456") {
		t.Error("YAML does not contain app ID")
	}
}

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"gopkg.in/yaml.v3"
)

func TestSelectDebugApps(t *testing.T) {
	cfg := &config.Config{
		Version: "1.0",
		GitHubApps: []config.GitHubApp{
			{Name: "App1", AppID: 111111},
			{Name: "App2", AppID: 222222, ClientID: "Iv1.ClientTwo"},
			{Name: "App3", AppID: 333333},
		},
	}

	tests := []struct {
		name        string
		appID       int64
		clientID    string
		appIDSet    bool
		clientIDSet bool
		wantCount   int
		wantName    string
		wantErr     bool
	}{
		{
			name:      "all apps when no filter",
			wantCount: 3,
		},
		{
			name:      "filter by app id",
			appID:     111111,
			appIDSet:  true,
			wantCount: 1,
			wantName:  "App1",
		},
		{
			name:        "filter by client id",
			clientID:    "Iv1.ClientTwo",
			clientIDSet: true,
			wantCount:   1,
			wantName:    "App2",
		},
		{
			name:      "app id not found",
			appID:     999999,
			appIDSet:  true,
			wantCount: 0,
			wantErr:   true,
		},
		{
			name:        "client id not found",
			clientID:    "Iv1.NotFound",
			clientIDSet: true,
			wantCount:   0,
			wantErr:     true,
		},
		{
			name:        "both flags set",
			appID:       111111,
			appIDSet:    true,
			clientID:    "Iv1.ClientTwo",
			clientIDSet: true,
			wantCount:   0,
			wantErr:     true,
		},
		{
			name:        "empty client id",
			clientIDSet: true,
			wantCount:   0,
			wantErr:     true,
		},
		{
			name:      "negative app id",
			appID:     -1,
			appIDSet:  true,
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps, err := selectDebugApps(cfg, tt.appID, tt.clientID, tt.appIDSet, tt.clientIDSet)

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

			if len(apps) != tt.wantCount {
				t.Errorf("Got %d apps, want %d", len(apps), tt.wantCount)
			}

			if tt.wantCount == 1 && apps[0].Name != tt.wantName {
				t.Errorf("App name = %q, want %q", apps[0].Name, tt.wantName)
			}
		})
	}
}

func TestNewDebugCmd_Flags(t *testing.T) {
	listInstallations := newListInstallationsCmd()
	if listInstallations.Flags().Lookup("client-id") == nil {
		t.Error("list-installations should define --client-id flag")
	}
	if listInstallations.Flags().Lookup("app-id") == nil {
		t.Error("list-installations should define --app-id flag")
	}

	listRepos := newListInstallationReposCmd()
	if listRepos.Flags().Lookup("client-id") == nil {
		t.Error("list-repositories should define --client-id flag")
	}
	if listRepos.Flags().Lookup("app-id") == nil {
		t.Error("list-repositories should define --app-id flag")
	}
}

func writeDebugTestConfig(t *testing.T, cfg *config.Config) string {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	return configPath
}

func debugTestApp(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test-key.pem")
	testKey := generateTestRSAKey(t)
	if err := os.WriteFile(keyPath, []byte(testKey), 0600); err != nil {
		t.Fatalf("Failed to write test key: %v", err)
	}

	cfg := &config.Config{
		Version: "1.0",
		GitHubApps: []config.GitHubApp{
			{
				Name:             "Debug Client App",
				ClientID:         "Iv1.DebugClient",
				InstallationID:   789012,
				Patterns:         []string{"github.com/org/*"},
				PrivateKeySource: config.PrivateKeySourceFilesystem,
				PrivateKeyPath:   keyPath,
			},
		},
	}

	return writeDebugTestConfig(t, cfg)
}

func TestDebugListInstallationsCmd_RunE(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("no apps configured", func(t *testing.T) {
		cfg := &config.Config{
			Version: "1.0",
			PATs: []config.PersonalAccessToken{
				{Name: "Test PAT", Patterns: []string{"github.com/test/*"}},
			},
		}
		configPath := writeDebugTestConfig(t, cfg)
		t.Setenv("GH_APP_AUTH_CONFIG", configPath)

		cmd := newListInstallationsCmd()
		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)

		err := cmd.Execute()
		if err != nil {
			t.Errorf("Command failed: %v", err)
		}
	})

	t.Run("client id not found", func(t *testing.T) {
		configPath := debugTestApp(t)
		t.Setenv("GH_APP_AUTH_CONFIG", configPath)

		cmd := newListInstallationsCmd()
		cmd.SetArgs([]string{"--client-id", "Iv1.NotFound"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for non-existent client ID")
		}
	})

	t.Run("client id filter hits and fails network", func(t *testing.T) {
		configPath := debugTestApp(t)
		t.Setenv("GH_APP_AUTH_CONFIG", configPath)

		cmd := newListInstallationsCmd()
		cmd.SetArgs([]string{"--client-id", "Iv1.DebugClient"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected network error when listing installations")
		}
	})

	t.Run("no filter iterates apps and fails network", func(t *testing.T) {
		configPath := debugTestApp(t)
		t.Setenv("GH_APP_AUTH_CONFIG", configPath)

		cmd := newListInstallationsCmd()
		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)

		err := cmd.Execute()
		if err != nil {
			t.Errorf("Unfiltered command should continue on network errors: %v", err)
		}
	})
}

func TestDebugListRepositoriesCmd_RunE(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("client id filter hits and fails network", func(t *testing.T) {
		configPath := debugTestApp(t)
		t.Setenv("GH_APP_AUTH_CONFIG", configPath)

		cmd := newListInstallationReposCmd()
		cmd.SetArgs([]string{"--client-id", "Iv1.DebugClient"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected network error when listing repositories")
		}
	})

	t.Run("no filter iterates apps and fails network", func(t *testing.T) {
		configPath := debugTestApp(t)
		t.Setenv("GH_APP_AUTH_CONFIG", configPath)

		cmd := newListInstallationReposCmd()
		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)

		err := cmd.Execute()
		if err != nil {
			t.Errorf("Unfiltered command should continue on network errors: %v", err)
		}
	})
}

func TestAppDisplayName(t *testing.T) {
	tests := []struct {
		name string
		app  config.GitHubApp
		want string
	}{
		{
			name: "with name",
			app:  config.GitHubApp{Name: "My App", AppID: 123456},
			want: "My App",
		},
		{
			name: "client id fallback",
			app:  config.GitHubApp{ClientID: "Iv1.Fallback"},
			want: "GitHub App Iv1.Fallback",
		},
		{
			name: "app id fallback",
			app:  config.GitHubApp{AppID: 123456},
			want: "GitHub App 123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appDisplayName(&tt.app); got != tt.want {
				t.Errorf("appDisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractHostFromPattern(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		{"github.com/org/*", "github.com"},
		{"https://github.example.com/org/repo", "github.example.com"},
		{"  http://host/path  ", "host"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			if got := extractHostFromPattern(tt.pattern); got != tt.want {
				t.Errorf("extractHostFromPattern(%q) = %q, want %q", tt.pattern, got, tt.want)
			}
		})
	}
}

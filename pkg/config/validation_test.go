package config

import (
	"testing"
)

func TestConfig_Validate_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "missing version",
			cfg: &Config{
				GitHubApps: []GitHubApp{
					{
						Name:  "Test",
						AppID: 123,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty apps",
			cfg: &Config{
				Version:    "1",
				GitHubApps: []GitHubApp{},
			},
			wantErr: true,
		},
		{
			name: "nil apps",
			cfg: &Config{
				Version:    "1",
				GitHubApps: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGitHubApp_Validate_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		app     GitHubApp
		wantErr bool
	}{
		{
			name: "empty name",
			app: GitHubApp{
				Name:           "",
				AppID:          123,
				InstallationID: 456,
				Patterns:       []string{"github.com/org/*"},
			},
			wantErr: true,
		},
		{
			name: "zero app ID",
			app: GitHubApp{
				Name:           "Test",
				AppID:          0,
				InstallationID: 456,
				Patterns:       []string{"github.com/org/*"},
			},
			wantErr: true,
		},
		{
			name: "negative app ID",
			app: GitHubApp{
				Name:           "Test",
				AppID:          -1,
				InstallationID: 456,
				Patterns:       []string{"github.com/org/*"},
			},
			wantErr: true,
		},
		{
			name: "empty patterns",
			app: GitHubApp{
				Name:           "Test",
				AppID:          123,
				InstallationID: 456,
				Patterns:       []string{},
			},
			wantErr: true,
		},
		{
			name: "nil patterns",
			app: GitHubApp{
				Name:           "Test",
				AppID:          123,
				InstallationID: 456,
				Patterns:       nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.app.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_AddOrUpdateApp(t *testing.T) {
	cfg := &Config{
		Version:    "1",
		GitHubApps: []GitHubApp{},
	}

	app1 := &GitHubApp{
		Name:           "App1",
		AppID:          111,
		InstallationID: 222,
		Patterns:       []string{"github.com/org1/*"},
	}

	// Add new app
	cfg.AddOrUpdateApp(app1)
	if len(cfg.GitHubApps) != 1 {
		t.Errorf("Expected 1 app after add, got %d", len(cfg.GitHubApps))
	}

	// Update existing app
	app1.Priority = 10
	cfg.AddOrUpdateApp(app1)
	if len(cfg.GitHubApps) != 1 {
		t.Errorf("Expected 1 app after update, got %d", len(cfg.GitHubApps))
	}
	if cfg.GitHubApps[0].Priority != 10 {
		t.Errorf("Expected priority 10, got %d", cfg.GitHubApps[0].Priority)
	}

	// Add different app
	app2 := &GitHubApp{
		Name:           "App2",
		AppID:          333,
		InstallationID: 444,
		Patterns:       []string{"github.com/org2/*"},
	}
	cfg.AddOrUpdateApp(app2)
	if len(cfg.GitHubApps) != 2 {
		t.Errorf("Expected 2 apps, got %d", len(cfg.GitHubApps))
	}
}

func TestConfig_RemoveApp(t *testing.T) {
	cfg := &Config{
		Version: "1",
		GitHubApps: []GitHubApp{
			{Name: "App1", AppID: 111},
			{Name: "App2", AppID: 222},
			{Name: "App3", AppID: 333},
		},
	}

	// Remove middle app by ID
	removed := cfg.RemoveApp(222)
	if !removed {
		t.Error("Expected RemoveApp to return true")
	}
	if len(cfg.GitHubApps) != 2 {
		t.Errorf("Expected 2 apps after remove, got %d", len(cfg.GitHubApps))
	}

	// Verify correct app was removed
	for _, app := range cfg.GitHubApps {
		if app.AppID == 222 {
			t.Error("App with ID 222 should have been removed")
		}
	}

	// Try to remove non-existent app
	removed = cfg.RemoveApp(999)
	if removed {
		t.Error("Expected RemoveApp to return false for non-existent app")
	}
	if len(cfg.GitHubApps) != 2 {
		t.Errorf("Expected 2 apps after failed remove, got %d", len(cfg.GitHubApps))
	}
}

func TestConfig_GetAppCount(t *testing.T) {
	cfg := &Config{
		Version: "1",
		GitHubApps: []GitHubApp{
			{Name: "App1", AppID: 111},
			{Name: "App2", AppID: 222},
		},
	}

	count := len(cfg.GitHubApps)
	if count != 2 {
		t.Errorf("Expected 2 apps, got %d", count)
	}

	// Empty config
	emptyConfig := &Config{Version: "1"}
	count = len(emptyConfig.GitHubApps)
	if count != 0 {
		t.Errorf("Expected 0 apps in empty config, got %d", count)
	}
}

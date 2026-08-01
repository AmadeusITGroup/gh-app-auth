package cmd

import (
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

func TestFindApp(t *testing.T) {
	cfg := &config.Config{
		Version: "1.0",
		GitHubApps: []config.GitHubApp{
			{Name: "App1", AppID: 111111},
			{Name: "App2", AppID: 222222, ClientID: "Iv1.ClientTwo"},
			{Name: "App3", AppID: 333333},
		},
	}

	tests := []struct {
		name      string
		appID     int64
		clientID  string
		wantIndex int
		wantName  string
		wantErr   bool
	}{
		{
			name:      "find first app by id",
			appID:     111111,
			wantIndex: 0,
			wantName:  "App1",
			wantErr:   false,
		},
		{
			name:      "find middle app by id",
			appID:     222222,
			wantIndex: 1,
			wantName:  "App2",
			wantErr:   false,
		},
		{
			name:      "find middle app by client id",
			clientID:  "Iv1.ClientTwo",
			wantIndex: 1,
			wantName:  "App2",
			wantErr:   false,
		},
		{
			name:      "find last app",
			appID:     333333,
			wantIndex: 2,
			wantName:  "App3",
			wantErr:   false,
		},
		{
			name:      "app not found by id",
			appID:     999999,
			wantIndex: -1,
			wantErr:   true,
		},
		{
			name:      "app not found by client id",
			clientID:  "Iv1.NotFound",
			wantIndex: -1,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index, app, err := findApp(cfg, tt.appID, tt.clientID)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if index != -1 {
					t.Errorf("Index = %d, want -1", index)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if index != tt.wantIndex {
				t.Errorf("Index = %d, want %d", index, tt.wantIndex)
			}

			if app.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", app.Name, tt.wantName)
			}

			if tt.clientID != "" {
				if app.ClientID != tt.clientID {
					t.Errorf("ClientID = %q, want %q", app.ClientID, tt.clientID)
				}
			} else if app.AppID != tt.appID {
				t.Errorf("AppID = %d, want %d", app.AppID, tt.appID)
			}
		})
	}
}

func TestClearCachedTokens(t *testing.T) {
	// Test placeholder implementation
	err := clearCachedTokens("123456")
	if err != nil {
		t.Errorf("clearCachedTokens() error = %v", err)
	}
}

func TestClearAllCachedTokens(t *testing.T) {
	// Test placeholder implementation
	err := clearAllCachedTokens()
	if err != nil {
		t.Errorf("clearAllCachedTokens() error = %v", err)
	}
}

func TestDisplayAllAppsRemovalSuccess(t *testing.T) {
	// This outputs to stdout - just verify it doesn't panic
	displayAllAppsRemovalSuccess(3)
	displayAllAppsRemovalSuccess(0)
	displayAllAppsRemovalSuccess(1)
}

func TestConfirmAppRemoval(t *testing.T) {
	// Skip interactive test - would need stdin mocking
	t.Skip("Interactive function - requires stdin mocking")
}

func TestConfirmAllAppsRemoval(t *testing.T) {
	// Skip interactive test - would need stdin mocking
	t.Skip("Interactive function - requires stdin mocking")
}

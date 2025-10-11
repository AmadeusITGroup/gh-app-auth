package cmd

import (
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

func TestFindAppByID(t *testing.T) {
	cfg := &config.Config{
		Version: "1.0",
		GitHubApps: []config.GitHubApp{
			{Name: "App1", AppID: 111111},
			{Name: "App2", AppID: 222222},
			{Name: "App3", AppID: 333333},
		},
	}

	tests := []struct {
		name      string
		appID     int64
		wantIndex int
		wantName  string
		wantErr   bool
	}{
		{
			name:      "find first app",
			appID:     111111,
			wantIndex: 0,
			wantName:  "App1",
			wantErr:   false,
		},
		{
			name:      "find middle app",
			appID:     222222,
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
			name:      "app not found",
			appID:     999999,
			wantIndex: -1,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index, app, err := findAppByID(cfg, tt.appID)

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

			if app.AppID != tt.appID {
				t.Errorf("AppID = %d, want %d", app.AppID, tt.appID)
			}
		})
	}
}

func TestClearCachedTokens(t *testing.T) {
	// Test placeholder implementation
	err := clearCachedTokens(123456)
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

package cmd

import (
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
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

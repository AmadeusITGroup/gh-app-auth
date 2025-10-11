package cmd

import (
	"testing"
	"time"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

func TestDisplayScope(t *testing.T) {
	t.Run("nil scope", func(t *testing.T) {
		app := &config.GitHubApp{
			Name:           "Test App",
			AppID:          123,
			InstallationID: 456,
			Scope:          nil,
		}

		// Should not panic
		displayScope(app)
	})

	t.Run("scope with all repositories", func(t *testing.T) {
		app := &config.GitHubApp{
			Name:           "Org App",
			AppID:          123,
			InstallationID: 456,
			Scope: &config.InstallationScope{
				RepositorySelection: "all",
				AccountLogin:        "testorg",
				AccountType:         "Organization",
				LastFetched:         time.Now(),
				CacheExpiry:         time.Now().Add(24 * time.Hour),
			},
		}

		// Should not panic
		displayScope(app)
	})

	t.Run("scope with selected repositories", func(t *testing.T) {
		app := &config.GitHubApp{
			Name:           "Selected App",
			AppID:          123,
			InstallationID: 456,
			Scope: &config.InstallationScope{
				RepositorySelection: "selected",
				AccountLogin:        "testorg",
				AccountType:         "Organization",
				Repositories: []config.RepositoryInfo{
					{FullName: "testorg/repo1", Private: true},
					{FullName: "testorg/repo2", Private: false},
				},
				LastFetched: time.Now(),
				CacheExpiry: time.Now().Add(24 * time.Hour),
			},
		}

		// Should not panic
		displayScope(app)
	})

	t.Run("expired scope", func(t *testing.T) {
		app := &config.GitHubApp{
			Name:           "Expired App",
			AppID:          123,
			InstallationID: 456,
			Scope: &config.InstallationScope{
				RepositorySelection: "all",
				AccountLogin:        "testorg",
				AccountType:         "Organization",
				LastFetched:         time.Now().Add(-2 * time.Hour),
				CacheExpiry:         time.Now().Add(-1 * time.Hour), // Expired
			},
		}

		// Should not panic and should show expiry warning
		displayScope(app)
	})

	t.Run("empty repository list", func(t *testing.T) {
		app := &config.GitHubApp{
			Name:           "Empty List App",
			AppID:          123,
			InstallationID: 456,
			Scope: &config.InstallationScope{
				RepositorySelection: "selected",
				AccountLogin:        "testorg",
				AccountType:         "Organization",
				Repositories:        []config.RepositoryInfo{}, // Empty
				LastFetched:         time.Now(),
				CacheExpiry:         time.Now().Add(24 * time.Hour),
			},
		}

		// Should not panic
		displayScope(app)
	})
}

func TestNewScopeCmd(t *testing.T) {
	cmd := NewScopeCmd()

	if cmd == nil {
		t.Fatal("NewScopeCmd() returned nil")
	}

	if cmd.Use != "scope" {
		t.Errorf("Expected Use to be 'scope', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	// Check that the --refresh flag exists
	refreshFlag := cmd.Flags().Lookup("refresh")
	if refreshFlag == nil {
		t.Error("Expected --refresh flag to exist")
		return
	}

	if refreshFlag.DefValue != "false" {
		t.Errorf("Expected --refresh default to be false, got %q", refreshFlag.DefValue)
	}
}

func TestScopeCmd_FlagValidation(t *testing.T) {
	cmd := NewScopeCmd()

	// Test that command accepts valid flags
	tests := []struct {
		name string
		args []string
		want bool // whether command should accept args
	}{
		{
			name: "no flags",
			args: []string{},
			want: true,
		},
		{
			name: "refresh flag",
			args: []string{"--refresh"},
			want: true,
		},
		{
			name: "refresh=true",
			args: []string{"--refresh=true"},
			want: true,
		},
		{
			name: "refresh=false",
			args: []string{"--refresh=false"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd.SetArgs(tt.args)
			err := cmd.ParseFlags(tt.args)
			if (err == nil) != tt.want {
				t.Errorf("ParseFlags() error = %v, want success = %v", err, tt.want)
			}
		})
	}
}

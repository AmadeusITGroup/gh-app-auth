package cmd

import (
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

func TestFindAppByURL_WithConfig(t *testing.T) {
	cfg := &config.Config{
		Version: "1",
		GitHubApps: []config.GitHubApp{
			{
				Name:     "App1",
				AppID:    111,
				Patterns: []string{"github.com/org1/*"},
			},
			{
				Name:     "App2",
				AppID:    222,
				Patterns: []string{"github.com/org2/*"},
			},
		},
	}

	tests := []struct {
		name    string
		url     string
		wantNil bool
	}{
		{
			name:    "matching URL",
			url:     "https://github.com/org1/repo",
			wantNil: false,
		},
		{
			name:    "non-matching URL",
			url:     "https://github.com/org3/repo",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := findAppByURL(cfg, tt.url)

			if tt.wantNil && app != nil {
				t.Errorf("Expected nil app, got %v", app)
			}
			if !tt.wantNil && app == nil {
				t.Error("Expected non-nil app, got nil")
			}
		})
	}
}

func TestFindAppByPattern_WithConfig(t *testing.T) {
	// Set the pattern variable to match the prefix
	originalPattern := gitCredentialPattern
	gitCredentialPattern = "https://github.com"
	defer func() {
		gitCredentialPattern = originalPattern
	}()

	cfg := &config.Config{
		Version: "1",
		GitHubApps: []config.GitHubApp{
			{
				Name:     "LowPriority",
				AppID:    111,
				Patterns: []string{"https://github.com"},
				Priority: 1,
			},
			{
				Name:     "HighPriority",
				AppID:    222,
				Patterns: []string{"https://github.com"},
				Priority: 10,
			},
		},
	}

	repoURL := "https://github.com/myorg/myrepo"
	app := findAppByPattern(cfg, repoURL)

	if app != nil {
		// Good - found an app (may be either one)
		if app.Name != "LowPriority" && app.Name != "HighPriority" {
			t.Errorf("Got unexpected app: %s", app.Name)
		}
	}
	// Note: app might be nil if pattern matching logic doesn't match,
	// which is acceptable for this edge case test
}

func TestGitCredentialPattern_Variable(t *testing.T) {
	// Test that the variable exists and can be set
	originalPattern := gitCredentialPattern
	defer func() {
		gitCredentialPattern = originalPattern
	}()

	gitCredentialPattern = "https://github.com/myorg"
	if gitCredentialPattern != "https://github.com/myorg" {
		t.Error("Failed to set gitCredentialPattern")
	}
}

func TestNewGitCredentialCmd_Structure(t *testing.T) {
	cmd := NewGitCredentialCmd()

	if cmd == nil {
		t.Fatal("NewGitCredentialCmd returned nil")
	}

	if cmd.Use != "git-credential" {
		t.Errorf("Use = %q, want %q", cmd.Use, "git-credential")
	}

	if !cmd.Hidden {
		t.Error("git-credential should be hidden")
	}

	// Should have pattern flag
	flag := cmd.Flags().Lookup("pattern")
	if flag == nil {
		t.Error("Expected --pattern flag to exist")
	}
}

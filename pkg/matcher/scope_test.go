package matcher

import (
	"testing"
	"time"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

func TestIsInScope_All(t *testing.T) {
	scope := &config.InstallationScope{
		RepositorySelection: "all",
		AccountLogin:        "myorg",
		AccountType:         "Organization",
	}

	tests := []struct {
		name     string
		repoPath string
		want     bool
	}{
		{
			name:     "repo in org",
			repoPath: "github.com/myorg/repo1",
			want:     true,
		},
		{
			name:     "another repo in org",
			repoPath: "github.com/myorg/another-repo",
			want:     true,
		},
		{
			name:     "repo in different org",
			repoPath: "github.com/otherorg/repo",
			want:     false,
		},
		{
			name:     "repo with subgroups",
			repoPath: "github.com/myorg/subgroup/repo",
			want:     true,
		},
		{
			name:     "invalid path - too short",
			repoPath: "github.com",
			want:     false,
		},
		{
			name:     "invalid path - no owner",
			repoPath: "github.com/",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInScope(tt.repoPath, scope)
			if got != tt.want {
				t.Errorf("isInScope(%q) = %v, want %v", tt.repoPath, got, tt.want)
			}
		})
	}
}

func TestIsInScope_Selected(t *testing.T) {
	scope := &config.InstallationScope{
		RepositorySelection: "selected",
		AccountLogin:        "myorg",
		AccountType:         "Organization",
		Repositories: []config.RepositoryInfo{
			{FullName: "myorg/repo1", Private: true},
			{FullName: "myorg/repo2", Private: false},
			{FullName: "myorg/special-repo", Private: true},
		},
	}

	tests := []struct {
		name     string
		repoPath string
		want     bool
	}{
		{
			name:     "repo1 in selected list",
			repoPath: "github.com/myorg/repo1",
			want:     true,
		},
		{
			name:     "repo2 in selected list",
			repoPath: "github.com/myorg/repo2",
			want:     true,
		},
		{
			name:     "special-repo in selected list",
			repoPath: "github.com/myorg/special-repo",
			want:     true,
		},
		{
			name:     "repo3 not in selected list",
			repoPath: "github.com/myorg/repo3",
			want:     false,
		},
		{
			name:     "repo in different org",
			repoPath: "github.com/otherorg/repo1",
			want:     false,
		},
		{
			name:     "invalid path - too short",
			repoPath: "github.com/myorg",
			want:     false,
		},
		{
			name:     "invalid path - no repo",
			repoPath: "github.com/myorg/",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInScope(tt.repoPath, scope)
			if got != tt.want {
				t.Errorf("isInScope(%q) = %v, want %v", tt.repoPath, got, tt.want)
			}
		})
	}
}

func TestIsInScope_EmptyRepositoryList(t *testing.T) {
	scope := &config.InstallationScope{
		RepositorySelection: "selected",
		AccountLogin:        "myorg",
		AccountType:         "Organization",
		Repositories:        []config.RepositoryInfo{}, // Empty list
	}

	got := isInScope("github.com/myorg/repo1", scope)
	if got {
		t.Error("isInScope should return false for empty repository list")
	}
}

func TestMatcher_WithScope(t *testing.T) {
	// Test that matcher respects scope validation
	apps := []config.GitHubApp{
		{
			Name:     "app-with-scope",
			AppID:    1,
			Patterns: []string{"github.com/myorg"},
			Scope: &config.InstallationScope{
				RepositorySelection: "selected",
				AccountLogin:        "myorg",
				AccountType:         "Organization",
				Repositories: []config.RepositoryInfo{
					{FullName: "myorg/allowed-repo", Private: false},
				},
				LastFetched: time.Now(),
				CacheExpiry: time.Now().Add(24 * time.Hour),
			},
		},
		{
			Name:     "app-without-scope",
			AppID:    2,
			Patterns: []string{"github.com"},
		},
	}

	matcher := NewMatcher(apps)

	tests := []struct {
		name        string
		repoURL     string
		wantAppName string
		wantNoMatch bool
	}{
		{
			name:        "allowed repo matches app with scope",
			repoURL:     "https://github.com/myorg/allowed-repo",
			wantAppName: "app-with-scope",
		},
		{
			name:        "disallowed repo falls back to app without scope",
			repoURL:     "https://github.com/myorg/other-repo",
			wantAppName: "app-without-scope",
		},
		{
			name:        "different org uses fallback app",
			repoURL:     "https://github.com/otherorg/repo",
			wantAppName: "app-without-scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := matcher.Match(tt.repoURL)
			if err != nil {
				t.Fatalf("Match() error = %v", err)
			}

			if tt.wantNoMatch {
				if app != nil {
					t.Errorf("Expected no match, got app %q", app.Name)
				}
				return
			}

			if app == nil {
				t.Fatal("Expected app, got nil")
			}

			if app.Name != tt.wantAppName {
				t.Errorf("Match() app name = %q, want %q", app.Name, tt.wantAppName)
			}
		})
	}
}

func TestMatcher_ScopeValidation_PrefersLongerPrefix(t *testing.T) {
	// Test that even with scope, longest prefix still wins
	apps := []config.GitHubApp{
		{
			Name:     "specific-repo-app",
			AppID:    1,
			Patterns: []string{"github.com/myorg/specific-repo"},
			Scope: &config.InstallationScope{
				RepositorySelection: "all",
				AccountLogin:        "myorg",
				AccountType:         "Organization",
				LastFetched:         time.Now(),
				CacheExpiry:         time.Now().Add(24 * time.Hour),
			},
		},
		{
			Name:     "org-app",
			AppID:    2,
			Patterns: []string{"github.com/myorg"},
			Scope: &config.InstallationScope{
				RepositorySelection: "all",
				AccountLogin:        "myorg",
				AccountType:         "Organization",
				LastFetched:         time.Now(),
				CacheExpiry:         time.Now().Add(24 * time.Hour),
			},
		},
	}

	matcher := NewMatcher(apps)

	// Specific repo should match the more specific app
	app, err := matcher.Match("https://github.com/myorg/specific-repo")
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}

	if app == nil {
		t.Fatal("Expected app, got nil")
	}

	if app.Name != "specific-repo-app" {
		t.Errorf("Expected 'specific-repo-app' (longer prefix), got %q", app.Name)
	}

	// Other repo should match the org app
	app, err = matcher.Match("https://github.com/myorg/other-repo")
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}

	if app == nil {
		t.Fatal("Expected app, got nil")
	}

	if app.Name != "org-app" {
		t.Errorf("Expected 'org-app', got %q", app.Name)
	}
}

func TestMatcher_ScopeValidation_SkipsOutOfScope(t *testing.T) {
	// Test that apps with scope that don't match are skipped
	apps := []config.GitHubApp{
		{
			Name:     "org1-app",
			AppID:    1,
			Patterns: []string{"github.com"},
			Scope: &config.InstallationScope{
				RepositorySelection: "all",
				AccountLogin:        "org1",
				AccountType:         "Organization",
				LastFetched:         time.Now(),
				CacheExpiry:         time.Now().Add(24 * time.Hour),
			},
		},
		{
			Name:     "org2-app",
			AppID:    2,
			Patterns: []string{"github.com"},
			Scope: &config.InstallationScope{
				RepositorySelection: "all",
				AccountLogin:        "org2",
				AccountType:         "Organization",
				LastFetched:         time.Now(),
				CacheExpiry:         time.Now().Add(24 * time.Hour),
			},
		},
	}

	matcher := NewMatcher(apps)

	// Should match org1-app
	app, err := matcher.Match("https://github.com/org1/repo")
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}
	if app == nil {
		t.Fatal("Expected app, got nil")
	}
	if app.Name != "org1-app" {
		t.Errorf("Expected 'org1-app', got %q", app.Name)
	}

	// Should match org2-app
	app, err = matcher.Match("https://github.com/org2/repo")
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}
	if app == nil {
		t.Fatal("Expected app, got nil")
	}
	if app.Name != "org2-app" {
		t.Errorf("Expected 'org2-app', got %q", app.Name)
	}

	// Should not match any app (org3 not in scope of either)
	app, err = matcher.Match("https://github.com/org3/repo")
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}
	if app != nil {
		t.Errorf("Expected no match for org3, got app %q", app.Name)
	}
}

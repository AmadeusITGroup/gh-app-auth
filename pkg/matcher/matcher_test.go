package matcher

import (
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

func TestParseRepositoryURL(t *testing.T) {
	tests := []struct {
		name         string
		repoURL      string
		wantHost     string
		wantOwner    string
		wantRepo     string
		wantFullPath string
		wantErr      bool
	}{
		{
			name:         "HTTPS URL",
			repoURL:      "https://github.com/owner/repo",
			wantHost:     "github.com",
			wantOwner:    "owner",
			wantRepo:     "repo",
			wantFullPath: "github.com/owner/repo",
			wantErr:      false,
		},
		{
			name:         "HTTPS URL with .git",
			repoURL:      "https://github.com/owner/repo.git",
			wantHost:     "github.com",
			wantOwner:    "owner",
			wantRepo:     "repo",
			wantFullPath: "github.com/owner/repo",
			wantErr:      false,
		},
		{
			name:         "SSH URL",
			repoURL:      "git@github.com:owner/repo.git",
			wantHost:     "github.com",
			wantOwner:    "owner",
			wantRepo:     "repo",
			wantFullPath: "github.com/owner/repo",
			wantErr:      false,
		},
		{
			name:         "Enterprise GitHub",
			repoURL:      "https://github.enterprise.com/org/project",
			wantHost:     "github.enterprise.com",
			wantOwner:    "org",
			wantRepo:     "project",
			wantFullPath: "github.enterprise.com/org/project",
			wantErr:      false,
		},
		{
			name:         "GitLab URL",
			repoURL:      "https://gitlab.com/group/project",
			wantHost:     "gitlab.com",
			wantOwner:    "group",
			wantRepo:     "project",
			wantFullPath: "gitlab.com/group/project",
			wantErr:      false,
		},
		{
			name:         "URL with subgroups",
			repoURL:      "https://gitlab.com/group/subgroup/project",
			wantHost:     "gitlab.com",
			wantOwner:    "group",
			wantRepo:     "subgroup/project",
			wantFullPath: "gitlab.com/group/subgroup/project",
			wantErr:      false,
		},
		{
			name:         "No scheme - assume GitHub",
			repoURL:      "github.com/owner/repo",
			wantHost:     "github.com",
			wantOwner:    "owner",
			wantRepo:     "repo",
			wantFullPath: "github.com/owner/repo",
			wantErr:      false,
		},
		{
			name:    "Empty URL",
			repoURL: "",
			wantErr: true,
		},
		{
			name:    "Invalid URL format",
			repoURL: "not-a-url",
			wantErr: true,
		},
		{
			name:    "URL with no path",
			repoURL: "https://github.com",
			wantErr: true,
		},
		{
			name:    "URL with insufficient path components",
			repoURL: "https://github.com/owner",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoInfo, err := parseRepositoryURL(tt.repoURL)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseRepositoryURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if repoInfo.Host != tt.wantHost {
				t.Errorf("parseRepositoryURL() Host = %v, want %v", repoInfo.Host, tt.wantHost)
			}
			if repoInfo.Owner != tt.wantOwner {
				t.Errorf("parseRepositoryURL() Owner = %v, want %v", repoInfo.Owner, tt.wantOwner)
			}
			if repoInfo.Repository != tt.wantRepo {
				t.Errorf("parseRepositoryURL() Repository = %v, want %v", repoInfo.Repository, tt.wantRepo)
			}
			if repoInfo.FullPath != tt.wantFullPath {
				t.Errorf("parseRepositoryURL() FullPath = %v, want %v", repoInfo.FullPath, tt.wantFullPath)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name    string
		repoURL string
		want    string
	}{
		{
			name:    "SSH URL",
			repoURL: "git@github.com:owner/repo.git",
			want:    "https://github.com/owner/repo.git",
		},
		{
			name:    "HTTPS URL unchanged",
			repoURL: "https://github.com/owner/repo",
			want:    "https://github.com/owner/repo",
		},
		{
			name:    "HTTP URL unchanged",
			repoURL: "http://github.com/owner/repo",
			want:    "http://github.com/owner/repo",
		},
		{
			name:    "No scheme - add https",
			repoURL: "github.com/owner/repo",
			want:    "https://github.com/owner/repo",
		},
		{
			name:    "Simple name unchanged",
			repoURL: "repo",
			want:    "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeURL(tt.repoURL); got != tt.want {
				t.Errorf("normalizeURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatcher_Match(t *testing.T) {
	apps := []config.GitHubApp{
		{
			Name:     "corporate-app",
			AppID:    1,
			Patterns: []string{"github.com/myorg", "github.enterprise.com"},
		},
		{
			Name:     "gitlab-app",
			AppID:    2,
			Patterns: []string{"gitlab.com"},
		},
		{
			Name:     "specific-repo-app",
			AppID:    4,
			Patterns: []string{"github.com/myorg/special-repo"},
		},
		{
			Name:     "github-fallback",
			AppID:    5,
			Patterns: []string{"github.com"},
		},
	}

	matcher := NewMatcher(apps)

	tests := []struct {
		name        string
		repoURL     string
		wantAppName string
		wantNoMatch bool
		wantErr     bool
	}{
		{
			name:        "specific repo - longest prefix wins",
			repoURL:     "https://github.com/myorg/special-repo",
			wantAppName: "specific-repo-app",
		},
		{
			name:        "corporate org - prefix match",
			repoURL:     "https://github.com/myorg/other-repo",
			wantAppName: "corporate-app",
		},
		{
			name:        "enterprise host - prefix match",
			repoURL:     "https://github.enterprise.com/company/project",
			wantAppName: "corporate-app",
		},
		{
			name:        "gitlab - prefix match",
			repoURL:     "https://gitlab.com/group/project",
			wantAppName: "gitlab-app",
		},
		{
			name:        "github fallback - shorter prefix",
			repoURL:     "https://github.com/other-org/repo",
			wantAppName: "github-fallback",
		},
		{
			name:        "no match",
			repoURL:     "https://bitbucket.org/user/repo",
			wantNoMatch: true,
		},
		{
			name:    "invalid URL",
			repoURL: "invalid-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := matcher.Match(tt.repoURL)

			if (err != nil) != tt.wantErr {
				t.Errorf("Matcher.Match() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if tt.wantNoMatch {
				if app != nil {
					t.Errorf("Matcher.Match() = %v, want nil", app.Name)
				}
				return
			}

			if app == nil {
				t.Error("Matcher.Match() returned nil, expected an app")
				return
			}

			if app.Name != tt.wantAppName {
				t.Errorf("Matcher.Match() app name = %v, want %v", app.Name, tt.wantAppName)
			}
		})
	}
}

func TestMatcher_EmptyApps(t *testing.T) {
	matcher := NewMatcher([]config.GitHubApp{})

	app, err := matcher.Match("https://github.com/owner/repo")
	if err != nil {
		t.Errorf("Matcher.Match() with empty apps error = %v, want nil", err)
	}
	if app != nil {
		t.Errorf("Matcher.Match() with empty apps = %v, want nil", app)
	}
}

// Old TestPatternMatching removed - no longer needed with simplified matcher
// The Match() function is thoroughly tested with various prefix scenarios above

func TestGetRepositoryInfo(t *testing.T) {
	// Test the public function
	repoInfo, err := GetRepositoryInfo("https://github.com/owner/repo")
	if err != nil {
		t.Errorf("GetRepositoryInfo() error = %v, want nil", err)
	}
	if repoInfo == nil {
		t.Error("GetRepositoryInfo() returned nil")
		return
	}
	if repoInfo.Host != "github.com" {
		t.Errorf("GetRepositoryInfo() Host = %v, want github.com", repoInfo.Host)
	}
}

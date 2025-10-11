package matcher

import (
	"testing"

	"github.com/wherka-ama/gh-app-auth/pkg/config"
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
			Priority: 100,
			Patterns: []string{"github.com/myorg/*", "github.enterprise.com/*"},
		},
		{
			Name:     "opensource-app",
			AppID:    2,
			Priority: 50,
			Patterns: []string{"github.com/*/open-*", "gitlab.com/*"},
		},
		{
			Name:     "default-app",
			AppID:    3,
			Priority: 1,
			Patterns: []string{"*"},
		},
		{
			Name:     "specific-repo-app",
			AppID:    4,
			Priority: 200,
			Patterns: []string{"github.com/myorg/special-repo"},
		},
	}

	matcher := NewMatcher(apps)

	tests := []struct {
		name          string
		repoURL       string
		wantAppName   string
		wantNoMatch   bool
		wantErr       bool
	}{
		{
			name:        "exact match with highest priority",
			repoURL:     "https://github.com/myorg/special-repo",
			wantAppName: "specific-repo-app",
		},
		{
			name:        "corporate org match",
			repoURL:     "https://github.com/myorg/other-repo",
			wantAppName: "corporate-app",
		},
		{
			name:        "opensource pattern match",
			repoURL:     "https://github.com/someone/open-source-project",
			wantAppName: "opensource-app",
		},
		{
			name:        "enterprise host match",
			repoURL:     "https://github.enterprise.com/company/project",
			wantAppName: "corporate-app",
		},
		{
			name:        "gitlab match",
			repoURL:     "https://gitlab.com/group/project",
			wantAppName: "opensource-app",
		},
		{
			name:        "fallback to default",
			repoURL:     "https://bitbucket.org/user/repo",
			wantAppName: "default-app",
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

func TestPatternMatching(t *testing.T) {
	repoInfo := &RepositoryInfo{
		Host:       "github.com",
		Owner:      "myorg",
		Repository: "my-repo",
		FullPath:   "github.com/myorg/my-repo",
		URL:        "https://github.com/myorg/my-repo",
	}

	matcher := &Matcher{}

	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{
			name:    "exact full path match",
			pattern: "github.com/myorg/my-repo",
			want:    true,
		},
		{
			name:    "wildcard org match",
			pattern: "github.com/myorg/*",
			want:    true,
		},
		{
			name:    "wildcard host match",
			pattern: "github.com/*/*",
			want:    true,
		},
		{
			name:    "host only match",
			pattern: "github.com",
			want:    true,
		},
		{
			name:    "owner/repo match",
			pattern: "myorg/my-repo",
			want:    true,
		},
		{
			name:    "repo only match",
			pattern: "my-repo",
			want:    true,
		},
		{
			name:    "wildcard repo match",
			pattern: "my-*",
			want:    true,
		},
		{
			name:    "global wildcard",
			pattern: "*",
			want:    true,
		},
		{
			name:    "no match",
			pattern: "github.com/other/*",
			want:    false,
		},
		{
			name:    "empty pattern",
			pattern: "",
			want:    false,
		},
		{
			name:    "whitespace pattern",
			pattern: "   ",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matcher.patternMatches(tt.pattern, repoInfo); got != tt.want {
				t.Errorf("patternMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindBestMatch(t *testing.T) {
	matches := []config.GitHubApp{
		{Name: "low-priority", Priority: 10},
		{Name: "high-priority", Priority: 100},
		{Name: "medium-priority", Priority: 50},
		{Name: "same-priority-a", Priority: 25},
		{Name: "same-priority-z", Priority: 25},
	}

	matcher := &Matcher{}
	best := matcher.findBestMatch(matches)

	if best.Name != "high-priority" {
		t.Errorf("findBestMatch() = %v, want high-priority", best.Name)
	}

	// Test tie-breaking by name
	tieMatches := []config.GitHubApp{
		{Name: "app-z", Priority: 50},
		{Name: "app-a", Priority: 50},
		{Name: "app-m", Priority: 50},
	}

	bestTie := matcher.findBestMatch(tieMatches)
	if bestTie.Name != "app-a" {
		t.Errorf("findBestMatch() tie-breaker = %v, want app-a", bestTie.Name)
	}

	// Test single match
	singleMatch := []config.GitHubApp{
		{Name: "only-app", Priority: 30},
	}

	bestSingle := matcher.findBestMatch(singleMatch)
	if bestSingle.Name != "only-app" {
		t.Errorf("findBestMatch() single = %v, want only-app", bestSingle.Name)
	}
}

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

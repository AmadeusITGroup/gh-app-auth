package auth

import (
	"testing"
)

func TestParseRepoURL_AdditionalCases(t *testing.T) {
	tests := []struct {
		name      string
		repoURL   string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "HTTPS URL with trailing .git",
			repoURL:   "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "SSH URL with trailing .git",
			repoURL:   "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:    "empty URL",
			repoURL: "",
			wantErr: true,
		},
		{
			name:    "malformed URL - no owner",
			repoURL: "https://github.com/",
			wantErr: true,
		},
		{
			name:    "malformed URL - only owner no repo",
			repoURL: "https://github.com/owner",
			wantErr: true,
		},
		{
			name:      "HTTP URL",
			repoURL:   "http://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "URL with extra slashes",
			repoURL:   "https://github.com/owner/repo/",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "enterprise GitHub URL",
			repoURL:   "https://github.enterprise.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseRepoURL(tt.repoURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for URL %s, got none", tt.repoURL)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}

			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

func TestNewAuthenticator_Initialization(t *testing.T) {
	auth := NewAuthenticator()

	if auth == nil {
		t.Fatal("NewAuthenticator returned nil")
	}

	if auth.jwtGenerator == nil {
		t.Error("jwtGenerator should not be nil")
	}

	if auth.tokenCache == nil {
		t.Error("tokenCache should not be nil")
	}

	if auth.secretsManager == nil {
		t.Error("secretsManager should not be nil")
	}
}

func TestGetInstallationToken_InputValidation(t *testing.T) {
	auth := NewAuthenticator()

	tests := []struct {
		name           string
		jwt            string
		installationID int64
		repoURL        string
		wantErr        bool
	}{
		{
			name:           "empty JWT",
			jwt:            "",
			installationID: 123456,
			repoURL:        "https://github.com/org/repo",
			wantErr:        true,
		},
		{
			name:           "zero installation ID",
			jwt:            "fake.jwt.token",
			installationID: 0,
			repoURL:        "https://github.com/org/repo",
			wantErr:        true,
		},
		{
			name:           "negative installation ID",
			jwt:            "fake.jwt.token",
			installationID: -1,
			repoURL:        "https://github.com/org/repo",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.GetInstallationToken(tt.jwt, tt.installationID, tt.repoURL)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

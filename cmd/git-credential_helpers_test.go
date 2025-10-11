package cmd

import (
	"testing"
)

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name           string
		appPattern     string
		gitCredPattern string
		want           bool
	}{
		{
			name:           "exact match",
			appPattern:     "github.com/myorg/*",
			gitCredPattern: "github.com/myorg/*",
			want:           true,
		},
		{
			name:           "url prefix match",
			appPattern:     "github.com/myorg",
			gitCredPattern: "https://github.com/myorg",
			want:           true,
		},
		{
			name:           "url prefix match with trailing slash",
			appPattern:     "github.com/myorg",
			gitCredPattern: "https://github.com/myorg/",
			want:           true,
		},
		{
			name:           "no match - different org",
			appPattern:     "github.com/myorg",
			gitCredPattern: "https://github.com/other",
			want:           false,
		},
		{
			name:           "prefix match with path",
			appPattern:     "github.com",
			gitCredPattern: "https://github.com/myorg",
			want:           true,
		},
		{
			name:           "enterprise github match",
			appPattern:     "github.enterprise.com/myorg",
			gitCredPattern: "https://github.enterprise.com/myorg",
			want:           true,
		},
		{
			name:           "empty git cred pattern",
			appPattern:     "github.com/myorg",
			gitCredPattern: "",
			want:           false,
		},
		{
			name:           "empty app pattern matches anything",
			appPattern:     "",
			gitCredPattern: "https://github.com/myorg",
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesPattern(tt.appPattern, tt.gitCredPattern)
			if got != tt.want {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v",
					tt.appPattern, tt.gitCredPattern, got, tt.want)
			}
		})
	}
}

func TestBuildRepositoryURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected string
	}{
		{
			name: "empty host",
			input: map[string]string{
				"protocol": "https",
				"host":     "",
				"path":     "myorg/myrepo",
			},
			expected: "",
		},
		{
			name: "no protocol defaults to https",
			input: map[string]string{
				"host": "github.com",
				"path": "myorg/myrepo",
			},
			expected: "github.com/myorg/myrepo",
		},
		{
			name: "path with leading slash",
			input: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "/myorg/myrepo",
			},
			expected: "github.com/myorg/myrepo",
		},
		{
			name: "path with trailing slash and .git",
			input: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "/myorg/myrepo.git/",
			},
			expected: "github.com/myorg/myrepo",
		},
		{
			name: "ssh protocol",
			input: map[string]string{
				"protocol": "ssh",
				"host":     "github.com",
				"path":     "myorg/myrepo",
			},
			expected: "github.com/myorg/myrepo",
		},
		{
			name:     "empty input",
			input:    map[string]string{},
			expected: "",
		},
		{
			name: "host only no path",
			input: map[string]string{
				"protocol": "https",
				"host":     "github.enterprise.com",
			},
			expected: "github.enterprise.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRepositoryURL(tt.input)
			if result != tt.expected {
				t.Errorf("buildRepositoryURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

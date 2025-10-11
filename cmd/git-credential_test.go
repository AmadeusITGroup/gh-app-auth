package cmd

import (
	"strings"
	"testing"
)

func TestReadCredentialInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name: "complete input",
			input: `protocol=https
host=github.com
path=myorg/myrepo

`,
			expected: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "myorg/myrepo",
			},
		},
		{
			name: "host only",
			input: `protocol=https
host=github.com

`,
			expected: map[string]string{
				"protocol": "https",
				"host":     "github.com",
			},
		},
		{
			name: "with username",
			input: `protocol=https
host=github.com
username=x-access-token
path=myorg/myrepo

`,
			expected: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"username": "x-access-token",
				"path":     "myorg/myrepo",
			},
		},
		{
			name:     "empty input",
			input:    "\n",
			expected: map[string]string{},
		},
		{
			name: "with extra whitespace",
			input: `  protocol=https  
  host=github.com  
  path=myorg/myrepo  

`,
			expected: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "myorg/myrepo",
			},
		},
		{
			name: "url format - complete",
			input: `url=https://github.com/myorg/myrepo

`,
			expected: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "myorg/myrepo",
			},
		},
		{
			name: "url format - with .git suffix",
			input: `url=https://github.com/myorg/myrepo.git

`,
			expected: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "myorg/myrepo.git",
			},
		},
		{
			name: "url format - with username",
			input: `url=https://username@github.com/myorg/myrepo

`,
			expected: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "myorg/myrepo",
				"username": "username",
			},
		},
		{
			name: "url format - host only",
			input: `url=https://github.com

`,
			expected: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "",
			},
		},
		{
			name: "url format - enterprise GitHub",
			input: `url=https://github.example.com/myorg/myrepo

`,
			expected: map[string]string{
				"protocol": "https",
				"host":     "github.example.com",
				"path":     "myorg/myrepo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := readCredentialInput(reader)
			if err != nil {
				t.Fatalf("readCredentialInput() error = %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("readCredentialInput() got %d keys, want %d", len(result), len(tt.expected))
			}

			for key, expectedValue := range tt.expected {
				if gotValue, ok := result[key]; !ok {
					t.Errorf("readCredentialInput() missing key %q", key)
				} else if gotValue != expectedValue {
					t.Errorf("readCredentialInput() key %q = %q, want %q", key, gotValue, expectedValue)
				}
			}
		})
	}
}

func TestBuildRepositoryURL(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected string
	}{
		{
			name: "complete HTTPS URL",
			input: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "myorg/myrepo",
			},
			expected: "github.com/myorg/myrepo",
		},
		{
			name: "host only",
			input: map[string]string{
				"protocol": "https",
				"host":     "github.com",
			},
			expected: "github.com",
		},
		{
			name: "no protocol (defaults to https)",
			input: map[string]string{
				"host": "github.com",
				"path": "myorg/myrepo",
			},
			expected: "github.com/myorg/myrepo",
		},
		{
			name: "path with .git suffix",
			input: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "myorg/myrepo.git",
			},
			expected: "github.com/myorg/myrepo",
		},
		{
			name: "path with leading/trailing slashes",
			input: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "/myorg/myrepo/",
			},
			expected: "github.com/myorg/myrepo",
		},
		{
			name: "enterprise GitHub",
			input: map[string]string{
				"protocol": "https",
				"host":     "github.example.com",
				"path":     "myorg/myrepo",
			},
			expected: "github.example.com/myorg/myrepo",
		},
		{
			name: "no host",
			input: map[string]string{
				"protocol": "https",
				"path":     "myorg/myrepo",
			},
			expected: "",
		},
		{
			name:     "empty input",
			input:    map[string]string{},
			expected: "",
		},
		{
			name: "HTTP protocol",
			input: map[string]string{
				"protocol": "http",
				"host":     "github.com",
				"path":     "myorg/myrepo",
			},
			expected: "github.com/myorg/myrepo",
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

func TestBuildRepositoryURL_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected string
	}{
		{
			name: "git clone HTTPS",
			input: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "AmadeusITGroup/gh-app-auth.git",
			},
			expected: "github.com/AmadeusITGroup/gh-app-auth",
		},
		{
			name: "git fetch",
			input: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"path":     "AmadeusITGroup/gh-app-auth",
			},
			expected: "github.com/AmadeusITGroup/gh-app-auth",
		},
		{
			name: "initial connection (host only)",
			input: map[string]string{
				"protocol": "https",
				"host":     "github.com",
			},
			expected: "github.com",
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

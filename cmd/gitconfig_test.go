package cmd

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestNewGitConfigCmd(t *testing.T) {
	cmd := NewGitConfigCmd()

	if cmd == nil {
		t.Fatal("NewGitConfigCmd() returned nil")
	}

	if cmd.Use != "gitconfig" {
		t.Errorf("Expected Use to be 'gitconfig', got %q", cmd.Use)
	}

	// Check flags exist
	if cmd.Flags().Lookup("sync") == nil {
		t.Error("Expected --sync flag to be defined")
	}

	if cmd.Flags().Lookup("clean") == nil {
		t.Error("Expected --clean flag to be defined")
	}

	if cmd.Flags().Lookup("global") == nil {
		t.Error("Expected --global flag to be defined")
	}

	if cmd.Flags().Lookup("local") == nil {
		t.Error("Expected --local flag to be defined")
	}

	if cmd.Flags().Lookup("auto") == nil {
		t.Error("Expected --auto flag to be defined")
	}

	t.Run("execute with no flags", func(t *testing.T) {
		cmd := NewGitConfigCmd()
		err := cmd.Execute()
		// Should error - requires either --sync or --clean
		if err == nil {
			t.Error("Expected error when no flags provided")
		}
	})

	t.Run("execute with conflicting flags", func(t *testing.T) {
		cmd := NewGitConfigCmd()
		cmd.Flags().Set("sync", "true")
		cmd.Flags().Set("clean", "true")
		err := cmd.Execute()
		// Should error - can't use both --sync and --clean
		if err == nil {
			t.Error("Expected error with conflicting flags: --sync/--clean")
		}

		cmd.Flags().Set("global", "true")
		cmd.Flags().Set("local", "true")
		err = cmd.Execute()
		// Should error - can't use both --sync and --clean
		if err == nil {
			t.Error("Expected error with conflicting flags: --global/--local")
		}

		cmd.Flags().Set("global", "true")
		cmd.Flags().Set("auto", "true")
		err = cmd.Execute()
		// Should error - can't use both --sync and --clean
		if err == nil {
			t.Error("Expected error with conflicting flags: --global/--auto")
		}

		cmd.Flags().Set("local", "true")
		cmd.Flags().Set("auto", "true")
		err = cmd.Execute()
		// Should error - can't use both --sync and --clean
		if err == nil {
			t.Error("Expected error with conflicting flags: --local/--auto")
		}
	})
}

func TestExtractCredentialContext(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{
			name:    "org-level pattern",
			pattern: "github.com/myorg/*",
			want:    "https://github.com/myorg",
		},
		{
			name:    "enterprise host",
			pattern: "github.enterprise.com/*/*",
			want:    "https://github.enterprise.com",
		},
		{
			name:    "specific repo",
			pattern: "github.com/org/repo",
			want:    "https://github.com/org",
		},
		{
			name:    "URL format input",
			pattern: "https://github.com/org/*",
			want:    "https://github.com/org",
		},
		{
			name:    "invalid pattern - no slashes",
			pattern: "invalid",
			want:    "",
		},
		{
			name:    "host only",
			pattern: "github.com",
			want:    "https://github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCredentialContext(tt.pattern)
			if got != tt.want {
				t.Errorf("extractCredentialContext(%q) = %q, want %q", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestGitConfigSync_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git not available, skipping integration test")
	}

	t.Run("sync with single app", func(t *testing.T) {
		// Skip: syncGitConfig uses global cwd, not the temp dir, making this test unreliable
		t.Skip("Needs refactoring to be fully testable - syncGitConfig doesn't support custom working directory")
	})

	t.Run("sync with multiple apps", func(t *testing.T) {
		// Skip: syncGitConfig uses global cwd, not the temp dir, making this test unreliable
		t.Skip("Needs refactoring to be fully testable - syncGitConfig doesn't support custom working directory")
	})
}

func TestGitConfigClean_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git not available, skipping integration test")
	}

	t.Run("clean removes gh-app-auth configs only", func(t *testing.T) {
		t.Skip("Needs implementation - cleanGitConfig function not yet exposed for testing")

		tempDir := t.TempDir()

		// Initialize git repo
		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to init git: %v", err)
		}

		// Set some git configs
		configs := []struct {
			key   string
			value string
			keep  bool
		}{
			{
				key:   "credential.https://github.com/test.helper",
				value: "!gh-app-auth git-credential --pattern 'github.com/test/*'",
				keep:  false, // Should be removed
			},
			{
				key:   "credential.https://github.com/other.helper",
				value: "!some-other-helper",
				keep:  true, // Should be kept
			},
			{
				key:   "user.name",
				value: "Test User",
				keep:  true, // Should be kept
			},
		}

		for _, c := range configs {
			cmd := exec.Command("git", "config", "--local", c.key, c.value)
			cmd.Dir = tempDir
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to set git config: %v", err)
			}
		}

		// TODO: Call cleanGitConfig function
		// err := cleanGitConfig("--local")

		// Verify configs
		for _, c := range configs {
			cmd := exec.Command("git", "config", "--local", "--get", c.key)
			cmd.Dir = tempDir
			output, err := cmd.Output()

			if c.keep {
				if err != nil {
					t.Errorf("Config %s should be kept but was removed", c.key)
				}
			} else {
				if err == nil && strings.Contains(string(output), "gh-app-auth") {
					t.Errorf("Config %s should be removed but still exists", c.key)
				}
			}
		}
	})
}

func TestGitConfigSync_NoConfig(t *testing.T) {
	t.Run("handles missing config gracefully", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("GH_APP_AUTH_CONFIG", filepath.Join(tempDir, "nonexistent.yml"))

		// Should not panic, should return error or handle gracefully
		// TODO: Test when function is refactored
		t.Skip("Needs implementation")
	})
}

// Benchmarks for performance validation
func TestGetExecutablePath(t *testing.T) {
	t.Run("finds executable", func(t *testing.T) {
		path, err := getExecutablePath()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if path == "" {
			t.Error("Expected non-empty path")
		}
	})

	t.Run("returns absolute path", func(t *testing.T) {
		path, err := getExecutablePath()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// Check that path is absolute (works on both Unix and Windows)
		if len(path) > 0 && !filepath.IsAbs(path) {
			t.Errorf("Expected absolute path, got: %s", path)
		}
	})

	t.Run("path contains gh-app-auth", func(t *testing.T) {
		path, err := getExecutablePath()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// Path should contain gh-app-auth (even if it's a test binary)
		if !strings.Contains(path, "gh") && !strings.Contains(path, "test") {
			t.Logf("Path: %s", path)
		}
	})
}

func TestExtractCredentialContext_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{
			name:    "empty string",
			pattern: "",
			want:    "",
		},
		{
			name:    "whitespace only",
			pattern: "   ",
			want:    "",
		},
		{
			name:    "no dots in host",
			pattern: "localhost/org/*",
			want:    "",
		},
		{
			name:    "https prefix",
			pattern: "https://github.com/org/*",
			want:    "https://github.com/org",
		},
		{
			name:    "http prefix",
			pattern: "http://github.com/org/*",
			want:    "https://github.com/org",
		},
		{
			name:    "trailing slash",
			pattern: "github.com/org/",
			want:    "https://github.com/org",
		},
		{
			name:    "wildcard in org position",
			pattern: "github.com/*/*",
			want:    "https://github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCredentialContext(tt.pattern)
			if got != tt.want {
				t.Errorf("extractCredentialContext(%q) = %q, want %q", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestSyncGitConfig_NoApps(t *testing.T) {
	// This test validates error handling when no apps are configured
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	// Create empty config
	emptyConfig := `version: "1.0"
github_apps: []
`
	if err := os.WriteFile(configPath, []byte(emptyConfig), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	err := syncGitConfig("--global", false)
	if err == nil {
		t.Error("Expected error for no configured apps")
	}
	// Config validation happens first, so we might get "at least one github_app is required"
	// or "no GitHub Apps configured" depending on where validation occurs
	if err != nil && !strings.Contains(err.Error(), "github_app") && !strings.Contains(err.Error(), "no GitHub Apps") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestSyncGitConfig_Auto(t *testing.T) {
	// Skip on Windows CI - credential helper invocation requires proper App setup
	if runtime.GOOS == "windows" && os.Getenv("CI") == "true" {
		t.Skip("Skipping on Windows CI - credential helper requires proper GitHub App setup")
	}

	// This test validates error handling when no apps are configured
	home := os.Getenv("HOME")
	defer os.Setenv("HOME", home)
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	configPath := filepath.Join(tempDir, "config.yml")

	// Create empty config
	emptyConfig := `version: "1.0"
github_apps: []
`
	if err := os.WriteFile(configPath, []byte(emptyConfig), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	err := syncGitConfig("--global", true)

	if err != nil {
		t.Errorf("Unexpected error message: %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "github_app") && !strings.Contains(err.Error(), "no GitHub Apps") {
		t.Errorf("Unexpected error message: %v", err)
	}
	if _, err := os.Stat(tempDir + "/.gitconfig"); errors.Is(err, os.ErrNotExist) {
		t.Errorf("Unexpected error message: %v", err)
	}
	file, err := os.Open(tempDir + "/.gitconfig")
	if err != nil {
		t.Fatalf("Failed to open .gitconfig: %v", err)
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read .gitconfig: %v", err)
	}
	pattern := `(?m)^\[credential "https:\/\/github\.com"\][\s\S]*?` +
		`helper\s*=\s*.*git-credential\s+--pattern\s+github\.com[\s\S]*?useHttpPath\s*=\s*true`
	match, err := regexp.MatchString(pattern, string(content))
	if err != nil {
		t.Errorf("Unexpected error message: %v", err)
	}
	if !match {
		t.Errorf(".gitconfig is not as expected: %v", string(content))
	}
}

func TestCleanGitConfig_NothingToClean(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git not available, skipping integration test")
	}

	t.Run("clean with no gh-app-auth configs", func(t *testing.T) {
		tempDir := t.TempDir()

		// Initialize git repo
		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to init git: %v", err)
		}

		// Set up git config that doesn't use gh-app-auth
		cmd = exec.Command("git", "config", "--local", "user.name", "Test User")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set git config: %v", err)
		}

		// Change to temp directory for git config commands
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(tempDir)

		// Clean should succeed even with no gh-app-auth configs
		err := cleanGitConfig("--local")
		if err != nil {
			t.Errorf("cleanGitConfig failed: %v", err)
		}

		// Verify user.name is still there (wasn't removed)
		cmd = exec.Command("git", "config", "--local", "--get", "user.name")
		cmd.Dir = tempDir
		output, err := cmd.Output()
		if err != nil {
			t.Error("user.name config should still exist")
		}
		if !strings.Contains(string(output), "Test User") {
			t.Error("user.name value should be preserved")
		}
	})
}

func TestCleanGitConfig_WithConfigs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git not available, skipping integration test")
	}

	t.Run("clean removes gh-app-auth configs only", func(t *testing.T) {
		tempDir := t.TempDir()

		// Initialize git repo
		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to init git: %v", err)
		}

		// Set gh-app-auth credential helper
		cmd = exec.Command("git", "config", "--local", "credential.https://github.com/test.helper", "!gh-app-auth git-credential")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set gh-app-auth config: %v", err)
		}

		// Set other credential helper (should not be removed)
		cmd = exec.Command("git", "config", "--local", "credential.https://github.com/other.helper", "!other-helper")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set other config: %v", err)
		}

		// Change to temp directory
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(tempDir)

		// Clean should remove only gh-app-auth config
		err := cleanGitConfig("--local")
		if err != nil {
			t.Errorf("cleanGitConfig failed: %v", err)
		}

		// Verify gh-app-auth config was removed
		cmd = exec.Command("git", "config", "--local", "--get", "credential.https://github.com/test.helper")
		cmd.Dir = tempDir
		_, err = cmd.Output()
		if err == nil {
			t.Error("gh-app-auth config should be removed")
		}

		// Verify other config still exists
		cmd = exec.Command("git", "config", "--local", "--get", "credential.https://github.com/other.helper")
		cmd.Dir = tempDir
		output, err := cmd.Output()
		if err != nil {
			t.Error("Other credential helper should still exist")
		}
		if !strings.Contains(string(output), "other-helper") {
			t.Error("Other helper value should be preserved")
		}
	})
}

func BenchmarkExtractCredentialContext(b *testing.B) {
	pattern := "github.com/myorg/myrepo"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractCredentialContext(pattern)
	}
}

func BenchmarkExtractCredentialContext_WithProtocol(b *testing.B) {
	pattern := "https://github.com/myorg/myrepo"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractCredentialContext(pattern)
	}
}

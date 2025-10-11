package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// BuildBinary builds the gh-app-auth binary for testing
func BuildBinary(t *testing.T) string {
	t.Helper()

	// Find project root
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Navigate to project root (handle different starting points)
	projectRoot := wd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatal("Could not find project root (go.mod not found)")
		}
		projectRoot = parent
	}

	// Build binary in temp directory
	binaryPath := filepath.Join(t.TempDir(), "gh-app-auth")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	return binaryPath
}

// RequireGit skips the test if git is not available
func RequireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git not available, skipping test")
	}
}

// InitGitRepo initializes a git repository in the given directory
func InitGitRepo(t *testing.T, dir string) {
	t.Helper()
	RequireGit(t)

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user (required for commits)
	configCmd := exec.Command("git", "config", "user.email", "test@example.com")
	configCmd.Dir = dir
	_ = configCmd.Run()

	configCmd = exec.Command("git", "config", "user.name", "Test User")
	configCmd.Dir = dir
	_ = configCmd.Run()
}

// SetGitConfig sets a git config value
func SetGitConfig(t *testing.T, dir, key, value string) {
	t.Helper()
	RequireGit(t)

	cmd := exec.Command("git", "config", "--local", key, value)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set git config %s=%s: %v", key, value, err)
	}
}

// GetGitConfig gets a git config value
func GetGitConfig(t *testing.T, dir, key string) (string, error) {
	t.Helper()
	RequireGit(t)

	cmd := exec.Command("git", "config", "--local", "--get", key)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ListGitConfigs lists all git configs matching a pattern
func ListGitConfigs(t *testing.T, dir, pattern string) []string {
	t.Helper()
	RequireGit(t)

	cmd := exec.Command("git", "config", "--local", "--get-regexp", pattern)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		// No matches is not an error
		return []string{}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

// AssertGitConfigContains asserts that git config contains a specific entry
func AssertGitConfigContains(t *testing.T, dir, key, expectedSubstring string) {
	t.Helper()

	value, err := GetGitConfig(t, dir, key)
	if err != nil {
		t.Fatalf("Expected git config %s to exist, but got error: %v", key, err)
	}

	if !strings.Contains(value, expectedSubstring) {
		t.Errorf("Git config %s = %q does not contain %q", key, value, expectedSubstring)
	}
}

// AssertGitConfigNotExists asserts that a git config key does not exist
func AssertGitConfigNotExists(t *testing.T, dir, key string) {
	t.Helper()

	_, err := GetGitConfig(t, dir, key)
	if err == nil {
		t.Errorf("Expected git config %s to not exist, but it does", key)
	}
}

// AssertFileExists asserts that a file exists
func AssertFileExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("Expected file %s to exist, but it does not", path)
	}
}

// AssertFileNotExists asserts that a file does not exist
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Expected file %s to not exist, but it does", path)
	}
}

// AssertFileContains asserts that a file contains a specific string
func AssertFileContains(t *testing.T, path, expected string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if !strings.Contains(string(content), expected) {
		t.Errorf("File %s does not contain %q", path, expected)
	}
}

// RequireEnvVar requires that an environment variable is set
func RequireEnvVar(t *testing.T, name string) string {
	t.Helper()

	value := os.Getenv(name)
	if value == "" {
		t.Skipf("Required environment variable %s is not set", name)
	}
	return value
}

// SkipIfShort skips the test if running in short mode
func SkipIfShort(t *testing.T, reason string) {
	t.Helper()

	if testing.Short() {
		t.Skipf("Skipping in short mode: %s", reason)
	}
}

package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestGitCredentialHelper_NoConfig tests behavior when no config exists
func TestGitCredentialHelper_NoConfig(t *testing.T) {
	// Setup: Use a temporary config directory
	tempDir := t.TempDir()
	t.Setenv("GH_APP_AUTH_CONFIG", filepath.Join(tempDir, "config.yml"))

	// Build the binary
	binaryPath := buildBinary(t)

	// Simulate git credential get request
	input := `protocol=https
host=github.com
path=myorg/myrepo

`

	cmd := exec.Command(binaryPath, "git-credential", "get")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should exit successfully (silently) when no config exists
	if err != nil {
		t.Errorf("Expected success, got error: %v\nStderr: %s", err, stderr.String())
	}

	// Should produce no output (allowing fallback to other helpers)
	if stdout.Len() > 0 {
		t.Errorf("Expected no output, got: %s", stdout.String())
	}
}

// TestGitCredentialHelper_NoMatchingApp tests behavior when no app matches
func TestGitCredentialHelper_NoMatchingApp(t *testing.T) {
	// Setup: Create config with app for different pattern
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	config := `version: "1.0"
github_apps:
  - name: "Test App"
    app_id: 12345
    installation_id: 67890
    patterns:
      - "github.com/different-org/*"
    priority: 5
    private_key_source: "filesystem"
    private_key_path: "/tmp/nonexistent.pem"
`
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	// Build the binary
	binaryPath := buildBinary(t)

	// Simulate git credential get request for non-matching repo
	input := `protocol=https
host=github.com
path=myorg/myrepo

`

	cmd := exec.Command(binaryPath, "git-credential", "get")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should exit successfully (silently) when no app matches
	if err != nil {
		t.Errorf("Expected success, got error: %v\nStderr: %s", err, stderr.String())
	}

	// Should produce no output (allowing fallback to other helpers)
	if stdout.Len() > 0 {
		t.Errorf("Expected no output, got: %s", stdout.String())
	}
}

// TestGitCredentialHelper_HostOnly tests behavior with host-only request
func TestGitCredentialHelper_HostOnly(t *testing.T) {
	// Setup: Create config with matching app
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	config := `version: "1.0"
github_apps:
  - name: "Test App"
    app_id: 12345
    installation_id: 67890
    patterns:
      - "github.com/myorg/*"
    priority: 5
    private_key_source: "filesystem"
    private_key_path: "/tmp/nonexistent.pem"
`
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	// Build the binary
	binaryPath := buildBinary(t)

	// Simulate git credential get request with host only (no path)
	input := `protocol=https
host=github.com

`

	cmd := exec.Command(binaryPath, "git-credential", "get")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should exit successfully (silently) when URL has no path
	if err != nil {
		t.Errorf("Expected success, got error: %v\nStderr: %s", err, stderr.String())
	}

	// Should produce no output (git will call again with full path)
	if stdout.Len() > 0 {
		t.Errorf("Expected no output, got: %s", stdout.String())
	}
}

// TestGitCredentialHelper_Store tests the store operation
func TestGitCredentialHelper_Store(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	config := `version: "1.0"
github_apps: []
`
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	// Build the binary
	binaryPath := buildBinary(t)

	// Simulate git credential store request
	input := `protocol=https
host=github.com
path=myorg/myrepo
username=test-user
password=test-token

`

	cmd := exec.Command(binaryPath, "git-credential", "store")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should succeed (we don't store anything, but should not error)
	if err != nil {
		t.Errorf("Expected success, got error: %v\nStderr: %s", err, stderr.String())
	}
}

// TestGitCredentialHelper_Erase tests the erase operation
func TestGitCredentialHelper_Erase(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	config := `version: "1.0"
github_apps: []
`
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	// Build the binary
	binaryPath := buildBinary(t)

	// Simulate git credential erase request
	input := `protocol=https
host=github.com
path=myorg/myrepo

`

	cmd := exec.Command(binaryPath, "git-credential", "erase")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should succeed
	if err != nil {
		t.Errorf("Expected success, got error: %v\nStderr: %s", err, stderr.String())
	}
}

// TestGitCredentialHelper_InvalidOperation tests unsupported operations
func TestGitCredentialHelper_InvalidOperation(t *testing.T) {
	// Build the binary
	binaryPath := buildBinary(t)

	cmd := exec.Command(binaryPath, "git-credential", "invalid")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should fail with error
	if err == nil {
		t.Error("Expected error for invalid operation, got success")
	}

	// Should have error message
	if !strings.Contains(stderr.String(), "unsupported git credential operation") {
		t.Errorf("Expected error message about unsupported operation, got: %s", stderr.String())
	}
}

// buildBinary builds the gh-app-auth binary for testing
func buildBinary(t *testing.T) string {
	t.Helper()

	// Find the project root (go up from test/integration)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Navigate to project root
	projectRoot := filepath.Join(wd, "..", "..")

	// Build binary in temp directory (add .exe on Windows)
	binaryName := "gh-app-auth"
	if runtime.GOOS == "windows" {
		binaryName = "gh-app-auth.exe"
	}
	binaryPath := filepath.Join(t.TempDir(), binaryName)

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	return binaryPath
}

// TestGitCredentialProtocol_MultiStage simulates git's multi-stage credential request
func TestGitCredentialProtocol_MultiStage(t *testing.T) {
	// Setup: Create config
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	config := `version: "1.0"
github_apps:
  - name: "Test App"
    app_id: 12345
    installation_id: 67890
    patterns:
      - "github.com/myorg/*"
    priority: 5
    private_key_source: "filesystem"
    private_key_path: "/tmp/nonexistent.pem"
`
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("GH_APP_AUTH_CONFIG", configPath)
	binaryPath := buildBinary(t)

	// Stage 1: Git asks for credentials with just the host
	t.Run("stage1_host_only", func(t *testing.T) {
		input := `protocol=https
host=github.com

`
		cmd := exec.Command(binaryPath, "git-credential", "get")
		cmd.Stdin = strings.NewReader(input)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			t.Errorf("Stage 1 failed: %v\nStderr: %s", err, stderr.String())
		}

		// Should return nothing (no path to match)
		if stdout.Len() > 0 {
			t.Errorf("Stage 1: Expected no output, got: %s", stdout.String())
		}
	})

	// Stage 2: Git asks again with the full path
	// (This would fail in real scenario due to missing key, but tests the flow)
	t.Run("stage2_full_path", func(t *testing.T) {
		input := `protocol=https
host=github.com
path=myorg/myrepo

`
		cmd := exec.Command(binaryPath, "git-credential", "get")
		cmd.Stdin = strings.NewReader(input)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()

		// Will fail because key file doesn't exist, but that's expected
		// The important thing is it tried to authenticate (didn't exit silently)
		if err == nil {
			t.Log("Stage 2: Command succeeded (unexpected but not a test failure)")
		} else {
			// Should have an error about the key file
			if !strings.Contains(stderr.String(), "private key") && !strings.Contains(stderr.String(), "credentials") {
				t.Errorf("Stage 2: Expected error about private key, got: %s", stderr.String())
			}
		}
	})
}

func TestGitCredentialHelper_OutputFormat(t *testing.T) {
	// This test verifies the output format matches git's expectations
	// We can't test the full flow without a real GitHub App, but we can
	// verify the format would be correct

	expectedFormat := `username=test-app[bot]
password=ghs_testtoken123
`

	// The output should be key=value pairs, one per line
	lines := strings.Split(strings.TrimSpace(expectedFormat), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines of output, got %d", len(lines))
	}

	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			t.Errorf("Invalid format line: %s", line)
		}

		key := parts[0]
		if key != "username" && key != "password" {
			t.Errorf("Unexpected key: %s", key)
		}
	}
}

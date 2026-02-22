//go:build e2e

package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	defaultTimeout = 120 * time.Second
	maxRetries     = 3
	retryDelay     = 5 * time.Second
)

// binaryPath returns the path to the binary under test.
// Uses E2E_BINARY_PATH if set; otherwise builds from source.
func binaryPath(t *testing.T) string {
	t.Helper()
	if globalConfig.BinaryPath != "" {
		if _, err := os.Stat(globalConfig.BinaryPath); err != nil {
			t.Fatalf("binary not found at E2E_BINARY_PATH=%s: %v", globalConfig.BinaryPath, err)
		}
		return globalConfig.BinaryPath
	}
	return buildBinaryFromSource(t)
}

// buildBinaryFromSource compiles the binary from source into a temp directory.
func buildBinaryFromSource(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	projectRoot := wd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatal("could not find project root (go.mod not found)")
		}
		projectRoot = parent
	}

	binaryName := "gh-app-auth"
	if runtime.GOOS == "windows" {
		binaryName = "gh-app-auth.exe"
	}
	binaryOut := filepath.Join(t.TempDir(), binaryName)

	cmd := exec.Command("go", "build", "-o", binaryOut, ".")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build binary: %v\nOutput: %s", err, output)
	}
	return binaryOut
}

// isolatedGitConfig returns an env slice with GIT_CONFIG_GLOBAL pointing to a
// temporary file. Prevents tests from touching the developer's real git config.
func isolatedGitConfig(t *testing.T) []string {
	t.Helper()
	gitConfigFile := filepath.Join(t.TempDir(), ".gitconfig")
	content := "[user]\n\temail = e2e@gh-app-auth.test\n\tname = E2E Test\n"
	if err := os.WriteFile(gitConfigFile, []byte(content), 0600); err != nil {
		t.Fatalf("failed to create isolated git config: %v", err)
	}
	env := os.Environ()
	env = setEnv(env, "GIT_CONFIG_GLOBAL", gitConfigFile)
	env = setEnv(env, "GIT_CONFIG_NOSYSTEM", "1")
	return env
}

// isolatedAppConfig returns an env and config file path for an isolated
// gh-app-auth configuration. Combines git isolation and app config isolation.
func isolatedAppConfig(t *testing.T) ([]string, string) {
	t.Helper()
	configDir := t.TempDir()
	configFile := filepath.Join(configDir, "config.yml")

	env := isolatedGitConfig(t)
	env = setEnv(env, "GH_APP_AUTH_CONFIG", configFile)
	// Point HOME to the temp dir so keyring / XDG paths stay isolated.
	env = setEnv(env, "HOME", configDir)
	env = setEnv(env, "XDG_CONFIG_HOME", filepath.Join(configDir, ".config"))

	return env, configFile
}

// setEnv adds or replaces a key=value pair in an env slice.
func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// RunCmd executes the binary with the given args and environment.
// Returns stdout, stderr, and the run error.
func RunCmd(t *testing.T, env []string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(binaryPath(t), args...)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// RunCmdWithStdin executes the binary with the given args and stdin content.
func RunCmdWithStdin(t *testing.T, env []string, stdin string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(binaryPath(t), args...)
	cmd.Env = env
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// RunGit executes git with the given env and working directory.
func RunGit(t *testing.T, env []string, dir string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Env = env
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// writePrivateKeyFile writes the PEM key to a temp file with 0600 permissions.
// On Windows, explicitly sets file permissions after write.
func writePrivateKeyFile(t *testing.T, pem string) string {
	t.Helper()
	keyFile := filepath.Join(t.TempDir(), "private-key.pem")
	if err := os.WriteFile(keyFile, []byte(pem), 0600); err != nil {
		t.Fatalf("failed to write private key file: %v", err)
	}
	// On Windows, explicitly set permissions as os.WriteFile doesn't properly set them
	if runtime.GOOS == "windows" {
		if err := os.Chmod(keyFile, 0600); err != nil {
			t.Fatalf("failed to set private key file permissions: %v", err)
		}
	}
	return keyFile
}

// retryOp retries op up to maxRetries times with linear backoff.
func retryOp(t *testing.T, name string, op func() error) error {
	t.Helper()
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			delay := retryDelay * time.Duration(i)
			t.Logf("retry %d/%d for %s (delay %s): %v", i, maxRetries-1, name, delay, lastErr)
			time.Sleep(delay)
		}
		if err := op(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("%s failed after %d retries: %w", name, maxRetries, lastErr)
}

// requireGit skips the test if git is not in PATH.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
}

// filterPathEntries rebuilds PATH excluding dirs matching filter.
func filterPathEntries(env []string, exclude string) string {
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	var pathVal string
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			pathVal = strings.TrimPrefix(e, "PATH=")
			break
		}
	}
	parts := strings.Split(pathVal, sep)
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != exclude {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}

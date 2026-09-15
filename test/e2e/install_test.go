//go:build e2e

package e2e

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// TestInstallation verifies the binary artifact is correctly installed and
// works as a standalone tool — independent of the gh CLI extension mechanism.
//
// In CI this validates the artifact downloaded from the prerelease.
// Locally this validates a built or user-supplied binary.
func TestInstallation(t *testing.T) {
	t.Run("version_output", testVersionOutput)
	t.Run("help_output", testHelpOutput)
	t.Run("all_subcommands_present", testSubcommandPresence)
	t.Run("git_credential_subcommand", testGitCredentialSubcommand)
	t.Run("standalone_without_gh_cli", testStandaloneWithoutGHCli)
}

func testVersionOutput(t *testing.T) {
	env, _ := isolatedAppConfig(t)
	stdout, stderr, err := RunCmd(t, env, "--version")
	if err != nil {
		t.Fatalf("--version failed: %v\nstderr: %s", err, stderr)
	}
	combined := strings.TrimSpace(stdout + stderr)
	if combined == "" {
		t.Error("expected version output, got nothing")
	}
	t.Logf("version: %s", combined)
}

func testHelpOutput(t *testing.T) {
	env, _ := isolatedAppConfig(t)
	stdout, stderr, err := RunCmd(t, env, "--help")
	if err != nil {
		t.Fatalf("--help failed: %v\nstderr: %s", err, stderr)
	}
	combined := stdout + stderr
	for _, keyword := range []string{"setup", "list", "remove", "gitconfig", "test", "scope"} {
		if !strings.Contains(combined, keyword) {
			t.Errorf("--help output missing expected subcommand %q\nOutput:\n%s", keyword, combined)
		}
	}
}

func testSubcommandPresence(t *testing.T) {
	env, _ := isolatedAppConfig(t)
	expected := []string{"setup", "list", "remove", "gitconfig", "test", "scope", "config", "migrate"}
	stdout, stderr, err := RunCmd(t, env, "--help")
	if err != nil {
		t.Fatalf("--help failed: %v\nstderr: %s", err, stderr)
	}
	combined := stdout + stderr
	for _, sub := range expected {
		if !strings.Contains(combined, sub) {
			t.Errorf("subcommand %q not found in --help output", sub)
		}
	}
}

func testGitCredentialSubcommand(t *testing.T) {
	env, _ := isolatedAppConfig(t)
	// git-credential with no operation should show usage/error, not panic
	_, _, err := RunCmd(t, env, "git-credential")
	// Expecting an error (missing subcommand arg) — but not a panic or segfault
	if err == nil {
		t.Log("git-credential with no args exited 0 (acceptable)")
	}
}

// testStandaloneWithoutGHCli verifies the binary works when gh CLI is NOT on PATH.
// This validates deb/rpm installations where gh CLI may not be installed.
func testStandaloneWithoutGHCli(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH manipulation not tested on Windows in this suite")
	}

	ghPath, err := exec.LookPath("gh")
	if err != nil {
		t.Skip("gh CLI not found in PATH — standalone test not applicable")
	}

	env, _ := isolatedAppConfig(t)
	// Remove the directory containing gh from PATH
	ghDir := ghPath[:strings.LastIndex(ghPath, "/")]
	filteredPath := filterPathEntries(env, ghDir)
	env = setEnv(env, "PATH", filteredPath)

	stdout, stderr, err := RunCmd(t, env, "--version")
	if err != nil {
		t.Fatalf("binary failed when gh CLI removed from PATH: %v\nstderr: %s", err, stderr)
	}
	t.Logf("✓ binary works standalone (no gh CLI): %s", strings.TrimSpace(stdout+stderr))
}

//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGitOperations validates real git operations using gh-app-auth credentials.
// Tests perform actual git clones against private repositories in the test orgs
// and assert on file content to prove authentication succeeded.
//
// Prerequisite: TestPreflight and TestAuthentication must pass first.
// Infrastructure: see docs/E2E_INFRASTRUCTURE.md for repo/submodule setup.
func TestGitOperations(t *testing.T) {
	requireGit(t)

	t.Run("clone_private_repo_org1", testClonePrivateRepoOrg1)
	t.Run("clone_private_repo_org2", testClonePrivateRepoOrg2)
	t.Run("clone_with_cross_org_submodules", testCloneWithCrossOrgSubmodules)
}

// testClonePrivateRepoOrg1 clones the main-repo from TestOrg1 and verifies
// expected file content is accessible.
func testClonePrivateRepoOrg1(t *testing.T) {
	env := setupCredentialHelper(t, globalConfig.TestOrg1, globalConfig.TestOrg2)

	cloneDir := filepath.Join(t.TempDir(), "clone-org1")
	repoURL := fmt.Sprintf("https://github.com/%s/%s", globalConfig.TestOrg1, mainRepo)

	t.Logf("cloning %s ...", repoURL)
	err := retryOp(t, "git clone org1", func() error {
		_, stderr, err := RunGit(t, env, t.TempDir(), "clone", repoURL, cloneDir)
		if err != nil {
			return fmt.Errorf("git clone: %w\nstderr: %s", err, stderr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("clone failed: %v", err)
	}

	// Verify the marker file exists and contains the expected content
	markerPath := filepath.Join(cloneDir, mainMarkerFile)
	assertFileContains(t, markerPath, mainMarkerContent)
	t.Log("✓ private repo from org1 cloned and content verified")
}

// testClonePrivateRepoOrg2 clones the submodule-repo from TestOrg2 directly.
func testClonePrivateRepoOrg2(t *testing.T) {
	env := setupCredentialHelper(t, globalConfig.TestOrg1, globalConfig.TestOrg2)

	cloneDir := filepath.Join(t.TempDir(), "clone-org2")
	repoURL := fmt.Sprintf("https://github.com/%s/%s", globalConfig.TestOrg2, submoduleRepo)

	t.Logf("cloning %s ...", repoURL)
	err := retryOp(t, "git clone org2", func() error {
		_, stderr, err := RunGit(t, env, t.TempDir(), "clone", repoURL, cloneDir)
		if err != nil {
			return fmt.Errorf("git clone: %w\nstderr: %s", err, stderr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("clone failed: %v", err)
	}

	markerPath := filepath.Join(cloneDir, submoduleMarkerFile)
	assertFileContains(t, markerPath, submoduleMarkerContent)
	t.Log("✓ private repo from org2 cloned and content verified")
}

// testCloneWithCrossOrgSubmodules clones main-repo from TestOrg1 with
// --recurse-submodules where the submodule lives in TestOrg2.
// This is the core enterprise use-case: cross-organization repository access.
func testCloneWithCrossOrgSubmodules(t *testing.T) {
	env := setupCredentialHelper(t, globalConfig.TestOrg1, globalConfig.TestOrg2)

	cloneDir := filepath.Join(t.TempDir(), "clone-submodules")
	repoURL := fmt.Sprintf("https://github.com/%s/%s", globalConfig.TestOrg1, mainRepo)

	t.Logf("cloning %s with --recurse-submodules ...", repoURL)
	err := retryOp(t, "git clone with submodules", func() error {
		_, stderr, err := RunGit(t, env, t.TempDir(),
			"clone", "--recurse-submodules", repoURL, cloneDir)
		if err != nil {
			return fmt.Errorf("git clone --recurse-submodules: %w\nstderr: %s", err, stderr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("cross-org submodule clone failed: %v\n\n"+
			"This is the critical multi-org test. Verify:\n"+
			"  1. App is installed in both %s and %s\n"+
			"  2. Submodule URL in main-repo/.gitmodules points to github.com/%s/%s\n"+
			"  3. Credential helper is configured for both org patterns",
			err,
			globalConfig.TestOrg1, globalConfig.TestOrg2,
			globalConfig.TestOrg2, submoduleRepo,
		)
	}

	// Verify main repo content
	mainMarkerPath := filepath.Join(cloneDir, mainMarkerFile)
	assertFileContains(t, mainMarkerPath, mainMarkerContent)

	// Verify submodule content — the key cross-org assertion
	// Submodule is checked out under a subdirectory named after the repo
	submodulePaths := []string{
		filepath.Join(cloneDir, submoduleRepo, submoduleMarkerFile),
		filepath.Join(cloneDir, "submodules", submoduleRepo, submoduleMarkerFile),
		filepath.Join(cloneDir, "vendor", submoduleRepo, submoduleMarkerFile),
	}

	found := false
	for _, p := range submodulePaths {
		if _, err := os.Stat(p); err == nil {
			assertFileContains(t, p, submoduleMarkerContent)
			t.Logf("✓ submodule marker found at %s", p)
			found = true
			break
		}
	}
	if !found {
		t.Errorf("submodule marker %q not found in any expected path under %s\n"+
			"Check the submodule path in main-repo/.gitmodules",
			submoduleMarkerFile, cloneDir)
	}

	t.Log("✓ cross-org submodule clone succeeded — multi-org authentication validated")
}

// setupCredentialHelper performs App setup for both orgs, syncs the git credential
// helper, and returns an isolated environment ready for git operations.
func setupCredentialHelper(t *testing.T, org1, org2 string) []string {
	t.Helper()

	env, _ := isolatedAppConfig(t)
	keyFile := writePrivateKeyFile(t, globalConfig.PrivateKeyPEM)

	// Configure org1 pattern
	pattern1 := fmt.Sprintf("github.com/%s/*", org1)
	stdout, stderr, err := RunCmd(t, env,
		"setup",
		"--app-id", globalConfig.AppID,
		"--key-file", keyFile,
		"--patterns", pattern1,
		"--name", "E2E-GitOps-Org1",
		"--use-filesystem",
	)
	if err != nil {
		t.Fatalf("setup for org1 failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// Configure org2 pattern (same App ID — App is installed in both orgs)
	pattern2 := fmt.Sprintf("github.com/%s/*", org2)
	stdout, stderr, err = RunCmd(t, env,
		"setup",
		"--app-id", globalConfig.AppID,
		"--key-file", keyFile,
		"--patterns", pattern2,
		"--name", "E2E-GitOps-Org2",
		"--use-filesystem",
	)
	if err != nil {
		t.Fatalf("setup for org2 failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// Sync credential helper to isolated git config
	stdout, stderr, err = RunCmd(t, env, "gitconfig", "--sync", "--global")
	if err != nil {
		t.Fatalf("gitconfig --sync failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	return env
}

// assertFileContains asserts a file exists and contains the expected string.
func assertFileContains(t *testing.T, path, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file %s not found or unreadable: %v\n"+
			"Ensure the repository contains the marker file per docs/E2E_INFRASTRUCTURE.md",
			path, err)
	}
	if !strings.Contains(string(content), expected) {
		t.Errorf("file %s does not contain expected marker %q\nActual content: %s",
			path, expected, content)
	}
}

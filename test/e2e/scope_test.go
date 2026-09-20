//go:build e2e

package e2e

import (
	"strings"
	"testing"
)

// TestScopeCommand covers the `scope` command and, through it, the whole
// pkg/scope package end-to-end: fetching installation scope from the GitHub
// API, caching it in the config, and honoring --refresh.
func TestScopeCommand(t *testing.T) {
	t.Run("fails_without_configuration", testScopeNoConfig)
	t.Run("fetch_then_cache_then_refresh", testScopeFetchCacheRefresh)
	t.Run("multi_org_scopes", testScopeMultiOrg)
}

func testScopeNoConfig(t *testing.T) {
	env := isolatedAppConfig(t)

	stdout, stderr, err := RunCmd(t, env, "scope")
	if err == nil {
		t.Fatalf("expected failure with no config file\nstdout: %s", stdout)
	}
	if !strings.Contains(stdout+stderr, "failed to load config") {
		t.Errorf("expected config-load diagnostic\nstdout: %s\nstderr: %s", stdout, stderr)
	}
}

func testScopeFetchCacheRefresh(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Scope-Org1")

	// First run fetches scope from the GitHub API — this is the call that
	// exercises pkg/scope (FetchScope, getInstallation, getRepositories).
	stdout, stderr, err := RunCmd(t, env, "scope")
	if err != nil {
		t.Fatalf("scope failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	combined := stdout + stderr
	if !strings.Contains(combined, "Fetching scope") {
		t.Errorf("expected a scope fetch on first run\nOutput: %s", combined)
	}
	if !strings.Contains(combined, globalConfig.TestOrg1) {
		t.Errorf("expected account %s in scope output\nOutput: %s", globalConfig.TestOrg1, combined)
	}
	if !strings.Contains(combined, "Scope:") {
		t.Errorf("expected scope detail line\nOutput: %s", combined)
	}

	// Second run within the 24h TTL must be served from the cache.
	stdout, stderr, err = RunCmd(t, env, "scope")
	if err != nil {
		t.Fatalf("cached scope run failed: %v\nstderr: %s", err, stderr)
	}
	if strings.Contains(stdout+stderr, "Fetching scope") {
		t.Errorf("second run refetched instead of using the cache\nOutput: %s", stdout+stderr)
	}
	if !strings.Contains(stdout+stderr, globalConfig.TestOrg1) {
		t.Errorf("cached scope lost the account info\nOutput: %s", stdout+stderr)
	}

	// --refresh forces a fetch even inside the TTL.
	stdout, stderr, err = RunCmd(t, env, "scope", "--refresh")
	if err != nil {
		t.Fatalf("scope --refresh failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout+stderr, "Fetching scope") {
		t.Errorf("--refresh did not force a fetch\nOutput: %s", stdout+stderr)
	}
	t.Log("✓ scope fetched, cached, and refreshed correctly")
}

func testScopeMultiOrg(t *testing.T) {
	env := setupCredentialHelper(t, globalConfig.TestOrg1, globalConfig.TestOrg2)

	stdout, stderr, err := RunCmd(t, env, "scope", "--refresh")
	if err != nil {
		t.Fatalf("scope failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	combined := stdout + stderr
	for _, org := range []string{globalConfig.TestOrg1, globalConfig.TestOrg2} {
		if !strings.Contains(combined, org) {
			t.Errorf("scope output missing org %s\nOutput: %s", org, combined)
		}
	}
	t.Log("✓ scope covers both configured org installations")
}

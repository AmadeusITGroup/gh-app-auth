//go:build e2e

package e2e

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// TestTokenCommand covers the `token` command: explicit, repository-scoped
// installation-token minting with fail-closed selectors (the #62 contract —
// --repo must match the selected App's configured route, --installation-id
// must be a configured installation of that App).
func TestTokenCommand(t *testing.T) {
	t.Run("mints_token_for_matching_repo", testTokenMintsForMatchingRepo)
	t.Run("accepts_repo_url_forms", testTokenRepoURLForms)
	t.Run("repo_disambiguates_multi_org_app", testTokenRepoDisambiguates)
	t.Run("installation_id_selector", testTokenInstallationID)
	t.Run("client_id_selector", testTokenClientID)
	t.Run("fail_closed_repo_off_route", testTokenRepoOffRoute)
	t.Run("fail_closed_unknown_app_id", testTokenUnknownAppID)
	t.Run("fail_closed_wrong_installation_id", testTokenWrongInstallationID)
	t.Run("requires_app_selector", testTokenRequiresSelector)
	t.Run("rejects_conflicting_selectors", testTokenConflictingSelectors)
	t.Run("requires_repo_flag", testTokenRequiresRepo)
	t.Run("rejects_non_https_repo_forms", testTokenRejectsNonHTTPS)
	t.Run("rejects_nonpositive_ids", testTokenRejectsBadIDs)
}

// mintedToken asserts a successful `token` run produced a single-line
// installation token, and returns it. The token itself is never logged.
func mintedToken(t *testing.T, stdout, stderr string, err error) string {
	t.Helper()
	if err != nil {
		t.Fatalf("token failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	token := strings.TrimSpace(stdout)
	if token == "" {
		t.Fatalf("token produced no output\nstderr: %s", stderr)
	}
	if strings.ContainsAny(token, " \t\r\n") {
		t.Fatalf("token output must be a single line (got %d chars)", len(token))
	}
	if !strings.HasPrefix(token, "ghs_") {
		t.Fatalf("expected ghs_ installation-token prefix, got %.4s…", token)
	}
	return token
}

// expectTokenError asserts the invocation fails and stderr/stdout contain
// the expected diagnostic substring.
func expectTokenError(t *testing.T, env []string, wantSubstr string, args ...string) {
	t.Helper()
	stdout, stderr, err := RunCmd(t, env, args...)
	if err == nil {
		t.Fatalf("expected failure for %v, got success\nstdout: %s", args, stdout)
	}
	if combined := stdout + stderr; !strings.Contains(combined, wantSubstr) {
		t.Errorf("expected error containing %q\nstdout: %s\nstderr: %s", wantSubstr, stdout, stderr)
	}
}

func testTokenMintsForMatchingRepo(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Token-Org1")
	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg1, mainRepo)

	stdout, stderr, err := RunCmd(t, env,
		"token", "--app-id", globalConfig.AppID, "--repo", repo)
	token := mintedToken(t, stdout, stderr, err)

	// Strongest possible assertion: the minted token authenticates
	// against the GitHub API for the target repository.
	info, err := fetchRepoInfo(token, globalConfig.TestOrg1, mainRepo)
	if err != nil {
		t.Fatalf("minted token rejected by GitHub API: %v", err)
	}
	if !info.Private {
		t.Errorf("expected %s/%s to be private", globalConfig.TestOrg1, mainRepo)
	}
	t.Log("✓ token minted and accepted by GitHub API")
}

func testTokenRepoURLForms(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Token-Forms")

	forms := []string{
		fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg1, mainRepo),
		fmt.Sprintf("https://github.com/%s/%s", globalConfig.TestOrg1, mainRepo),
	}
	for _, repo := range forms {
		stdout, stderr, err := RunCmd(t, env,
			"token", "--app-id", globalConfig.AppID, "--repo", repo)
		mintedToken(t, stdout, stderr, err)
	}
	t.Log("✓ host/owner/repo and https:// forms both accepted")
}

// With the same App configured under two org patterns (two installations),
// --repo alone must select the right installation — the multi-org enterprise case.
func testTokenRepoDisambiguates(t *testing.T) {
	env := setupCredentialHelper(t, globalConfig.TestOrg1, globalConfig.TestOrg2)
	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg2, submoduleRepo)

	stdout, stderr, err := RunCmd(t, env,
		"token", "--app-id", globalConfig.AppID, "--repo", repo)
	token := mintedToken(t, stdout, stderr, err)

	if _, err := fetchRepoInfo(token, globalConfig.TestOrg2, submoduleRepo); err != nil {
		t.Fatalf("minted token rejected by GitHub API for org2 repo: %v", err)
	}
	t.Log("✓ --repo disambiguated the multi-org App installation")
}

func testTokenInstallationID(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Token-InstID")
	instID, err := discoverInstallationID(globalConfig.TestOrg1)
	if err != nil {
		t.Fatalf("installation ID discovery failed: %v", err)
	}
	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg1, mainRepo)

	stdout, stderr, err := RunCmd(t, env,
		"token",
		"--app-id", globalConfig.AppID,
		"--installation-id", strconv.FormatInt(instID, 10),
		"--repo", repo,
	)
	mintedToken(t, stdout, stderr, err)
	t.Log("✓ --installation-id selector minted a token")
}

func testTokenClientID(t *testing.T) {
	clientID, err := discoverClientID()
	if err != nil {
		t.Fatalf("client ID discovery failed: %v", err)
	}

	// Configure via --client-id so the stored entry is selectable by it.
	env := isolatedAppConfig(t)
	keyFile := writePrivateKeyFile(t, globalConfig.PrivateKeyPEM)
	pattern := fmt.Sprintf("github.com/%s/*", globalConfig.TestOrg1)
	stdout, stderr, err := RunCmd(t, env,
		"setup",
		"--client-id", clientID,
		"--key-file", keyFile,
		"--patterns", pattern,
		"--name", "E2E-Token-ClientID",
		"--use-filesystem",
	)
	if err != nil {
		t.Fatalf("setup --client-id failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg1, mainRepo)
	stdout, stderr, err = RunCmd(t, env,
		"token", "--client-id", clientID, "--repo", repo)
	mintedToken(t, stdout, stderr, err)
	t.Log("✓ --client-id selector minted a token")
}

// Fail-closed: --repo must match the selected App's configured route.
func testTokenRepoOffRoute(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Token-OffRoute")
	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg2, submoduleRepo)

	expectTokenError(t, env, "matches repository",
		"token", "--app-id", globalConfig.AppID, "--repo", repo)
}

func testTokenUnknownAppID(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Token-UnknownApp")
	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg1, mainRepo)

	expectTokenError(t, env, "no configured GitHub App matches app ID 999999999",
		"token", "--app-id", "999999999", "--repo", repo)
}

// Fail-closed: --installation-id must be a configured installation of the
// App — using org2's real installation ID against an org1-only config.
func testTokenWrongInstallationID(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Token-WrongInst")
	otherInstID, err := discoverInstallationID(globalConfig.TestOrg2)
	if err != nil {
		t.Fatalf("installation ID discovery failed: %v", err)
	}
	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg1, mainRepo)

	expectTokenError(t, env, "installation ID",
		"token",
		"--app-id", globalConfig.AppID,
		"--installation-id", strconv.FormatInt(otherInstID, 10),
		"--repo", repo,
	)
}

func testTokenRequiresSelector(t *testing.T) {
	env := isolatedAppConfig(t)
	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg1, mainRepo)

	expectTokenError(t, env, "must specify either --app-id or --client-id",
		"token", "--repo", repo)
}

func testTokenConflictingSelectors(t *testing.T) {
	env := isolatedAppConfig(t)
	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg1, mainRepo)

	expectTokenError(t, env, "cannot use both --app-id and --client-id",
		"token", "--app-id", globalConfig.AppID, "--client-id", "Iv1.placeholder", "--repo", repo)
}

func testTokenRequiresRepo(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Token-NoRepo")

	expectTokenError(t, env, "repository is required",
		"token", "--app-id", globalConfig.AppID)
}

func testTokenRejectsNonHTTPS(t *testing.T) {
	env := isolatedAppConfig(t)

	expectTokenError(t, env, "SSH repository syntax is not supported",
		"token", "--app-id", globalConfig.AppID,
		"--repo", fmt.Sprintf("git@github.com:%s/%s.git", globalConfig.TestOrg1, mainRepo))
	expectTokenError(t, env, "must use HTTPS",
		"token", "--app-id", globalConfig.AppID,
		"--repo", fmt.Sprintf("http://github.com/%s/%s", globalConfig.TestOrg1, mainRepo))
}

func testTokenRejectsBadIDs(t *testing.T) {
	env := isolatedAppConfig(t)
	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg1, mainRepo)

	expectTokenError(t, env, "app ID must be positive",
		"token", "--app-id", "-1", "--repo", repo)
	expectTokenError(t, env, "installation ID must be positive",
		"token", "--app-id", globalConfig.AppID, "--installation-id", "-1", "--repo", repo)
}

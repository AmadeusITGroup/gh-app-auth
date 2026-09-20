//go:build e2e

package e2e

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestExecCommand covers the `exec` command: resolving a configured
// credential, injecting it into a child process environment (never printing
// it), and propagating the child's exit behavior. Selector handling shares
// the fail-closed contract introduced for `token` in #62.
func TestExecCommand(t *testing.T) {
	t.Run("injects_token_env_for_repo", testExecInjectsTokenEnv)
	t.Run("child_stdout_flows_through", testExecChildStdout)
	t.Run("child_failure_propagates", testExecChildFailure)
	t.Run("app_and_installation_selectors_without_repo", testExecSelectorsNoRepo)
	t.Run("repo_disambiguates_multi_org_app", testExecRepoDisambiguates)
	t.Run("fail_closed_ambiguous_app_no_repo", testExecAmbiguousNoRepo)
	t.Run("fail_closed_repo_off_route", testExecRepoOffRoute)
	t.Run("fail_closed_no_credential_for_repo", testExecNoCredentialForRepo)
	t.Run("fails_outside_git_repo_without_repo", testExecOutsideGitRepo)
	t.Run("requires_command", testExecRequiresCommand)
}

const (
	poisonedToken = "e2e-poisoned-not-a-token"
	githubHost    = "github.com"
)

// execEnvProbe returns a child command that exits 0 only when the expected
// environment is present. It never prints the token — CI logs stay clean.
// wantRepo == "" asserts GH_REPO is NOT set (repo-less selector path).
func execEnvProbe(wantRepo string) []string {
	if runtime.GOOS == "windows" {
		// Concatenation, not Sprintf: %VAR% collides with format verbs.
		script := `if not defined GH_TOKEN exit /b 1& if "%GH_TOKEN%"=="` + poisonedToken +
			`" exit /b 1& if not "%GH_HOST%"=="` + githubHost + `" exit /b 1`
		if wantRepo != "" {
			script += `& if not "%GH_REPO%"=="` + wantRepo + `" exit /b 1`
		} else {
			script += `& if defined GH_REPO exit /b 1`
		}
		return []string{"cmd", "/c", script}
	}

	script := fmt.Sprintf(
		`[ -n "$GH_TOKEN" ] && [ "$GH_TOKEN" != "%s" ] && [ "$GH_HOST" = "%s" ]`,
		poisonedToken, githubHost)
	if wantRepo != "" {
		script += fmt.Sprintf(` && [ "$GH_REPO" = "%s" ]`, wantRepo)
	} else {
		script += ` && [ -z "$GH_REPO" ]`
	}
	return []string{"sh", "-c", script}
}

// expectExecError mirrors expectTokenError for the exec command.
func expectExecError(t *testing.T, env []string, wantSubstr string, args ...string) {
	t.Helper()
	stdout, stderr, err := RunCmd(t, env, args...)
	if err == nil {
		t.Fatalf("expected failure for %v, got success\nstdout: %s", args, stdout)
	}
	if combined := stdout + stderr; !strings.Contains(combined, wantSubstr) {
		t.Errorf("expected error containing %q\nstdout: %s\nstderr: %s", wantSubstr, stdout, stderr)
	}
}

// exec with only --repo resolves via repository routing, mints a token and
// injects GH_TOKEN/GH_HOST/GH_REPO into the child — overwriting any
// pre-existing token variables.
func testExecInjectsTokenEnv(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Exec-Inject")
	env = setEnv(env, "GH_TOKEN", poisonedToken) // must be replaced, not inherited
	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg1, mainRepo)

	args := append([]string{"exec", "--repo", repo, "--"}, execEnvProbe(repo)...)
	stdout, stderr, err := RunCmd(t, env, args...)
	if err != nil {
		t.Fatalf("exec probe failed — GH_TOKEN/GH_HOST/GH_REPO not injected correctly: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}
	t.Log("✓ exec injected fresh GH_TOKEN/GH_HOST/GH_REPO into the child")
}

func testExecChildStdout(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Exec-Stdout")
	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg1, mainRepo)

	// The child is the binary itself — guaranteed present, prints "version".
	stdout, stderr, err := RunCmd(t, env,
		"exec", "--repo", repo, "--", binaryPath(t), "--version")
	if err != nil {
		t.Fatalf("exec --version child failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "version") {
		t.Errorf("child stdout did not flow through\nstdout: %s", stdout)
	}
	t.Log("✓ child stdout propagated through exec")
}

// A failing child must surface as a non-zero exit (and its stderr must
// flow through) — callers depend on exec being transparent.
func testExecChildFailure(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Exec-ChildFail")
	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg1, mainRepo)

	var child []string
	if runtime.GOOS == "windows" {
		child = []string{"cmd", "/c", "echo child-failed 1>&2& exit /b 3"}
	} else {
		child = []string{"sh", "-c", "echo child-failed >&2; exit 3"}
	}

	args := append([]string{"exec", "--repo", repo, "--"}, child...)
	stdout, stderr, err := RunCmd(t, env, args...)
	if err == nil {
		t.Fatalf("expected non-zero exit when child fails\nstdout: %s", stdout)
	}
	if !strings.Contains(stderr, "child-failed") {
		t.Errorf("child stderr did not flow through\nstderr: %s", stderr)
	}
	t.Log("✓ failing child propagated as non-zero exit with its stderr")
}

// --app-id + --installation-id with no --repo: host is inferred from the
// App's patterns, GH_REPO stays unset — the repo-independent API path.
func testExecSelectorsNoRepo(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Exec-Selectors")
	instID, err := discoverInstallationID(globalConfig.TestOrg1)
	if err != nil {
		t.Fatalf("installation ID discovery failed: %v", err)
	}

	args := append([]string{
		"exec",
		"--app-id", globalConfig.AppID,
		"--installation-id", strconv.FormatInt(instID, 10),
		"--",
	}, execEnvProbe("")...)
	stdout, stderr, err := RunCmd(t, env, args...)
	if err != nil {
		t.Fatalf("exec with selectors failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	t.Log("✓ --app-id + --installation-id minted a token without --repo")
}

func testExecRepoDisambiguates(t *testing.T) {
	env := setupCredentialHelper(t, globalConfig.TestOrg1, globalConfig.TestOrg2)
	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg2, submoduleRepo)

	args := append([]string{
		"exec", "--app-id", globalConfig.AppID, "--repo", repo, "--",
	}, execEnvProbe(repo)...)
	stdout, stderr, err := RunCmd(t, env, args...)
	if err != nil {
		t.Fatalf("exec disambiguation failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	t.Log("✓ --repo selected the correct installation across two orgs")
}

// Same App ID configured for two orgs, no --repo: fail closed instead of
// guessing which installation the caller meant.
func testExecAmbiguousNoRepo(t *testing.T) {
	env := setupCredentialHelper(t, globalConfig.TestOrg1, globalConfig.TestOrg2)

	expectExecError(t, env, "multiple GitHub App configurations match",
		"exec", "--app-id", globalConfig.AppID, "--", binaryPath(t), "--version")
}

func testExecRepoOffRoute(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Exec-OffRoute")
	repo := fmt.Sprintf("github.com/%s/%s", globalConfig.TestOrg2, submoduleRepo)

	expectExecError(t, env, "matches repository",
		"exec", "--app-id", globalConfig.AppID, "--repo", repo, "--", binaryPath(t), "--version")
}

func testExecNoCredentialForRepo(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Exec-NoCred")

	expectExecError(t, env, "no credential configured",
		"exec", "--repo", "github.com/nonexistent-org-xyz/repo", "--", binaryPath(t), "--version")
}

// Without selectors or --repo, exec derives the repository from the git
// remote — outside a repository it must fail with a clear message.
func testExecOutsideGitRepo(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Exec-NoGit")
	dir := t.TempDir()

	args := append([]string{"exec", "--"}, execEnvProbe("")...)
	stdout, stderr, err := RunCmdInDir(t, env, dir, args...)
	if err == nil {
		t.Fatalf("expected failure outside a git repository\nstdout: %s", stdout)
	}
	if !strings.Contains(stdout+stderr, "not in a git repository") {
		t.Errorf("expected 'not in a git repository' diagnostic\nstdout: %s\nstderr: %s", stdout, stderr)
	}
}

func testExecRequiresCommand(t *testing.T) {
	env := setupSingleApp(t, globalConfig.TestOrg1, "E2E-Exec-NoCmd")

	stdout, stderr, err := RunCmd(t, env, "exec")
	if err == nil {
		t.Fatalf("expected failure for bare 'exec'\nstdout: %s", stdout)
	}
	if !strings.Contains(stdout+stderr, "command") {
		t.Errorf("expected a usage error mentioning the missing command\nstdout: %s\nstderr: %s",
			stdout, stderr)
	}
}

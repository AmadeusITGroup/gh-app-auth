---
title: Testing guide
description: The test levels in gh-app-auth, how to run them, how tests are isolated, and where live end-to-end testing is documented.
sidebar:
  order: 3
---

gh-app-auth has three kinds of automated tests, and one legacy script that is not part of any gate.
This page describes how they are organized, not their current counts or pass rates; run the commands to
see those.

## Test levels

| Level | Where | Needs credentials? |
| --- | --- | --- |
| Unit tests | `*_test.go` next to the code in `cmd/` and `pkg/` | No |
| Integration tests | [`test/integration`](../../test/integration), using helpers in [`test/testutil`](../../test/testutil) | No |
| Live end-to-end tests | [`test/e2e`](../../test/e2e), behind the `e2e` build tag | **Yes**: a real GitHub App and test organizations |

Unit and integration tests use temporary directories, fake configurations, static test fixtures and a
mock GitHub server (`MockGitHubServer` in `test/testutil`) instead of real accounts. They never need real
credentials.

## Running the default tests

```bash
make test
go test -race ./...
go test -v ./pkg/auth/...
go test -v -run TestName ./pkg/...
```

`make test` runs `go test ./...`. Every file in `test/e2e` carries `//go:build e2e`, so the default
`go test ./...` and `make test` **do not compile or run the live end-to-end suite**. The CI test job
runs `go test -race` with coverage over `./...`, and `make quality` adds the lint, coverage threshold and
security scans. See [development](development.md) for the everyday commands.

## Writing unit tests

- Use table-driven tests with cases for valid input, edge cases and errors.
- Isolate the filesystem and environment with `t.TempDir()` and `t.Setenv()`. Point
  `GH_APP_AUTH_CONFIG` at a file in the temporary directory so a test cannot read your real configuration
  file, and also point `HOME` at a temporary directory so default locations, such as the secrets
  directory under `HOME`, stay inside the test. `GH_APP_AUTH_CONFIG` alone does not isolate stored
  secrets, and neither override isolates the **OS keyring**: a test that stores or reads secrets must
  use a mock secrets or keyring backend and must never touch real entries.
- Build configuration with the helpers in `test/testutil`, such as `ConfigBuilder` and
  `CreateTempConfig`. `CreateTestPrivateKey` writes a **static legacy fixture**; it does not generate a
  key. New tests that need a key should generate a throwaway key at runtime rather than extend the
  embedded fixtures.
- Mock external services through interfaces or the mock server, and never call the real GitHub API from a
  default test.
- Never log, print or commit tokens or keys, and never commit test credentials, even expired ones.
- Keep coverage at or above the thresholds enforced by `make quality`.

```go
func TestFunction(t *testing.T) {
    home := t.TempDir()
    t.Setenv("HOME", home)
    t.Setenv("GH_APP_AUTH_CONFIG", filepath.Join(home, "config.yml"))
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "example", "example", false},
        {"empty input", "", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Fatalf("Function() error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("Function() = %q, want %q", got, tt.want)
            }
        })
    }
}
```

## Integration tests

The integration tests in `test/integration` exercise the Git credential helper with the credential
protocol's input and output: no configuration, no matching App, a host-only query, `store` and
`erase`, an invalid operation, multi-stage protocols and the output format. Each test points
`GH_APP_AUTH_CONFIG` at a temporary directory and none contacts GitHub.

For the host-only case, the helper returns empty output when Git provides a host without a path. Do not
assume Git then asks again with the full path. `gitconfig --sync` writes `useHttpPath` for path-specific
patterns, which is what makes Git send the repository path in the first place. See the
[Git credential helper guide](../guides/git-credential-helper.md).

## Diagnosing a failing test

- Run one package or test with `-v` and `-run`, and add `-race` when a failure looks timing-related.
- For helper behavior, enable the [diagnostic log](../troubleshoot/index.md#diagnostics) with an absolute
  `GH_APP_AUTH_DEBUG_LOG` path and a **test** configuration, and read the flow steps. Do not capture Git or
  curl traces together with real credentials.
- A test that needs a built binary can use `testutil.BuildBinary` instead of relying on one you built
  earlier.

## Live end-to-end tests

The end-to-end (E2E) suite checks gh-app-auth against **real GitHub**: a real GitHub App, two dedicated
test organizations and private repositories, including a submodule in the second organization. It is the
only test level that needs real credentials. It is not part of `make test`, and it is not a demo to run
casually.

This tutorial covers running the maintained Go suite. Provisioning the fixture is a separate,
maintainer-only job described in [end-to-end test infrastructure](e2e-infrastructure.md).

### Before you start

- You are an authorized maintainer, or you have been given access to the dedicated test fixture. Never
  point the suite at production organizations or repositories.
- The fixture exists: two **private** repositories in two non-production organizations, and a GitHub App
  installed on both with **read-only Contents** permission. Creating those, installing the App and
  consenting to its permissions require organization administrators; this tutorial does not do them.
- You have the App's numeric ID and its private key in a file approved for secrets, readable only by you.
- Go matching `go.mod`, Git, and the GitHub CLI are installed.

### What the suite covers

Every Go file in [`test/e2e`](../../test/e2e) has the `//go:build e2e` constraint. The tests are:

| File | Area |
| --- | --- |
| `preflight_test.go` | Fixture and credential sanity checks |
| `install_test.go` | Extension installation checks |
| `auth_test.go` | Authentication flows |
| `token_test.go` | The `token` command |
| `exec_test.go` | The `exec` command |
| `scope_test.go` | Installation scope |
| `gitops_test.go` | Git operations through the helper, including the submodule fixture |
| `helpers_test.go`, `suite_test.go` | Shared helpers, configuration loading and `TestMain` |

The helpers run each test with an isolated configuration, so a test does not read or write your own
gh-app-auth configuration. The fixture is:

- a private `main-repo` in organization 1, containing `data/main-marker.txt` with the text
  `gh-app-auth-e2e-main-marker`; and
- a private `submodule-repo` in organization 2, containing `data/submodule-marker.txt` with the text
  `gh-app-auth-e2e-submodule-marker`, referenced as a submodule of `main-repo`.

### Configure the environment

The suite reads these variables. Set them in your shell for the test run only; do not write them to files in
the repository.

| Variable | Required | Meaning |
| --- | --- | --- |
| `E2E_APP_ID` | Yes | The App ID |
| `E2E_PRIVATE_KEY` | One of the two | The raw PEM. Takes precedence over the base64 form |
| `E2E_PRIVATE_KEY_B64` | One of the two | The PEM, base64-encoded |
| `E2E_BINARY_PATH` | No | A binary to test; the suite builds from source when unset |
| `E2E_TEST_ORG_1` | No | Defaults to `gh-app-auth-test-1` |
| `E2E_TEST_ORG_2` | No | Defaults to `gh-app-auth-test-2` |

`TestMain` **fails** rather than skips when the credentials are missing: it needs `E2E_APP_ID` and either
`E2E_PRIVATE_KEY` or `E2E_PRIVATE_KEY_B64`. Load the key from the approved key file without printing it:

```bash
export E2E_APP_ID="123456"
export E2E_PRIVATE_KEY_B64="$(base64 < "$KEY_FILE" | tr -d '\n')"
```

`KEY_FILE` is the path of the approved test key file, and `123456` is a placeholder. Never echo these
variables, enable shell tracing, or paste them into logs and issues.

### Run the preflight first

```bash
go test -v -tags=e2e -run TestPreflight -timeout=2m ./test/e2e/...
```

The preflight has four subtests, and all must pass before anything else: `binary_accessible`,
`app_token_valid`, `org1_main_repo_is_private` and `org2_submodule_repo_is_private`. They confirm the
binary runs, the App can obtain a token, and both fixture repositories are private. A failure here
means the fixture or credentials are wrong; fix that rather than running the rest.

### Run the maintained suite

```bash
make test-e2e
make test-e2e-local
go test -v -tags=e2e -run TestExec -timeout=5m ./test/e2e/...
```

`make test-e2e` runs `go test -v -tags=e2e -timeout=15m ./test/e2e/...` and builds the binary from source.
`make test-e2e-local` builds a local binary and points `E2E_BINARY_PATH` at it. `E2E_BINARY_PATH` also lets
you test a downloaded release binary. To measure coverage of the CLI during the E2E run, see the
coverage section of the [infrastructure runbook](e2e-infrastructure.md).

### Inspect the results

Read the Go test output and the local artifacts a run leaves, such as coverage files when you ran the
coverage target. Do not add output of credentials to debug a failure. If you need more detail, enable the [diagnostic log](../troubleshoot/index.md#diagnostics) with
an absolute `GH_APP_AUTH_DEBUG_LOG` path, review it for sensitive values before sharing it, and delete it
afterwards.

### What the suite does not do

- It does not create or delete organizations, repositories, Apps or installations.
- It does not need, and you should not run, broad cleanup. The fixture is a set of static repositories that
  runs do not modify.
- It is not a performance benchmark; do not rely on timing observations from it.

### In CI

`.github/workflows/e2e.yml` runs the suite on pull requests, on pushes to `main` and on manual dispatch,
and the release pipeline gates on it. A secrets check skips the three operating-system jobs when the
repository credentials are unavailable, for example on a fork. Running the Go suite directly never
skips: it fails without credentials. See the [release runbook](releasing.md).

### Legacy scripts

The shell scripts under [`test/e2e/scripts`](../../test/e2e/scripts) are an older, manual approach, are not
maintained, and are not the acceptance path. Some use flags that no longer exist and one removes files
broadly. Use the Go suite above instead; the directory README says which scripts are legacy.

## Legacy manual script

`test/manual/test-git-credential.sh` predates the current suite and is **not a gate and not
recommended**. It unsets `GH_APP_AUTH_CONFIG`, so it may fall back to your actual configuration, and it
captures credential output, which can contain a real token. There is no execution recipe for it here.
If you must read it, treat it as a live-secret risk and do not run it with real keys.

## Best practices

- Test the security-sensitive paths first: key handling, permission checks, token handling and logging.
- Keep tests deterministic and independent; no test should rely on another's state.
- Prefer small tests that fail with a precise message over large end-to-end scripts.
- Add the test with the fix: reproduce a bug, watch the test fail, then fix it.

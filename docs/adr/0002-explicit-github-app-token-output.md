# 0002 Explicit GitHub App Token Output

Date: 2026-08-20
Status: Proposed

## Context

`gh auth token` returns a token for the active GitHub CLI account. `gh-app-auth` does not have an active-account concept: one local configuration can contain multiple GitHub Apps and multiple installations of the same App. Users need a deterministic way to request an installation token for a particular configured App and repository:

```text
gh app-auth token --app-id <id> --repo <host>/<owner>/<repo>
```

The token is a GitHub App installation token, not a user OAuth token. It is generated from the configured App private key, exchanged with the GitHub API, and returned to the caller.

The current `exec` implementation contains the reusable selection and authentication flow, but its explicit selectors need a strict configuration boundary before they can be shared with a raw-token command:

- `--repo` is currently used to derive the host for the installation-token request without always verifying that it matches the selected App's configured patterns.
- `--installation-id` can fall back to an App-ID candidate and overwrite its configured installation ID when the requested installation is not configured.

These behaviors are unsafe and surprising under the credential-broker threat model, where scripts, CI steps, repository automation, or agent-generated commands may supply the arguments. They are also correctness issues under a trusted-operator model because the flags are documented as selectors but can act as authorization overrides.

The implementation is anchored in:

- `cmd/exec.go` — explicit App selection and child-environment credential injection;
- `cmd/token.go` — raw-token command and strict input contract;
- `pkg/auth/authenticator.go` — JWT generation and installation-token exchange;
- `pkg/config/config.go` — numeric App ID and Client ID configuration;
- `pkg/matcher/matcher.go` — longest-prefix and optional installation-scope matching;
- `pkg/cache/cache.go` — memory-only installation-token caching.

### Alternatives considered

- **Thin wrapper around `exec`.** Rejected because it would inherit the existing host and installation-boundary defects, PAT fallback behavior, and cache semantics without defining a raw-secret output contract.
- **Delegate to `gh auth token`.** Rejected because it assumes the active GitHub CLI account and cannot use a configured GitHub App private key.
- **Allow arbitrary repository hosts and installation IDs.** Rejected as the default because it bypasses configured routing policy. If this capability is required later, it must be a separately named, explicitly privileged operation.
- **Persist installation tokens in the keyring.** Rejected because installation tokens are powerful and short-lived; the existing project policy is memory-only token handling.
- **Duplicate the resolver in `cmd/token.go`.** Rejected because `exec` and `token` would develop different security policies around the same private-key authority.

## Decision

We will add an explicit `token` command backed by a shared, strict App-selection path.

### Command contract

The first version accepts:

```text
gh app-auth token \
  --app-id <positive numeric ID> \
  --repo <host>/<owner>/<repo>
```

It also accepts `--client-id` as the alternative identifier and `--installation-id` as an optional disambiguator.

The command will:

1. Require exactly one of `--app-id` and `--client-id`.
2. Require a repository argument and canonicalize it to `host/owner/repository`.
3. Reject unsupported or ambiguous repository input before authentication.
4. Select only GitHub App configurations matching the explicit identifier.
5. Require an explicit installation ID to be an exact configured identifier/installation pair.
6. Require the repository to match the selected App's configured host/pattern and optional installation scope.
7. Load the private key only after selection and route validation succeed.
8. Generate a fresh JWT and request a fresh installation token without using the existing token cache.
9. Print exactly the installation token followed by one newline to stdout.
10. Print no labels, username, progress messages, JWTs, private keys, or token values to stdout or diagnostic logs.

PATs and the active GitHub CLI account are not selected implicitly by this command.

### Shared selector hardening

The explicit selector path used by `exec` and `token` will:

- match both App identifier and installation ID when both are supplied;
- fail if an explicit installation ID is not configured;
- match a supplied repository against the selected App's configured route before generating a JWT;
- fail rather than selecting the first candidate when no unique configured route exists;
- never overwrite a selected configuration's installation ID from caller input.

The selected, validated repository remains the target for the installation-token request. The request host cannot be introduced by an unvalidated caller-controlled value.

### Fresh token operation

`pkg/auth.Authenticator` will expose a dedicated fresh-mint operation separate from `GetCredentials`. The operation generates a new JWT and calls the installation-token endpoint without reading from or writing to the in-memory installation-token cache. It will reject empty or newline-containing token responses and use an HTTP client that refuses cross-host redirects.

### Threat model

This decision treats configured App patterns and installations as meaningful policy boundaries. The extension may be invoked by automation that is not trusted to choose arbitrary hosts or installations, even though a fully trusted local operator may have enough access to mint tokens directly with the private key.

The command's raw stdout is an intentional disclosure to the direct caller. Documentation must warn callers about shell tracing, terminal capture, pipeline exposure, and writing the token to unprotected files.

## Consequences

### Positive

- Users can request a GitHub App installation token without an active `gh` account.
- Multiple App installations can be selected deterministically by App ID, Client ID, installation ID, and repository route.
- Foreign repository hosts and unconfigured installations are rejected before JWT generation and network access.
- `exec` and `token` share one selection policy rather than duplicating credential-boundary logic.
- Installation tokens remain ephemeral and are not persisted to disk or the keyring.
- Machine-readable use is straightforward because successful stdout contains only the token.

### Costs and limitations

- The first implementation requires an explicit App selector and repository; it does not infer the current repository or active GitHub account.
- `--installation-id` cannot target an installation absent from local configuration. Supporting arbitrary installations requires a separate security decision and command contract.
- Repeated `token` calls intentionally bypass the in-process cache and consume GitHub API rate limit.
- Existing `exec` invocations that relied on arbitrary installation IDs or repository hosts will fail instead of silently bypassing configured policy. This is an intentional security correction and must be documented.
- The caller is responsible for protecting the token after it is printed. The extension cannot prevent shell variables, pipes, terminal scrollback, or redirected files from exposing it.
- Go strings and process memory cannot be perfectly zeroed; a compromised local OS or process remains out of scope.
- The current secret-storage identity is based on the configured display name. Duplicate names across distinct Apps remain a separate hardening concern and should be addressed before relying on names as immutable secret identities.

### Regression protection

Tests will cover:

- exact App/installation matching;
- rejection of unconfigured installation IDs;
- rejection of repositories outside configured App routes and scopes;
- rejection before authentication/network calls;
- Client ID selection;
- exact token-only stdout;
- fresh minting without cache population;
- cross-host redirect rejection;
- absence of token material from logs and errors.

## Follow-up actions

- [ ] Review and accept this ADR through the repository ADR process introduced by PR #59.
- [ ] Add the ADR to the ADR index when the process documentation is merged.
- [x] Update README and security documentation with the new command and stdout disclosure guidance.
- [x] Add changelog entry under `[Unreleased]`.
- [ ] Consider changing secret-storage identities from display names to immutable App/installation identifiers.

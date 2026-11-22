---
title: Security model
description: What gh-app-auth protects, where secrets live, who can still read them, and the limits of the design.
sidebar:
  order: 4
---

gh-app-auth handles private keys, personal access tokens and installation tokens. This page states
what it does about them and, just as important, what it does not. It makes no certification or
compliance claims.

To report a vulnerability, follow the project's
[security policy](../../.github/SECURITY.md).

## Where each secret lives

| Secret | Where | Protection |
| --- | --- | --- |
| App private key | OS keyring by default | Encrypted at rest by the operating system's keyring |
| App private key, filesystem source | A file you provide with `--key-file`, or a managed copy | Plain file; relies on file permissions |
| Automatic filesystem fallback | `~/.config/gh/extensions/gh-app-auth/secrets/<name>.<type>` | Raw bytes, created with mode `0400` (an existing writable file keeps its mode); **not encrypted** |
| PAT | OS keyring | Encrypted at rest by the keyring. Filesystem retrieval is not implemented |
| Installation token | Process memory | Never written to disk; 55-minute cache per process |
| JWT | Process memory | Short-lived, not cached |

The configuration file holds identifiers, names, patterns and priorities, not secret values.

## Key file permission checks

On POSIX systems, `setup` and JWT loading reject a key file whose permission bits include group or
world **read** (`0044`). Use `600` or `400`. The check does not cover symlinks, group write, or
changes made after the check, and it is not repeated every time a configured key is read from the path
stored in the configuration. On Windows the check is skipped, and you are responsible for file ACLs.

## Boundaries and exposure

- **Same user and root.** Anything running as your user, or as root, can read your keyring entries,
  your key files, the process memory of gh-app-auth, and the environment of its children. gh-app-auth
  does not defend against that.
- **Process memory.** Tokens are plain strings in process memory. Go strings are immutable, so
  zeroization is best effort and not guaranteed.
- **Child environment.** `exec` places a token in the child's environment, and every descendant of the
  child inherits it. Treat the child as trusted for the token's life.
- **Standard output.** `token` writes a token to standard output. Terminals, shell tracing, CI logs and
  pipelines can capture it. Prefer `exec`. This is a deliberate choice recorded in
  [ADR 0002](../adr/0002-explicit-github-app-token-output.md).
- **Process arguments.** `--pat` is a command-line argument, so a PAT can appear in the process list while
  `setup` runs.
- **Config path.** `GH_APP_AUTH_CONFIG` moves the configuration file, not the secrets directory.

## Tokens and scope

Patterns select a credential; they do not restrict it. The limits on what a token can do come from the
GitHub App installation's permissions and repository selection. Grant the minimum. See
[tokens and caching](#tokens-and-caching).

## Tokens and caching

Several clocks run at once in gh-app-auth. They are separate and should not be confused.

| Item | Lifetime | Where it lives |
| --- | --- | --- |
| App JWT | `exp` is 9 minutes ahead and `iat` is 60 seconds behind the current time | Created on demand, not cached |
| Installation token | Up to 1 hour, set by GitHub | Memory only |
| Installation token cache entry | 55 minutes, per process | Memory only |
| Installation scope metadata | 24 hours | The configuration file |
| PAT | Provider policy, expiry or revocation | OS keyring |

### JWT

To act as the App, gh-app-auth signs a JWT with the App's private key. The `iat` claim is set 60
seconds in the past to tolerate clock skew, and `exp` is 9 minutes in the future. A JWT is used to
request an installation token and is not stored. See the
[ADR on the clock-skew margin](../adr/0001-jwt-clock-skew-margin.md).

### Installation token

GitHub issues an installation token for one installation, valid for up to an hour. The helper
caches it in memory for 55 minutes, which leaves a margin before expiry.

- The cache exists only inside one process. Git starts a **fresh helper process for each request**,
  so each request normally mints a new token. The cache helps processes that stay alive and make
  several requests.
- A background worker removes expired entries. It does **not** renew them. There is no daemon and no disk
  cache.
- Tokens are never written to disk.

### exec and token

`exec` takes a snapshot of a token and gives it to its child. Nothing renews it, so a child that runs
longer than the token's life starts failing. `token` bypasses the cache and mints a fresh token each
time; its output goes to standard output, which is a secret boundary. See the
[exposure boundaries](#boundaries-and-exposure).

### Scope metadata

`gh app-auth scope` records which repositories an installation can see and caches that for 24
hours. It is metadata used for matching decisions; it does not extend or shorten any token's
life, and `scope --refresh` re-reads it.

### What caching does not change

- A cached token is still bound to its installation's permissions.
- The local cache has no independent revocation and receives no push notification. GitHub authorizes
  each request, and it may reject a token before the cache's 55 minutes are up. Suspending or uninstalling
  the installation is how access is blocked; see GitHub's
  [guide to reviewing and modifying installed GitHub Apps](https://docs.github.com/en/apps/using-github-apps/reviewing-and-modifying-installed-github-apps)
  and the [REST reference for revoking an installation access token](https://docs.github.com/en/rest/apps/installations#revoke-an-installation-access-token).
- Restarting a process starts with an empty cache.

## Logging

Diagnostic logging is off unless `GH_APP_AUTH_DEBUG_LOG` is set. When on, log lines record flow steps
and represent tokens by redacted metadata (type and length, not a cryptographic digest or unique identifier), rather than printing their values, and obvious secret-looking values are redacted. Logs still
contain command-line arguments, repository URLs and file paths, so review them before sharing. See
[diagnostics](../troubleshoot/index.md#diagnostics).

## Operational guidance

- Prefer a GitHub App over a PAT, and the keyring over files.
- In CI use disposable runners and the explicit-key-file layout from the [CI guide](../guides/ci/index.md).
- Rotate keys and PATs on a schedule, and remove credentials you no longer use; see
  [credential lifecycle](../guides/credentials.md#maintain-credentials).

## Supply-chain assurance

These indicators measure different things; none guarantees that the application is safe.

| Indicator | What it means |
| --- | --- |
| [OpenSSF Best Practices](https://www.bestpractices.dev/projects/16991) | Maintainer-reported practices and their justifications. An **in progress** badge is not a passing award. |
| [OpenSSF Scorecard](https://scorecard.dev/viewer/?uri=github.com/amadeusitgroup/gh-app-auth) | Automated repository-security heuristics, reported out of 10. Not a certification or a SLSA level. |
| SLSA Build L2, self-assessed | A conservative assessment of the release build provenance against the [SLSA v1.2 Build requirements](https://slsa.dev/spec/v1.2/build-requirements). Not an independent certification or a claim about the Source track. |
| [Provenance verification](https://github.com/AmadeusITGroup/gh-app-auth/actions/workflows/verify-release-provenance.yml?query=branch%3Amain) | The most recent automated check of published artifacts against the expected repository, signer, source commit and hosted-runner policy. Not an automated SLSA-level award. |

### SLSA assessment and evidence

The release process builds on GitHub-hosted runners and generates signed SLSA provenance in the
repository's attestation store, where consumers can retrieve and verify it. This supports a
**Build L2 self-assessment**. The assessment concerns build origin and integrity, not absence of
vulnerabilities, the security of dependencies, or permission to trust the application.

The assessment was checked against the published
[v0.2.0 release](https://github.com/AmadeusITGroup/gh-app-auth/releases/tag/v0.2.0), source commit
`8981f431700857cf5276f0edc092ca9818efc0c4`. All six binaries, four Linux packages and the RPM public
key were downloaded and passed cryptographic verification with the expected repository, attestation
workflow, source ref and commit, signer commit, and GitHub-hosted runner identity. The verified
statements' digests also match the release metadata.

**Build L3 is not claimed.** An attestation-only reusable workflow signs digests supplied by the
parent build job; that separation alone is not an assessment of L3's stronger unforgeability,
isolation and complete external-parameter requirements. A higher claim needs a documented trusted
builder and review of those guarantees. See GitHub's
[guidance on building and attesting in a reusable workflow](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/increase-security-rating).

### Automated checks and their limits

The [Scorecard workflow](../../.github/workflows/scorecard.yml) publishes results after upstream
`main` changes, on branch-protection changes, and weekly. Its badge needs a successful upstream
publication before a score is available; scans are not published from this project's forks.
Some checks depend on information accessible to the workflow's GitHub token, so interpret individual
findings and access limitations rather than treating the aggregate as a complete audit.
Action revisions are pinned by the [workflow lockfile](../../.github/workflows/actions.lock).
The Scorecard action's own container still uses a version tag rather than an image digest;
the lock tool reports that separate limitation.

The [provenance workflow](../../.github/workflows/verify-release-provenance.yml) checks the latest
published release daily, after either release workflow completes, and on manual dispatch from
upstream `main`. It runs after failed release workflows too, because a failure can occur after
publication. It is a read-only monitor, **not a pre-publication gate**; a green result records the
release checked in that run, not a promise about releases published afterwards.

The [verification policy](../../scripts/verify-release-provenance.sh) resolves the release tag to a
commit, downloads its assets without executing them, and verifies every asset except `checksums.txt`
using GitHub's SLSA provenance verifier. It requires this repository, the
`attest-release.yml` signer at that same commit, source ref `refs/heads/main`, and a GitHub-hosted
runner. The checksum manifest is auxiliary metadata covered by a different GitHub release-attestation
predicate, not by the build-provenance statement. The monitor verifies assets listed by the release API (except
`checksums.txt`); missing downloads for listed assets, metadata-only releases and verification failures fail.
It does not enforce the expected platform/package inventory: an asset omitted from API metadata can escape
this check. Release completeness is a separate check. This check does not establish all of L3's platform guarantees.

## Not covered

- Defense against malware or a compromised machine.
- Protection of a key file after it has been copied elsewhere.
- Any guarantee about third-party credential helpers configured next to gh-app-auth.

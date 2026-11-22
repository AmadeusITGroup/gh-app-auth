<p align="center"><img src="website/public/favicon.svg" width="112" height="112" alt="gh-app-auth routing key logo"></p>
<h1 align="center">gh-app-auth</h1>
<p align="center"><strong>The right identity. For every remote.</strong><br>A <a href="https://cli.github.com/">GitHub CLI</a> extension and Git credential helper for GitHub Apps and personal access tokens.</p>

<p align="center">
  <a href="https://github.com/AmadeusITGroup/gh-app-auth/actions/workflows/ci.yml?query=branch%3Amain+event%3Apush"><img src="https://github.com/AmadeusITGroup/gh-app-auth/actions/workflows/ci.yml/badge.svg?branch=main&amp;event=push" alt="CI status on upstream main" height="20"></a>
  <a href="https://github.com/AmadeusITGroup/gh-app-auth/actions/workflows/e2e.yml?query=branch%3Amain+event%3Apush"><img src="https://github.com/AmadeusITGroup/gh-app-auth/actions/workflows/e2e.yml/badge.svg?branch=main&amp;event=push" alt="End-to-end test status on upstream main" height="20"></a>
  <a href="https://github.com/AmadeusITGroup/gh-app-auth/actions/workflows/codeql.yml?query=branch%3Amain+event%3Apush"><img src="https://github.com/AmadeusITGroup/gh-app-auth/actions/workflows/codeql.yml/badge.svg?branch=main&amp;event=push" alt="CodeQL analysis status on upstream main" height="20"></a>
  <a href="https://app.codecov.io/gh/AmadeusITGroup/gh-app-auth/tree/main"><img src="https://img.shields.io/codecov/c/github/AmadeusITGroup/gh-app-auth/main?label=coverage" alt="Codecov coverage on upstream main" height="20"></a>
</p>

<p align="center">
  <a href="https://github.com/AmadeusITGroup/gh-app-auth/releases/latest"><img src="https://img.shields.io/github/v/release/AmadeusITGroup/gh-app-auth?label=release&amp;color=1d4f9c" alt="Latest published release" height="20"></a>
  <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/AmadeusITGroup/gh-app-auth?label=Go&amp;color=1d4f9c" alt="Go version required by go.mod" height="20"></a>
  <a href="https://github.com/AmadeusITGroup/gh-app-auth/releases"><img src="https://img.shields.io/github/downloads/AmadeusITGroup/gh-app-auth/total?label=asset%20downloads&amp;color=1d4f9c" alt="Total downloads of GitHub release assets" height="20"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/AmadeusITGroup/gh-app-auth?color=1d4f9c" alt="Repository license" height="20"></a>
</p>

<p align="center">
  <a href="https://www.bestpractices.dev/projects/16991"><img src="https://www.bestpractices.dev/projects/16991/badge" alt="OpenSSF Best Practices assessment progress" height="20"></a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/amadeusitgroup/gh-app-auth"><img src="https://api.scorecard.dev/projects/github.com/amadeusitgroup/gh-app-auth/badge" alt="OpenSSF Scorecard automated repository assessment" height="20"></a>
  <a href="docs/understand/security-model.md#slsa-assessment-and-evidence"><img src="https://img.shields.io/badge/SLSA%20Build-L2%20self--assessed-1d4f9c" alt="SLSA Build L2 self-assessed (not certified)" height="20"></a>
  <a href="https://github.com/AmadeusITGroup/gh-app-auth/actions/workflows/verify-release-provenance.yml?query=branch%3Amain"><img src="https://img.shields.io/github/actions/workflow/status/AmadeusITGroup/gh-app-auth/verify-release-provenance.yml?branch=main&amp;label=provenance" alt="Published-release provenance verification on upstream main" height="20"></a>
</p>

<p align="center">
  <a href="docs/start-here/first-clone.md">Quick start</a> ·
  <a href="docs/README.md">Documentation</a> ·
  <a href="docs/guides/ci/index.md">Use in CI</a> ·
  <a href="CONTRIBUTING.md">Contribute</a>
</p>

A GitHub App installation token belongs to one installation and expires after an hour. If you or
your pipelines touch repositories in several organizations, including submodules, you end up
juggling credentials. gh-app-auth registers each credential with the repository patterns it covers,
and Git asks it for credentials per request.

## What it does

- Registers GitHub Apps (Client ID or App ID, private key) and PATs together with repository patterns.
- Acts as a Git credential helper: it picks the credential for the remote URL and, for an App, mints an
  installation token.
- Runs commands with a token in their environment: `gh app-auth exec`. For a GitHub App the token is a
  short-lived installation token; for a PAT it is the stored token.
- Keeps installation tokens in memory only (55-minute cache per process) and private keys in the OS
  keyring by default.
- Supports github.com, GitHub Enterprise Server, and PATs for hosts such as Bitbucket.

It does not run a daemon, does not renew tokens in the background, and does not limit what a token can do:
the GitHub App installation's permissions do that.

## Install

```bash
gh extension install AmadeusITGroup/gh-app-auth
```

Add `--pin <release-tag>` to select a particular release, and upgrade with `gh extension upgrade app-auth`.

## A safe starting path

This registers one GitHub App that is already installed on your organization with **Contents: read**
access, and clones a private repository. It needs the GitHub CLI and Git. All values are placeholders.
The `chmod` command is for Linux and macOS; on Windows, restrict the key file with ACLs, because gh-app-auth
skips its permission check there.

```bash
chmod 600 "$HOME/keys/example-app.pem"
gh app-auth setup \
  --client-id Iv1.example \
  --key-file "$HOME/keys/example-app.pem" \
  --patterns 'github.com/example-org/' \
  --name 'Example App'
gh app-auth gitconfig --sync
gh app-auth list --verify-keys
git clone https://github.com/example-org/private-repo.git
```

The key is stored in the OS keyring by default. On a headless machine without a keyring, add
`--use-filesystem` together with `--key-file`; do not rely on the automatic filesystem fallback. The
[first clone tutorial](docs/start-here/first-clone.md) explains each step and what to expect.

To call the GitHub API with the same credential, prefer `exec` over printing a token. Name the repository
explicitly, because `exec` otherwise uses the repository of your current directory:

```bash
gh app-auth exec --repo github.com/example-org/private-repo -- gh api repos/example-org/private-repo --jq .full_name
```

## Documentation

The documentation sources are in [`docs/`](docs/README.md). A documentation site is built from them
with Astro Starlight in [`website/`](website/package.json), and the documentation workflow publishes it to GitHub
Pages from `main`. You can also read the files on GitHub through the table below.

| If you want to | Read |
| --- | --- |
| Get started | [Get started](docs/start-here/first-clone.md) |
| Do a task | [Manage credentials](docs/guides/credentials.md), [Git integration](docs/guides/git-credential-helper.md), [CI](docs/guides/ci/index.md) |
| Look something up | [CLI reference](docs/reference/cli/index.md), [configuration and environment](docs/reference/configuration.md) |
| Understand it | [Credential routing](docs/understand/credential-routing.mdx), [security model](docs/understand/security-model.md), [architecture](docs/understand/architecture.mdx) |
| Fix a problem | [Troubleshooting](docs/troubleshoot/index.md) |
| Contribute | [Contributing](CONTRIBUTING.md), [development](docs/contribute/development.md), [release runbook](docs/contribute/releasing.md) |

## Known limitations

- The automatic filesystem fallback for App keys can leave a configuration that is rejected later.
  Use `--use-filesystem --key-file`.
- A PAT cannot be read from the filesystem store, so PATs need an OS keyring.
- `exec` gives its child a token snapshot that is not renewed during the run.
- The `--config` and `--debug` flags are not wired as their help suggests; use `GH_APP_AUTH_CONFIG`
  and `GH_APP_AUTH_DEBUG_LOG` (with an absolute file path).

Details and alternatives are in [known limitations](docs/troubleshoot/index.md#known-limitations).

## Contributing and security

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and the pull request process, and the
[documentation guide](docs/contribute/documentation.md) for working on the docs. Report vulnerabilities
privately as described in the [security policy](.github/SECURITY.md).

## License

MIT License. See [LICENSE](LICENSE) for details.

---
title: Development
description: Set up a development environment, build and test gh-app-auth, and follow the project's code and security conventions.
sidebar:
  order: 2
---

## Prerequisites

- Go matching [go.mod](../../go.mod)
- The GitHub CLI (`gh`) and Git
- `make`
- Node.js 24 only if you work on the documentation site

## Build and run

```bash
git clone https://github.com/AmadeusITGroup/gh-app-auth.git
cd gh-app-auth
go mod download
go build -o gh-app-auth .
gh extension install .
```

## Everyday commands

| Command | Purpose |
| --- | --- |
| `make test` | Run the test suite; run it after changing Go files |
| `go test -race ./...` | Run with the race detector |
| `go test -v ./pkg/auth/...` | Run one package verbosely |
| `make fmt` | Format code; run it before committing |
| `make lint` | The full golangci-lint configuration, including `lll` line length; run it as well as `make quality` |
| `make quality` | Lint, coverage threshold and security scan |
| `make ci` | Simulate the CI pipeline locally |
| `make security-scan` | Run the security scanners |

## Project structure

```text
cmd/        Cobra commands, including the Git credential helper
pkg/auth    GitHub App authentication and token exchange
pkg/cache   In-memory installation token cache
pkg/config  Configuration loading and validation
pkg/jwt     JWT creation
pkg/matcher URL pattern matching for GitHub Apps
pkg/secrets Keyring and file secret storage
pkg/scope   Installation scope metadata
pkg/logger  Diagnostic logging
test/       Integration and end-to-end tests
docs/       Documentation (the source of the site)
website/    The documentation site shell
```

## Conventions

- Wrap errors with context: `fmt.Errorf("failed to load config from %s: %w", path, err)`.
- Use `fmt.Println` for static strings and `fmt.Printf` only when formatting values.
- Never log tokens in clear text; log redacted secret metadata (`logger.HashToken` is a legacy name for redaction, not a digest). Never hardcode credentials or commit test keys.
- Validate key file permissions, and prefer the OS keyring.
- Use `t.TempDir()` and `t.Setenv()` for test isolation.
- Follow the commit message format in [CONTRIBUTING.md](../../CONTRIBUTING.md).

## Security-sensitive areas

Take extra care, and ask the maintainers first, when you change `pkg/secrets`, `pkg/auth`, `pkg/jwt`,
the Git credential helper in `cmd/git-credential.go`, the configuration file format or the CI
workflows. See the [security model](../understand/security-model.md) for current limits and
[architecture decisions](#architecture-decisions) for the recorded rationale.

## Changes that need more than code

A new command needs a test file, registration in `cmd/root.go` and a regenerated CLI reference page.
Run `npm run reference:generate` in `website/` after changing any command or flag, so the generated
reference matches. See the [documentation guide](documentation.md).

## Architecture decisions

Significant technical decisions are recorded as numbered, immutable Architecture Decision Records (ADRs).
The [ADR index](../adr/README.md) holds the template, naming rules, status lifecycle and the list of
records with their actual statuses. Write an ADR for breaking changes, new features that introduce a
concept or trust boundary, security decisions, technology choices, significant refactoring and rejected
alternatives worth recording. Bug fixes, test additions, documentation-only changes and formatting do not
need one. Never rewrite an accepted ADR; supersede it with a new one.

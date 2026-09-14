# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Add `gh app-auth token` for fresh, explicitly selected GitHub App installation tokens.
- Add `--client-id` selector to `gh app-auth exec`.
- Add a cross-platform E2E pipeline (`e2e.yml`) that installs and exercises the
  real release assets on Linux (deb/rpm, amd64+arm64), macOS (arm64+intel), and
  Windows (amd64+arm64), including cross-org submodule clones.
- Add `scripts/next-version.sh` and SLSA build-provenance attestation
  (`attest-release.yml`) to the release pipeline.

### Changed

- **Release process**: releases are now triggered by `workflow_dispatch` (or a
  `v*` tag push) instead of prerelease creation. The pipeline stages assets on a
  **draft** release, gates publication on the E2E suite and attestation, then
  publishes the draft as latest. This is compatible with GitHub's immutable
  releases, which lock assets at publish — the previous prerelease-promotion
  model would fail with `HTTP 422` once enabled. See `docs/RELEASE_PROCESS.md`.
  **Breaking change** for the maintainer release runbook: create releases via
  `gh workflow run release.yml`, not `gh release create --prerelease`.

- `gh app-auth exec` explicit selectors now fail closed: `--repo` must match the
  selected App's configured route and `--installation-id` must be a configured
  installation of that App. Previously a single App-ID match bypassed repository
  validation and `--installation-id` could override the configured installation.
  **Breaking change** for invocations that relied on those behaviours.

### Fixed

- `gh app-auth setup` rejects malformed App patterns instead of silently
  creating no configuration entry.

### Security

- Enforce configured repository routes and exact installation IDs for explicit App selectors.
- Bound authenticated redirects: refuse cross-host redirects and stop after 10 hops.

[Unreleased]: https://github.com/AmadeusITGroup/gh-app-auth/compare/v1.0.0...HEAD

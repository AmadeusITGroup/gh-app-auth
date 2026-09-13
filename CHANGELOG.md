# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Add `gh app-auth token` for fresh, explicitly selected GitHub App installation tokens.
- Add `--client-id` selector to `gh app-auth exec`.

### Changed

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

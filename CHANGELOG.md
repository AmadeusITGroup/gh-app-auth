# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0](https://github.com/AmadeusITGroup/gh-app-auth/compare/v0.0.17...v0.1.0) (2026-10-05)


### ⚠ BREAKING CHANGES

* **cli:** gh app-auth exec explicit selectors now fail closed: --repo must match the selected App's configured route and --installation-id must be a configured installation of that App. Invocations that relied on unconfigured repository hosts or installation IDs now return an error.

### Features

* add RPM/DEB package builds for amd64, arm64 ([#37](https://github.com/AmadeusITGroup/gh-app-auth/issues/37)) ([8aab024](https://github.com/AmadeusITGroup/gh-app-auth/commit/8aab0245ee6928dad96066dfe34ed487fdb9b411))
* auto-detect installation ID and add config command ([6734926](https://github.com/AmadeusITGroup/gh-app-auth/commit/673492603b760fbc49d6b14f199129b93266d8f7))
* auto-detect mode for app installation id ([ca0b4ce](https://github.com/AmadeusITGroup/gh-app-auth/commit/ca0b4ce131c05cc0faab093123c81cfe14f1b9c6))
* Automatic setup of GitHub App during git clone ([fd8ba89](https://github.com/AmadeusITGroup/gh-app-auth/commit/fd8ba89d4f25d9bb638470d2a77219c708b63e2f))
* Automatic setup of GitHub App during git clone ([8f640ac](https://github.com/AmadeusITGroup/gh-app-auth/commit/8f640ac472c890d23ab3bef70e3e262d57ad8850))
* **cli:** add token command ([#62](https://github.com/AmadeusITGroup/gh-app-auth/issues/62)) ([544775e](https://github.com/AmadeusITGroup/gh-app-auth/commit/544775ee31e7d5bfd208eb03779d26e3a34fb1e6))
* **cli:** support GitHub App Client ID ([#53](https://github.com/AmadeusITGroup/gh-app-auth/issues/53)) ([59cb28a](https://github.com/AmadeusITGroup/gh-app-auth/commit/59cb28aa6f14bf80fe6286a8d50c24feff85f9dd))
* enable useHttpPath for path-specific patterns ([3656d03](https://github.com/AmadeusITGroup/gh-app-auth/commit/3656d03d59cc359c6c1fe1cc02460c0aa5dd97c8))
* GitHub CLI extension for GitHub App and PAT authentication ([6df981b](https://github.com/AmadeusITGroup/gh-app-auth/commit/6df981b046cbfabe872061841385f92c37efd7a5))
* run commands with app authentication ([#51](https://github.com/AmadeusITGroup/gh-app-auth/issues/51)) ([d959ae1](https://github.com/AmadeusITGroup/gh-app-auth/commit/d959ae10591f6e2f00bca30b32c6a9b8f81e6afe))


### Bug Fixes

* **auth:** leave clock skew margin on JWT exp and iat claims ([#58](https://github.com/AmadeusITGroup/gh-app-auth/issues/58)) ([620f73d](https://github.com/AmadeusITGroup/gh-app-auth/commit/620f73d8e27a81ea5736acbf5643b461da61c0f4))
* **ci:** handle fork PRs and edge cases in workflows ([#30](https://github.com/AmadeusITGroup/gh-app-auth/issues/30)) ([a3b8475](https://github.com/AmadeusITGroup/gh-app-auth/commit/a3b84757eb650486c5508dea6c3c2ac2961f8146))
* **config:** split multi-pattern entries with unique installation IDs per org ([#35](https://github.com/AmadeusITGroup/gh-app-auth/issues/35)) ([601632a](https://github.com/AmadeusITGroup/gh-app-auth/commit/601632a86f197294ad294981dd30f3ff0e4b35f7)), closes [#34](https://github.com/AmadeusITGroup/gh-app-auth/issues/34)
* create draft release before building assets ([8fb03f5](https://github.com/AmadeusITGroup/gh-app-auth/commit/8fb03f5d324dc2f8268ef9ad1e79cb58858a1255))
* handle existing releases in release workflow ([76890af](https://github.com/AmadeusITGroup/gh-app-auth/commit/76890af085b4982ac3fcbc1cd6e10c13ba1cc412))
* reorder credential helpers for correct precedence ([16af226](https://github.com/AmadeusITGroup/gh-app-auth/commit/16af226b0dddc38c6b4bc063047733b5dac1240e)), closes [#20](https://github.com/AmadeusITGroup/gh-app-auth/issues/20)
* resolve git credential helper precedence for path-specific patterns ([5aa0f3f](https://github.com/AmadeusITGroup/gh-app-auth/commit/5aa0f3f3d0881bdb92817590c1e536f0ef4583b8))
* restoring NFPM_CMD which was removed by mistake ([#57](https://github.com/AmadeusITGroup/gh-app-auth/issues/57)) ([055ff64](https://github.com/AmadeusITGroup/gh-app-auth/commit/055ff64a8995ac7021917b77e5c454ecf1ba9636))
* **security:** address CodeQL security alerts in diagnostic logging ([#28](https://github.com/AmadeusITGroup/gh-app-auth/issues/28)) ([570e7b0](https://github.com/AmadeusITGroup/gh-app-auth/commit/570e7b01ad041de9809dbf759e40d04a557be200))
* **security:** implement multi-layered sensitive data redaction ([#29](https://github.com/AmadeusITGroup/gh-app-auth/issues/29)) ([9982156](https://github.com/AmadeusITGroup/gh-app-auth/commit/99821564bc543f836df8b4743daf20b908c1a5c1))
* When both filesystem+env var are set, no app could be configured anymore ([60e7336](https://github.com/AmadeusITGroup/gh-app-auth/commit/60e7336ebf0b4317497887e9de548709f2b5283a))
* When both filesystem+env var are set, no app could be configured anymore ([9a35515](https://github.com/AmadeusITGroup/gh-app-auth/commit/9a355150253643bb4446c87ca8f6c0e0b2f98367))
* Windows test compatibility and security workflow ([f9af0be](https://github.com/AmadeusITGroup/gh-app-auth/commit/f9af0becc8e5b75d595ad2d8c265d3ec68e1e54a))

## [Unreleased]

<!-- Sections below are generated by release-please from conventional commits.
     Curated prose can be hand-edited into the release PR before merging. -->

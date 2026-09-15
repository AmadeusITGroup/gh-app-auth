# Release Process

How to publish a new version of `gh-app-auth`.

The short version: **dispatch the `Release` workflow.** It resolves the version,
creates the tag and a *draft* release, builds and attaches every asset, gates on
the cross-platform E2E suite and SLSA attestation, then publishes the draft as
latest. Nothing is publicly visible until the final publish step — the draft is
the staging area, not a prerelease.

## TL;DR

```bash
# 1. Make sure main is green and CHANGELOG.md is updated
git checkout main && git pull

# 2. Dispatch the workflow (auto-detects the semver bump from conventional
#    commits; pass an explicit version or bump to override)
gh workflow run release.yml                    # bump=auto, ref=main
gh workflow run release.yml -f bump=minor
gh workflow run release.yml -f version=v1.2.3
gh workflow run release.yml -f dry_run=true    # everything except publish

# 3. Watch the pipeline
gh run watch

# 4. Verify
gh release view v1.2.3
```

Equivalent UI path: **Actions → Release → Run workflow**.

Alternative trigger: push a `v*` tag by hand (`git tag v1.2.3 && git push origin
v1.2.3`) — the same pipeline stages the draft on it. This is also the fallback if
the tag already exists.

## Automated trigger (release-please)

`release-please.yml` runs on every push to `main` and maintains a **release PR**
that accumulates conventional-commit changelog entries and the proposed version
bump. The maintainer's release action becomes **"merge the release PR"**:

```
merge release PR → release-please cuts tag + draft → workflow_call → release.yml
```

- release-please creates the release as a **draft** (`"draft": true` in
  `release-please-config.json` — mandatory: a published release would lock its
  assets under immutable releases before the pipeline could attach them).
- The handoff is an explicit `workflow_call`, not an event — tags created by
  `GITHUB_TOKEN` never fire `push: tags`.
- Version selection then comes entirely from conventional commits;
  `.release-please-manifest.json` tracks the current version and `version.txt`
  is bumped by the release PR.
- While the project is `0.x`, `bump-minor-pre-major` keeps breaking changes on a
  minor bump rather than jumping to `1.0.0`.

The manual dispatch above remains the override — same pipeline either way.

## Pipeline shape

```
prepare → build → [e2e ∥ attest] → publish
(version,  (assets   (gates must    (draft →
 tag,       to the    both pass)     latest)
 draft)     draft)
```

| Job | What it does |
|-----|--------------|
| `prepare` | Resolves the version (`scripts/next-version.sh`), creates the git tag, creates or reuses the **draft** release |
| `build` | Checks out the tag, runs unit tests, `make release packages`, writes `checksums.txt`, uploads `dist/*` to the draft with `--clobber` |
| `e2e` | Calls `e2e.yml` — the 10-job matrix validates the draft's real assets on Linux (deb/rpm, amd64+arm64), macOS (arm64+intel), Windows (amd64+arm64) |
| `attest` | Calls `attest-release.yml` — generates SLSA build provenance for every asset digest, in an isolated reusable workflow |
| `publish` | `gh release edit --draft=false --latest`, then verifies the release and its attestation |
| `report-failure` | On any failure, summarizes that the release stayed a draft and how to re-run |

## Why draft-first (and not prerelease)

Two reasons, both verified on the fork:

1. **Zero public exposure window.** Drafts are invisible to everyone without
   push access — a failed gate leaves an invisible draft, not a public
   prerelease with missing assets. `gh extension install` only sees `latest`.
2. **Immutable releases.** GitHub's "Make new releases immutable" locks assets
   and the tag *at publish* — a published prerelease is already locked
   (`HTTP 422: Cannot upload assets to an immutable release`). Drafts are the
   only unconditionally mutable state; all asset mutation happens before
   publish.

Draft specifics that shaped the pipeline:

- **Drafts do not create the git tag** — it materializes at publish. `prepare`
  creates the tag explicitly so `build` can check it out and
  `git describe --tags --exact-match` (which feeds `LDFLAGS` in the Makefile)
  resolves.
- **Downloading draft assets needs `contents: write`** — a read-only
  `GITHUB_TOKEN` gets `release not found`. The e2e jobs carry `contents: write`
  for this reason; they never mutate the release.

## Version resolution (`scripts/next-version.sh`)

| Input | Behaviour |
|-------|-----------|
| `version` set | Validated as semver; must be newer than the latest `v*` tag and not already exist |
| `bump=auto` | Scans conventional commits since the latest tag: `BREAKING CHANGE`/`!:` → major (**minor while `0.x`**, matching release-please's `bump-minor-pre-major`), `feat` → minor, `fix`/`perf`/`revert`/`deps` → patch; aborts if nothing releasable |
| `bump=patch\|minor\|major` | Applied directly |
| `release_tag` (workflow_call) | Used as-is after validation — this is the release-please path |

## Failure and recovery

- **Any gate fails** → the release stays a draft; nothing is public. Fix the
  cause and re-dispatch with the same version — the pipeline is idempotent:
  the existing draft is reused and uploads are `--clobber`ed.
- **Tag exists at the wrong commit** → `prepare` aborts rather than re-tag.
- **Version already published** → `prepare` aborts; published releases are
  never touched.
- **A bad release is already published** → immutable releases mean no
  post-publish fixes; cut the next patch release. Drafts can be deleted freely.

## What the workflow does (build detail)

| Step | Command | Purpose |
|------|---------|---------|
| Checkout | `actions/checkout` at the tag, `fetch-depth: 0` | Full history + the tag so `git describe --tags --exact-match` resolves the version |
| Set up Go | `actions/setup-go` with `go-version-file: go.mod` | Toolchain matches go.mod |
| Test | `go test ./...` | Release gate — a failing test aborts the release |
| Build | `make release packages` | Produces every asset in `dist/` |
| Checksums | `sha256sum` of `dist/*` → `dist/checksums.txt` | Attestation manifest + manual verification |
| Upload | `gh release upload "$TAG" dist/* --clobber` | Attaches **everything** in `dist/` (idempotent re-runs) |
| Attest | `actions/attest-build-provenance` with `subject-checksums` | SLSA provenance in the repo attestation store |
| Promote | `gh release edit "$TAG" --draft=false --latest` | Makes the release installable |

Two environment values drive the version:

- `VERSION` — derived by the Makefile from `git describe --tags --exact-match`
  (e.g. `v1.2.3`); `PKG_VERSION` strips the leading `v` for package filenames
  and metadata.
- `RPM_RELEASE=1` — the RPM release/revision number. Bump it manually only if you need to rebuild
  the same upstream version as a new RPM.

`make release` itself derives `VERSION` from `git describe --tags --exact-match`, which is why the
tag must exist and point at the checked-out commit. Without an exact tag match the version falls
back to the string `dev` and the binaries report `dev` from `--version`.

## Assets built during the release

`make release packages` writes everything into `dist/`, and the upload step attaches the whole
directory. For version `1.2.3` you should see 10 assets: 6 binaries and 4 Linux packages.

### Cross-platform binaries (`make release`)

Built from `BUILD_MATRIX` in the `Makefile` with `CGO_ENABLED=0` for static, dependency-free
binaries:

| Asset | GOOS/GOARCH |
|-------|-------------|
| `linux-amd64` | linux/amd64 |
| `linux-arm64` | linux/arm64 |
| `darwin-amd64` | darwin/amd64 |
| `darwin-arm64` | darwin/arm64 |
| `windows-amd64.exe` | windows/amd64 |
| `windows-arm64.exe` | windows/arm64 |

**The filenames are not cosmetic.** `gh extension install` looks for release assets named exactly
`<goos>-<goarch>[.exe]` to detect a precompiled extension. Renaming them (for example to
`gh-app-auth-linux-amd64`) breaks `gh extension install AmadeusITGroup/gh-app-auth`, which would
silently fall back to a source build or fail. If you add a platform, add it to `BUILD_MATRIX` and
keep the naming convention.

Each binary is stamped through `LDFLAGS`:

```makefile
-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)
```

### Linux packages (`make packages`)

Built with [nFPM](https://nfpm.goreleaser.com/) from the templated `nfpm.yaml`. `envsubst`
substitutes `${ARCH}`, `${GOARCH}`, `${VERSION}` and `${RPM_RELEASE}` before each `nfpm pkg` call:

| Asset | Packager | Arch |
|-------|----------|------|
| `gh-app-auth_1.2.3_amd64.deb` | deb | amd64 |
| `gh-app-auth_1.2.3_arm64.deb` | deb | arm64 |
| `gh-app-auth_1.2.3-1_x86_64.rpm` | rpm | x86_64 |
| `gh-app-auth_1.2.3-1_aarch64.rpm` | rpm | aarch64 |

Package contents (from `nfpm.yaml`):

- `/usr/bin/gh-app-auth` (mode `0755`, `root:root`)
- `/usr/share/doc/gh-app-auth/LICENSE`
- `/usr/share/doc/gh-app-auth/README.md`
- `Depends: git`, `Recommends: gh`

Note the differing version conventions: DEB uses `1.2.3`, RPM appends the release number as
`1.2.3-1`.

### Known gap: no armhf packages

`make help` advertises `package-deb-arm` and `package-rpm-arm` (32-bit arm/armhf), and `packages`
lists them as prerequisites. Neither target has a recipe — they exist only in the `.PHONY` list, so
make treats them as satisfied and silently does nothing. **No armhf packages are produced**, and
`make packages` still reports "All packages built successfully!". If armhf support is needed, add
`linux-arm` to `BUILD_MATRIX` and write the two missing targets.

## Release checklist

Before dispatching the workflow:

- [ ] `main` is green in CI (test matrix, lint, security, CodeQL)
- [ ] `make quality` passes locally
- [ ] `CHANGELOG.md` has a section for the new version (move items out of `[Unreleased]`, add the
      compare link at the bottom)
- [ ] Version number follows [Semantic Versioning](https://semver.org/) and matches the commit types
      since the last tag: breaking change → major (minor while `0.x`), `feat` → minor, `fix` → patch
- [ ] Any breaking change or significant new feature has an [ADR](adr/README.md)
- [ ] Docs (`README.md`, `docs/`) reflect new or changed commands and flags

## Verifying a release

```bash
# All 10 assets present?
gh release view v1.2.3 --json assets --jq '.assets[].name'

# Release is published (not a draft)?
gh release view v1.2.3 --json isDraft,isPrerelease

# Extension install picks up the precompiled binary
gh extension install AmadeusITGroup/gh-app-auth
gh app-auth --version   # should print 1.2.3, not "dev"

# Package installs cleanly
sudo dpkg -i gh-app-auth_1.2.3_amd64.deb   # or: sudo rpm -i ...
gh-app-auth --version
```

Locally, `make validate-packages` cross-checks that each binary and package really carries the
architecture its filename claims.

## Building assets locally

Useful to reproduce a release build or debug a packaging failure:

```bash
# Binaries for all platforms (runs `clean` first — wipes dist/)
make release

# Everything the release workflow builds
VERSION=1.2.3 RPM_RELEASE=1 make release packages

# Only your own architecture (fastest)
make packages-local

# Confirm architectures match filenames
make validate-packages
```

`make release` depends on `clean`, which removes `dist/`, the local `gh-app-auth` binary, and
coverage files. `make packages` also runs `dev-setup`, which sets your local
`git config commit.template`.

Packaging needs `envsubst` (`gettext-base` on Debian/Ubuntu, `gettext` on Fedora/RHEL). `nfpm` is
fetched on demand via `go run`, so no separate install is required.

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `prepare` fails: "tag exists at a different commit" | The tag was already pushed/published at another commit | Pick a new version, or delete the stray tag deliberately |
| `prepare` fails: "already published" | That version was already released | Bump the version — published releases are never touched |
| E2E fails mid-pipeline | Infra issue or real regression | Release stays a draft; fix and re-dispatch the same version |
| Binaries report version `dev` | `git describe --tags --exact-match` found no tag on HEAD, or checkout lacked full history | Ensure the tag points at the released commit and `fetch-depth: 0` is set |
| `envsubst: command not found` | Missing `gettext-base` | Install it (`gettext-base` on Debian/Ubuntu) |
| `gh extension install` builds from source instead of downloading | Asset names do not match `<goos>-<goarch>` | Restore the `BUILD_MATRIX` naming |
| Upload rejected as duplicate | Asset already attached from an earlier run | Already handled by `--clobber`; if editing manually, delete the asset first |
| Wrong architecture inside a package | `nfpm.yaml` templating or `GOARCH` mismatch | Run `make validate-packages` to locate the mismatch |

## Related documentation

- [`.github/workflows/release.yml`](../.github/workflows/release.yml) — the workflow itself
- [`Makefile`](../Makefile) — `release`, `packages`, `packages-local`, `validate-packages` targets
- [`nfpm.yaml`](../nfpm.yaml) — DEB/RPM package definition
- [CONTRIBUTING.md](../CONTRIBUTING.md) — conventional commits, which drive version selection
- [Installation Guide](installation.md) — how users consume these assets
- [ADR index](adr/README.md) — record the decisions behind notable changes in a release

# Release Process

How to publish a new version of `gh-app-auth`.

The normal path is **merge the release-please PR**. That starts the `Release`
workflow, which stages the tag and creates a *draft* release, builds and attaches
every asset, gates on the cross-platform E2E suite and provenance, then publishes
the draft as latest. Nothing is publicly visible until the final publish step —
the draft is the staging area, not a prerelease.

## TL;DR

```bash
# Normal releases: review and merge the release-please PR.

# Manual override / pipeline validation: run explicitly against main and
# provide either a version or an operator-selected bump.
gh workflow run release.yml --ref main -f bump=minor
gh workflow run release.yml --ref main -f version=v1.2.3
gh workflow run release.yml --ref main -f version=v1.2.3 -f dry_run=true

# Watch and verify
gh run watch
gh release view v1.2.3
```

Equivalent UI path: **Actions → Release → Run workflow**, selecting `main`.
The workflow rejects dispatches whose selected ref is not `main`; there is no
tag-push release trigger. Protect `main` with required review so the workflow
and release source can only change through reviewed commits. Tags are staged
and published only by the pipeline. The YAML guard prevents accidental
wrong-ref runs; it is not a substitute for protecting workflow files. Require
review for changes under `.github/workflows/` on `main` and limit manual release
execution to trusted maintainers.

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
- The handoff is an explicit `workflow_call`, not a tag event; the release
  workflow is only called from a run on `main`.
- Release-please is authoritative for automatic version selection from
  conventional commits. `.release-please-manifest.json` tracks its current
  version and the release PR updates the generated changelog.
- Manual dispatch is an explicit operator override: supply `version` or
  `bump=patch|minor|major`. It does not independently interpret conventional
  commits; after an emergency manual release, reconcile the manifest and
  changelog through the release-please PR before the next normal release
  (run `gh workflow run release-please.yml --ref main` to refresh it if needed).
- While the project is `0.x`, `bump-minor-pre-major` keeps breaking changes on a
  minor bump rather than jumping to `1.0.0`.
- **Post-merge regeneration quirk (observed on fork)**: immediately after a
  release PR merges, release-please updates its standing PR *before* the
  pipeline publishes — so the just-cut tag does not exist yet and the
  regenerated PR mis-anchors on the previous tag, showing a noisy changelog and
  a speculative next version. It self-corrects on the next push to `main` once
  the tag has materialized. Do not merge a freshly regenerated release PR;
  leave it open and let it recompute.
- **Tag names are one-shot under immutable releases**: deleting a *published*
  release does not free its `tag_name` — the name is permanently bound and any
  later release reusing it fails at publish (`tag_name was used by an
  immutable release`). Never re-cut a released version; bump instead.
- Configure a `v*` tag ruleset as defense in depth: block tag updates and
  deletion, while allowing the release workflow to create the tag at publish.
  This is repository configuration, not enforced by the workflow YAML.

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
| `prepare` | Resolves the version (`scripts/next-version.sh`), stages the git tag (local ref — remote materializes at publish), creates or reuses the **draft** release |
| `build` | Checks out the release commit and tags it locally, runs unit tests, `make release packages`, writes `checksums.txt`, uploads `dist/*` to the draft with `--clobber` |
| `e2e` | Calls `e2e-release.yml`; one write-scoped staging job downloads the draft assets, then ten read-scoped jobs test per-platform Actions artifact copies of those exact bytes |
| `attest` | Calls `attest-release.yml` — generates GitHub build provenance for every asset digest, in an isolated reusable workflow; provenance records origin/integrity, not artifact safety |
| `publish` | `gh release edit --draft=false --latest`, then verifies the release and its attestation |
| `report-failure` | On failure, tells maintainers to check draft/published state before retrying |

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

- **Drafts do not create the git tag** — it materializes at publish. With
  immutable releases enabled the draft *binds* the tag name: pushing the ref
  is evaluated as an update and rejected even for admins (verified on fork).
  `prepare`/`build` therefore tag the release commit **locally** so
  `git describe --tags --exact-match` (which feeds `LDFLAGS` in the Makefile)
  resolves; `publish` materializes the remote tag at `target_commitish`.
- **Downloading draft assets needs `contents: write`** — a read-only
  `GITHUB_TOKEN` gets `release not found`. One staging job downloads the assets
  and uploads per-platform Actions artifacts; the ten E2E jobs use those exact
  downloaded bytes with `contents: read`. Only the staging job has write access;
  the test jobs still receive the dedicated E2E App credentials for the test
  step, so those credentials must remain limited to the test organizations.

## Version resolution (`scripts/next-version.sh`)

| Input | Behaviour |
|-------|-----------|
| `version` set | Validated as semver; must be newer than the latest `v*` tag and not already exist |
| `bump=patch\|minor\|major` | Explicit operator selection, applied to the latest tag; requires a prior release tag |
| no version or bump | Rejected — release-please owns automatic conventional-commit version selection |
| `release_tag` (workflow_call) | Used as-is after validation — this is the release-please path |

## Failure and recovery

- **Failure before publish** → the release stays a draft; nothing is public.
  Fix the cause and re-run with the same version and source commit — the draft
  is reused and uploads are `--clobber`ed.
- **Tag exists at the wrong commit** → `prepare` aborts rather than re-tag.
- **Version already published** → `prepare` aborts; published releases are
  never touched.
- **Failure after publish** → immutable release contents and tag cannot be
  repaired or reused. Cut the next version (normally the next patch); do not
  retry the published version. Drafts that have not been published can be
  deleted and recreated.

## What the workflow does (build detail)

| Step | Command | Purpose |
|------|---------|---------|
| Checkout | `actions/checkout` at the main event SHA, `fetch-depth: 0` | Trusted source commit plus history for version validation; build stages the release tag locally before compiling |
| Set up Go | `actions/setup-go` with `go-version-file: go.mod` | Toolchain matches go.mod |
| Test | `go test ./...` | Release gate — a failing test aborts the release |
| Build | `make release packages` | Produces every asset in `dist/` |
| Checksums | `sha256sum` of `dist/*` → `dist/checksums.txt` | Attestation manifest + manual verification |
| Upload | `gh release upload "$TAG" dist/* --clobber` | Attaches **everything** in `dist/` (idempotent re-runs) |
| Attest | `actions/attest-build-provenance` with `subject-checksums` | GitHub build provenance in the repository attestation store; origin/integrity evidence, not a safety verdict |
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

## Release checklist

Before merging the release-please PR (or running an explicit manual override):

- [ ] `main` is green in CI (test matrix, lint, security, CodeQL)
- [ ] `make quality` passes locally
- [ ] Review the release-please PR's generated `CHANGELOG.md` section and edit release notes there if needed
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
| `prepare` fails: "tag exists at a different commit" | The version is already bound to another source commit | Pick a new version; never move a published tag |
| `prepare` fails: "already published" | That version was already released | Bump the version — published releases are never touched |
| E2E fails before publish | Infra issue or real regression | Release stays a draft; fix and re-run the same version from the same main commit |
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

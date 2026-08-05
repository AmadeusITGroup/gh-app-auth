# Release Process

How to publish a new version of `gh-app-auth`.

The short version: **create a GitHub *pre-release* on a `vX.Y.Z` tag.** The
[`Release` workflow](../.github/workflows/release.yml) then builds every asset, uploads them to that
release, and finally flips it to a normal "latest" release. Nothing is built for a release created
directly as final — the workflow would never run.

## TL;DR

```bash
# 1. Make sure main is green and CHANGELOG.md is updated
git checkout main && git pull

# 2. Create an annotated tag
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3

# 3. Create the release AS A PRE-RELEASE (this is the trigger)
gh release create v1.2.3 --prerelease --generate-notes --title "v1.2.3"

# 4. Watch the workflow build and attach the assets
gh run watch

# 5. Verify
gh release view v1.2.3
```

After step 4 the release is no longer a pre-release: the workflow promotes it with
`gh release edit --prerelease=false --latest`.

## Why the pre-release step exists

`release.yml` is triggered by `on: release: types: [prereleased]`.

```yaml
on:
  release:
    types: [prereleased]
```

This gives a two-phase release:

1. **Pre-release phase** — the release exists and is visible, but marked as a pre-release, so
   `gh extension install` and `gh extension upgrade` ignore it. The workflow runs tests, builds all
   binaries and packages, and uploads them.
2. **Promotion phase** — once every asset is uploaded, the same workflow clears the pre-release flag
   and marks the release as `--latest`.

The consequence is that users never see a "latest" release with missing or partial assets. A release
either has no assets and is a pre-release, or it has all assets and is latest.

The practical rules that follow:

- **Always create the release as a pre-release.** `gh release create v1.2.3` without `--prerelease`
  publishes a release with zero assets and no build. `gh extension install` will fail for everyone.
- **Do not manually unmark the pre-release flag.** The workflow does that as its last step.
- **Re-running the workflow is safe.** Uploads use `--clobber`, so assets are overwritten. To re-run
  after a failure, either re-run the failed job from the Actions UI, or mark the release back to
  pre-release (`gh release edit v1.2.3 --prerelease`) and re-publish to fire the event again.

## What the workflow does

| Step | Command | Purpose |
|------|---------|---------|
| Checkout | `actions/checkout@v7` with `fetch-depth: 0` | Full history so `git describe --tags --exact-match` resolves the version |
| Set up Go | `actions/setup-go@v5`, Go 1.21 | Toolchain |
| Install tools | `gettext-base` if `envsubst` is missing | `envsubst` templates `nfpm.yaml` |
| Test | `go test ./...` | Release gate — a failing test aborts the release |
| Build | `make release packages` | Produces every asset in `dist/` |
| Inspect | `dpkg-deb -I`, `rpm -qip` | Logs package metadata for auditing |
| Upload | `gh release upload "$VERSION" dist/* --clobber` | Attaches **everything** in `dist/` |
| Promote | `gh release edit "$VERSION" --prerelease=false --latest` | Makes the release installable |

Two environment values drive the version:

- `VERSION="${GITHUB_REF_NAME#v}"` — the tag with the leading `v` stripped, used for package
  filenames and package metadata.
- `RPM_RELEASE=1` — the RPM release/revision number. Bump it manually only if you need to rebuild
  the same upstream version as a new RPM.

`make release` itself derives `VERSION` from `git describe --tags --exact-match`, which is why the
tag must exist and point at the checked-out commit. Without an exact tag match the version falls
back to the string `dev` and the binaries report `dev` from `--version`.

## Assets built during the pre-release

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

## Pre-release checklist

Before creating the pre-release:

- [ ] `main` is green in CI (test matrix, lint, security, CodeQL)
- [ ] `make quality` passes locally
- [ ] `CHANGELOG.md` has a section for the new version (move items out of `[Unreleased]`, add the
      compare link at the bottom)
- [ ] Version number follows [Semantic Versioning](https://semver.org/) and matches the commit types
      since the last tag: breaking change → major, `feat` → minor, `fix` → patch
- [ ] Any breaking change or significant new feature has an [ADR](adr/README.md)
- [ ] Docs (`README.md`, `docs/`) reflect new or changed commands and flags

## Verifying a release

```bash
# All 10 assets present?
gh release view v1.2.3 --json assets --jq '.assets[].name'

# Release is latest and no longer a pre-release?
gh release view v1.2.3 --json isLatest,isPrerelease

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
| Release published with no assets | Release was created as a final release, not a pre-release | `gh release edit vX.Y.Z --prerelease`, then re-publish to fire the `prereleased` event |
| Workflow never started | Tag pushed but no GitHub release object created | Create the release: `gh release create vX.Y.Z --prerelease` |
| Binaries report version `dev` | `git describe --tags --exact-match` found no tag on HEAD, or checkout lacked full history | Ensure the tag points at the released commit and `fetch-depth: 0` is set |
| `envsubst: command not found` | Missing `gettext-base` | Install it; the workflow does this automatically |
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

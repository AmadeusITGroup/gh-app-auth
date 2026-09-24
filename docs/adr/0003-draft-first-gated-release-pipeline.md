# 0003 Draft-First Gated Release Pipeline

Date: 2026-09-15
Status: Proposed

## Context

Releases were produced by a prerelease-promotion model: the maintainer created a
GitHub **prerelease**, the workflow attached assets to it, then flipped it to
latest. Two requirements made that model untenable:

- **GitHub immutable releases** (issue #41). When the repository setting "Make
  new releases immutable" is enabled, a release's assets and tag lock *at
  publish*. Verified empirically on a fork: uploading to a **published
  prerelease** returns `HTTP 422: Cannot upload assets to an immutable
  release`. Any flow that mutates assets on a published release — prerelease
  included — is dead under immutability. Drafts are the only unconditionally
  mutable state: `isImmutable` stays `false` until publish.
- **A release gate** (issue #39). Artifacts must pass the cross-platform E2E
  suite and build provenance *before* they are installable. A published
  prerelease is public — a failed gate would leave a public release missing or
  carrying bad assets. A draft is invisible to anyone without push access, so a
  failed gate leaves nothing exposed.

Verified mechanics that shaped the design (fork probes, recorded in
`.tmp/studies/ideal-release-flow-design.md` §9a):

- Draft releases do **not** materialize the git tag — it appears only at
  publish. Under immutable releases the draft *binds* the tag name, so the ref
  cannot be pushed while the draft exists (a push is evaluated as an update and
  rejected even for admin actors — verified on fork). The pipeline therefore
  tags the release commit **locally** in `prepare`/`build` so
  `git describe --tags --exact-match` (which feeds `LDFLAGS` in the Makefile)
  resolves; `publish` materializes the remote ref at `target_commitish`.
- `gh release download` on a draft requires `contents: write` on
  `GITHUB_TOKEN` — read-only tokens get `release not found`. A single staging
  job therefore downloads the draft assets and uploads per-platform Actions
  artifacts; the test matrix consumes those exact bytes with `contents: read`.
- Release runs are restricted to source commits on `main`: manual dispatch is
  guarded to `refs/heads/main`, the free-form ref input and tag-push trigger are
  removed, and the secret-bearing E2E workflow is reusable-only. This prevents
  accidentally running release/test code from an arbitrary selected ref.
  Protect `.github/workflows/` on `main` with required review and limit manual
  release execution to trusted maintainers; the YAML guard is not a defense
  against a writer who can modify and run a different workflow definition. A
  `v*` tag ruleset remains defense in depth and must be configured in repo
  settings.
- Deleting a published release's tag reverts the release to draft — never
  delete tags of published releases.

Alternatives considered:

- **Keep prerelease promotion.** Rejected — incompatible with immutable
  releases (HTTP 422 on asset upload), and a failed gate leaves a *public*
  broken release instead of an invisible draft.
- **`release: [prereleased]` and tag-push triggers.** Rejected — they couple
  the trigger to mutable/published release state or execute workflow code from
  a pushed tag. Main-only manual dispatch plus the release-please `workflow_call`
  provides explicit release entry points without the arbitrary-ref surface.
- **GoReleaser end-to-end.** Deferred — it would rewrite a working,
  reviewed Makefile+nfpm build path inside a supply-chain-sensitive change.
- **Attest files across the boundary.** Rejected in favour of digests — the
  attestation workflow receives a `sha256sum` manifest, never build files, so
  it does not execute the build outputs and can isolate signing permissions.
  The build still controls both artifacts and their digests; provenance binds
  those digests to the workflow execution, not to an assertion that the
  artifacts are safe.
- **Serial e2e → attest.** Rejected — digests don't depend on test outcomes;
  the gates run in parallel.

## Decision

Replace the prerelease model with a draft-first gated pipeline in
`.github/workflows/release.yml`:

```text
main dispatch / workflow_call → prepare → build → [e2e ∥ attest] → publish
                                     (version,  (assets    (gates must  (draft →
                                      tag,      to draft)   both pass)   latest)
                                      draft)
```

- **Triggers**: normal releases use the release-please `workflow_call` from
  `main`; manual `workflow_dispatch` is guarded to `main` and accepts an
  explicit `version` or operator-selected `bump` plus `dry_run`. The free-form
  `ref` input and `push: v*` trigger are removed. The release-please workflow's
  manual refresh is also guarded to `main`.
- **`prepare`** resolves the version via `scripts/next-version.sh` (explicit
  manual version, explicit operator-selected bump, or caller-provided tag),
  stages the git tag locally at the main release commit, and creates or reuses
  the draft release. It refuses to touch a published release or a tag pointing
  at a different commit.
- **`build`** checks out the main event SHA, stages the release tag locally,
  runs unit tests and `make release packages`, writes `dist/checksums.txt`, and
  uploads everything to the draft with `--clobber` (idempotent re-runs).
- **`e2e`** calls `e2e-release.yml`. One job downloads the real draft assets
  with `contents: write` and uploads per-platform Actions artifacts; the ten
  platform jobs use `contents: read` to test those exact bytes on Linux
  (deb/rpm, amd64+arm64), macOS (arm64+intel), and Windows (amd64+arm64). E2E
  App credentials are exposed only to the test steps and must be limited to the
  test organizations. The test suite answers "are the artifacts good", never
  "ship them".
- **`attest`** calls `attest-release.yml`, a separate reusable workflow that
  receives only the checksum manifest and generates GitHub build provenance
  (keyless Sigstore, `id-token: write` + `attestations: write` + `contents:
  read`, no secrets). Provenance establishes artifact origin/integrity, not
  artifact safety.
- **`publish`** runs only when every gate passed and `dry_run` is false:
  `gh release edit --draft=false --latest`, then a post-publish sanity check
  (download a binary, run `--version`, `gh attestation verify`).
- **`report-failure`** tells maintainers to check draft/published state before
  retrying: reuse the same version only while still a draft; cut the next version
  if already published.
- Trust-boundary actions are pinned to commit SHA; top-level `permissions: {}`
  with per-job least-privilege grants; a `concurrency` lock serializes
  dispatches; `dry_run` exercises everything except publish. Main protection
  with required review is part of the source trust boundary; manual releases
  and all secret-bearing reusable jobs reject non-main refs.

The maintainer runbook changes: normal releases come from merging the
release-please PR. Explicit manual overrides use `gh workflow run release.yml
--ref main -f version=vX.Y.Z` (or an explicit bump), never a non-main ref or a
manual tag push. Do not publish releases directly with `gh release create`.

## Consequences

- **Zero public exposure window.** Partial or failed releases are invisible
  drafts; `gh extension install` only ever sees a complete `latest` release.
- **Immutable-releases compatible.** All asset mutation happens pre-publish;
  enabling the repo setting is a no-op for this pipeline. The remote tag
  materializes at publish; a `v*` ruleset blocking tag updates/deletion is
  defense in depth and must be configured in repository settings.
- **Idempotent recovery before publish.** A failed run leaves the draft;
  re-running the same version from the same main commit reuses the local tag
  and draft, and uploads are clobbered. After publish, do not retry or reuse
  that version: an immutable release cannot be repaired, so recovery means
  cutting the next version (normally the next patch).
- GitHub build provenance for every asset digest, verifiable via
  `gh attestation verify`; it records workflow origin and digest integrity, not
  an independent safety assessment.
- Costs: the release flow is more machinery (three workflows + a version
  script); the ten E2E jobs still receive the scoped test App credentials even
  though they have only `contents: read`; the staging job tests release download
  once and matrix jobs test byte-identical workflow artifacts; maintainers must
  use release-please for normal version selection and follow the main-only
  manual override runbook documented in `docs/RELEASE_PROCESS.md`.
- Deliberately not done: an `environment: release` approval gate on `publish`
  and repository-level `v*` tag ruleset configuration. The latter remains
  defense in depth for tag immutability and must be configured by maintainers.
- Regression protection: the pipeline is exercised by `dry_run` dispatches
  from `main`; `scripts/next-version.sh` has shell tests for explicit version,
  explicit bump, resume, validation, and `workflow_call` cases; `make packages`
  validates that all four supported amd64/arm64 DEB/RPM artifacts exist.

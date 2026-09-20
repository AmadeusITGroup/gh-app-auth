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
  suite and SLSA attestation *before* they are installable. A published
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
  `GITHUB_TOKEN` — read-only tokens get `release not found`.
- `gh release create` server-side tag creation *does* fire `push: tags`, but a
  `GITHUB_TOKEN`-created tag/release does not — relevant to any future
  event-driven chaining (release-please must use `workflow_call`, not events).
- Deleting a published release's tag reverts the release to draft — never
  delete tags of published releases.

Alternatives considered:

- **Keep prerelease promotion.** Rejected — incompatible with immutable
  releases (HTTP 422 on asset upload), and a failed gate leaves a *public*
  broken release instead of an invisible draft.
- **`release: [prereleased]` event trigger.** Rejected — it couples the trigger
  to the staging object and cannot express the draft lifecycle; dispatch +
  tag-push covers manual and git-driven flows.
- **GoReleaser end-to-end.** Deferred — it would rewrite a working,
  reviewed Makefile+nfpm build path inside a supply-chain-sensitive change.
- **Attest files across the boundary.** Rejected in favour of digests — the
  attestation workflow receives a `sha256sum` manifest, never build files, so
  the builder cannot influence the attested identity (SLSA Build L3 isolation)
  and no cross-workflow artifact transfer is needed.
- **Serial e2e → attest.** Rejected — digests don't depend on test outcomes;
  the gates run in parallel.

## Decision

Replace the prerelease model with a draft-first gated pipeline in
`.github/workflows/release.yml`:

```text
dispatch / tag-push / workflow_call → prepare → build → [e2e ∥ attest] → publish
                                     (version,  (assets    (gates must  (draft →
                                      tag,      to draft)   both pass)   latest)
                                      draft)
```

- **Triggers**: `workflow_dispatch` (inputs `version` / `bump` / `ref` /
  `dry_run`) is the primary "release now" path; `push: v*` remains as a
  fallback; `workflow_call` is the reserved seam for release-please (phase 3,
  which must create *draft* releases and chain explicitly, never relying on
  `GITHUB_TOKEN` events).
- **`prepare`** resolves the version via `scripts/next-version.sh` (explicit
  version, conventional-commit auto-bump, or caller-provided tag), stages the
  git tag locally at the release commit, and creates or reuses the draft
  release. It refuses to touch a published release or a tag pointing at a
  different commit.
- **`build`** checks out the tag, runs unit tests, `make release packages`,
  writes `dist/checksums.txt`, and uploads everything to the draft with
  `--clobber` (idempotent re-runs).
- **`e2e`** calls `e2e.yml` — the existing 10-job matrix validates the draft's
  real assets on Linux (deb/rpm, amd64+arm64), macOS (arm64+intel), and Windows
  (amd64+arm64). The `promote-release` job moved out of `e2e.yml` into the
  pipeline's `publish` job — the test suite answers "are the artifacts good",
  never "ship them".
- **`attest`** calls `attest-release.yml`, a separate reusable workflow that
  receives only the checksum manifest and generates SLSA build provenance
  (keyless Sigstore, `id-token: write` + `attestations: write` + `contents:
  read`, no secrets).
- **`publish`** runs only when every gate passed and `dry_run` is false:
  `gh release edit --draft=false --latest`, then a post-publish sanity check
  (download a binary, run `--version`, `gh attestation verify`).
- **`report-failure`** summarizes that the release stays a draft and how to
  resume.
- Trust-boundary actions are pinned to commit SHA; top-level `permissions: {}`
  with per-job least-privilege grants; a `concurrency` lock serializes
  dispatches; `dry_run` exercises everything except publish.

The maintainer runbook changes: releases are dispatched via
`gh workflow run release.yml` (or Actions → Release → Run workflow), not
created with `gh release create --prerelease`.

## Consequences

- **Zero public exposure window.** Partial or failed releases are invisible
  drafts; `gh extension install` only ever sees a complete `latest` release.
- **Immutable-releases compatible.** All asset mutation happens pre-publish;
  enabling the repo setting is a no-op for this pipeline. Tag protection at
  publish comes free.
- **Idempotent recovery.** A failed run leaves the draft; re-dispatching the
  same version reuses the tag (must point at the same commit) and draft, and
  uploads are clobbered. A published-but-bad release cannot be mutated —
  recovery is cutting the next patch.
- **SLSA Build L3** provenance for every asset, verifiable via
  `gh attestation verify`, without managing signing keys.
- Costs: the release flow is more machinery (three workflows + a version
  script); `contents: write` on e2e jobs is broader than ideal (a read-only
  alternative exists — gate on `actions/download-artifact` bytes — at the cost
  of testing repo-artifact copies rather than the real release assets);
  maintainers must learn the dispatch runbook (a breaking change to the
  release process, documented in `docs/RELEASE_PROCESS.md`).
- Deliberately not done: release-please merge-to-main automation (phase 3
  seam is in place), the optional `environment: release` approval gate on
  `publish`, a `v*` tag ruleset, and `version.txt` for non-git builds.
- Regression protection: the pipeline itself is exercised by `dry_run`
  dispatches on any ref; `scripts/next-version.sh` has shell tests covering
  the version-resolution matrix; `make packages` now fails loudly on missing
  artifacts instead of reporting success.

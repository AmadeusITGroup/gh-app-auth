#!/usr/bin/env bash
# next-version.test.sh — unit-ish tests for next-version.sh on synthetic git repos.
#
# Usage: ./scripts/next-version.test.sh
# Exits 0 when every case passes, 1 otherwise.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NEXT_VERSION="$SCRIPT_DIR/next-version.sh"

PASS=0
FAIL=0
WORKDIR=""

cleanup() { [ -n "$WORKDIR" ] && rm -rf "$WORKDIR"; }
trap cleanup EXIT

# new_repo <name> — create a scratch repo, cd into it, return via $REPO
new_repo() {
    local dir="$WORKDIR/$1"
    mkdir -p "$dir"
    git init -q "$dir"
    git -C "$dir" config user.email "test@example.com"
    git -C "$dir" config user.name "test"
    git -C "$dir" commit -qm "chore: init" --allow-empty
    REPO="$dir"
}

commit() { git -C "$REPO" commit -qm "$1" --allow-empty; }
tag()    { git -C "$REPO" tag "$1" "${2:-HEAD}"; }

# run_case — run the script in $REPO with the current env; captures RC,
# LAST_OUT and RELEASE_TAG for the subsequent check.
run_case() {
    local out
    out="$(cd "$REPO" && \
        INPUT_TAG="${INPUT_TAG:-}" INPUT_VERSION="${INPUT_VERSION:-}" \
        INPUT_BUMP="${INPUT_BUMP:-}" EVENT_NAME="${EVENT_NAME:-}" \
        REF_NAME="${REF_NAME:-}" GITHUB_OUTPUT="$WORKDIR/output" \
        bash "$NEXT_VERSION" 2>&1)"
    RC=$?
    LAST_OUT="$out"
    [ -f "$WORKDIR/output" ] && RELEASE_TAG=$(grep '^release_tag=' "$WORKDIR/output" | cut -d= -f2) || RELEASE_TAG=""
    rm -f "$WORKDIR/output"
}

check() { # <name> <cond...>
    local name="$1"; shift
    if "$@"; then
        PASS=$((PASS + 1)); echo "  ok   $name"
    else
        FAIL=$((FAIL + 1)); echo "  FAIL $name  (rc=$RC tag='$RELEASE_TAG' out: $LAST_OUT)"
    fi
}

expect_tag()  { [ "$RC" -eq 0 ] && [ "$RELEASE_TAG" = "$1" ]; }
expect_fail() { [ "$RC" -ne 0 ]; }

WORKDIR="$(mktemp -d)"

echo "next-version.sh tests"

# ── explicit version ────────────────────────────────────────────────
new_repo explicit-version
tag v1.2.0
commit "fix: a bug"
INPUT_VERSION=v1.3.0 run_case; check "explicit newer version resolves"          expect_tag v1.3.0
INPUT_VERSION=v1.1.0 run_case; check "explicit older version rejected"          expect_fail
INPUT_VERSION=1.3.0  run_case; check "missing v prefix rejected"                expect_fail
INPUT_VERSION=v9.9.9 run_case; check "explicit new major+ version resolves"     expect_tag v9.9.9

# ── resume: tag exists at HEAD vs different commit ──────────────────
new_repo resume
tag v1.0.0
commit "fix: a bug"
tag v1.0.1            # tag at HEAD (as prepare would have created it)
INPUT_VERSION=v1.0.1 run_case; check "tag at HEAD resumes (same commit)"        expect_tag v1.0.1
tag v1.0.2 HEAD~1     # tag at a different commit
INPUT_VERSION=v1.0.2 run_case; check "tag at different commit rejected"         expect_fail

# ── auto-bump detection ─────────────────────────────────────────────
new_repo auto-minor
tag v1.0.0
commit "feat(auth): add thing"
commit "docs: update readme"
INPUT_BUMP=auto run_case; check "auto: feat → minor"                            expect_tag v1.1.0

new_repo auto-patch
tag v2.3.4
commit "fix(cli): repair flag"
INPUT_BUMP=auto run_case; check "auto: fix → patch"                             expect_tag v2.3.5

new_repo auto-major
tag v1.4.0
commit "feat!: drop old flag"
INPUT_BUMP=auto run_case; check "auto: feat! → major"                           expect_tag v2.0.0

new_repo auto-major-body
tag v1.4.0
commit "refactor: reshape api

BREAKING CHANGE: config format changed"
INPUT_BUMP=auto run_case; check "auto: BREAKING CHANGE body → major"            expect_tag v2.0.0

# bump-minor-pre-major: breaking on 0.x bumps minor, matching release-please
new_repo auto-pre-major-bang
tag v0.4.0
commit "feat!: drop old flag"
INPUT_BUMP=auto run_case; check "auto: feat! on 0.x → minor (pre-major)"        expect_tag v0.5.0

new_repo auto-pre-major-body
tag v0.4.0
commit "refactor: reshape api

BREAKING CHANGE: config format changed"
INPUT_BUMP=auto run_case; check "auto: BREAKING on 0.x → minor (pre-major)"     expect_tag v0.5.0

new_repo explicit-major-pre-1x
tag v0.4.0
commit "docs: only docs"
INPUT_BUMP=major run_case; check "explicit major on 0.x still forces 1.0.0"     expect_tag v1.0.0

new_repo auto-nothing
tag v1.0.0
commit "docs: tweak"
commit "ci: adjust workflow"
INPUT_BUMP=auto run_case; check "auto: nothing releasable → abort"              expect_fail

new_repo auto-explicit-bump
tag v3.1.4
commit "docs: only docs"        # no releasable type — explicit bump must still work
INPUT_BUMP=minor run_case; check "explicit minor applies"                       expect_tag v3.2.0
INPUT_BUMP=major run_case; check "explicit major applies"                       expect_tag v4.0.0
# shellcheck disable=SC2209  # INPUT_BUMP is an env prefix, not a substitution
INPUT_BUMP=patch run_case; check "explicit patch applies"                       expect_tag v3.1.5

# ── no prior tag ────────────────────────────────────────────────────
new_repo no-tag
INPUT_BUMP=auto  run_case; check "no tag + nothing releasable → abort"          expect_fail
new_repo no-tag-feat
commit "feat: first feature"
INPUT_BUMP=auto run_case; check "no tag + feat → v0.1.0"                        expect_tag v0.1.0

# ── push: tags path ─────────────────────────────────────────────────
new_repo push-event
tag v5.5.5
EVENT_NAME=push REF_NAME=v5.5.5 run_case; check "push event uses REF_NAME"      expect_tag v5.5.5
EVENT_NAME=push REF_NAME=bad-tag run_case; check "push event rejects non-semver" expect_fail

# ── workflow_call path ──────────────────────────────────────────────
new_repo call-input
commit "feat: x"
INPUT_TAG=v7.0.0 run_case; check "caller-provided tag used as-is"               expect_tag v7.0.0
INPUT_TAG=v7 run_case;      check "caller-provided tag validated"               expect_fail

echo
echo "$PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]

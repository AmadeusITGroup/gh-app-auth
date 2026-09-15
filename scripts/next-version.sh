#!/usr/bin/env bash
# next-version.sh — resolve the release tag for the release pipeline.
#
# Reads (environment):
#   INPUT_TAG      tag provided by a workflow_call caller (e.g. release-please)
#   INPUT_VERSION  explicit version from workflow_dispatch (e.g. "v1.4.0")
#   INPUT_BUMP     auto|patch|minor|major — used when INPUT_VERSION is empty
#   EVENT_NAME     github.event_name
#   REF_NAME       github.ref_name (the tag name on push: tags events)
#
# Writes to $GITHUB_OUTPUT:
#   release_tag    resolved vX.Y.Z tag
#   ref_sha        commit the release is cut from (current HEAD)
#
# Resolution order:
#   1. INPUT_TAG        (workflow_call — tag/release created by the caller)
#   2. push: tags       (tag pushed by hand — REF_NAME is the tag)
#   3. INPUT_VERSION    (dispatch with explicit version)
#   4. INPUT_BUMP       (dispatch — computed from conventional commits)

set -euo pipefail

SEMVER_RE='^v[0-9]+\.[0-9]+\.[0-9]+$'

die() { echo "::error::$*" >&2; exit 1; }

ref_sha="$(git rev-parse HEAD)"
latest="$(git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || true)"

# version_gt A B → true if A sorts after B (semver)
version_gt() {
    [ "$1" != "$2" ] && [ "$(printf '%s\n%s\n' "${1#v}" "${2#v}" | sort -V | head -n1)" = "${2#v}" ]
}

# detect_bump → prints major|minor|patch or nothing, based on conventional
# commits in <latest-tag>..HEAD
detect_bump() {
    local range="HEAD"
    [ -n "$latest" ] && range="$latest..HEAD"

    local subjects bodies
    subjects="$(git log "$range" --format=%s)"
    bodies="$(git log "$range" --format=%B)"

    if printf '%s' "$bodies" | grep -qE 'BREAKING[ -]CHANGE|^[a-zA-Z]+(\([^)]*\))?!:'; then
        # bump-minor-pre-major (matches release-please-config.json): a breaking
        # change on a 0.x project bumps minor, not major. An explicit
        # 'bump=major' dispatch still forces 1.0.0.
        if [[ "$latest" =~ ^v0\. ]]; then
            echo "minor"
        else
            echo "major"
        fi
    elif printf '%s' "$subjects" | grep -qE '^feat(\(|:)'; then
        echo "minor"
    elif printf '%s' "$subjects" | grep -qE '^(fix|perf|revert|deps)(\(|:)'; then
        echo "patch"
    fi
}

# apply_bump LATEST BUMP → prints the next tag
apply_bump() {
    local cur="$1" bump="$2"
    if [ -z "$cur" ]; then
        echo "v0.1.0"
        return
    fi
    local major minor patch
    IFS=. read -r major minor patch <<< "${cur#v}"
    case "$bump" in
        major) echo "v$((major + 1)).0.0" ;;
        minor) echo "v${major}.$((minor + 1)).0" ;;
        patch) echo "v${major}.${minor}.$((patch + 1))" ;;
        *) die "unknown bump '$bump' (expected auto|patch|minor|major)" ;;
    esac
}

tag=""

if [ -n "${INPUT_TAG:-}" ]; then
    # workflow_call path — caller (release-please) decided the version.
    # The tag may not exist yet: draft releases don't materialize git tags.
    tag="$INPUT_TAG"
    [[ "$tag" =~ $SEMVER_RE ]] || die "invalid release_tag '$tag' (expected vX.Y.Z)"

elif [ "${EVENT_NAME:-}" = "push" ]; then
    # push: tags path — the tag exists by definition.
    tag="${REF_NAME:?}"
    [[ "$tag" =~ $SEMVER_RE ]] || die "invalid tag '$tag' (expected vX.Y.Z)"

elif [ -n "${INPUT_VERSION:-}" ]; then
    # dispatch with explicit version — new and newer than the latest tag, OR an
    # existing tag at ref_sha (resume a failed run whose draft is still staged).
    tag="$INPUT_VERSION"
    [[ "$tag" =~ $SEMVER_RE ]] || die "invalid version '$tag' (expected vX.Y.Z)"
    if git rev-parse -q --verify "refs/tags/$tag" >/dev/null 2>&1; then
        [ "$(git rev-parse "$tag^{commit}")" = "$ref_sha" ] || die "tag $tag exists at a different commit ($(git rev-parse "$tag^{commit}") != $ref_sha)"
        echo "tag $tag exists at ref_sha — resuming (existing draft will be reused)"
    else
        [ -z "$latest" ] || version_gt "$tag" "$latest" || die "$tag is not newer than latest tag $latest"
    fi

else
    # dispatch with bump
    bump="${INPUT_BUMP:-auto}"
    if [ "$bump" = "auto" ]; then
        bump="$(detect_bump)"
        [ -n "$bump" ] || die "no releasable conventional commits since ${latest:-<beginning>} — pass an explicit version or bump"
        echo "auto-detected bump: $bump (since ${latest:-<beginning>})"
    fi
    tag="$(apply_bump "$latest" "$bump")"
    git rev-parse -q --verify "refs/tags/$tag" >/dev/null 2>&1 && die "tag $tag already exists"
fi

echo "resolved release tag: $tag (at $ref_sha, latest: ${latest:-none})"
echo "release_tag=$tag" >> "$GITHUB_OUTPUT"
echo "ref_sha=$ref_sha" >> "$GITHUB_OUTPUT"

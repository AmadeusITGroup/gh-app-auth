#!/usr/bin/env bash
# next-version.sh — resolve the release tag for the release pipeline.
#
# Reads (environment):
#   INPUT_TAG      tag provided by a workflow_call caller (e.g. release-please)
#   INPUT_VERSION  explicit version from workflow_dispatch (e.g. "v1.4.0")
#   INPUT_BUMP     patch|minor|major — used when INPUT_VERSION is empty
#
# Writes to $GITHUB_OUTPUT:
#   release_tag    resolved vX.Y.Z tag
#   ref_sha        commit the release is cut from (current HEAD)
#
# Resolution order:
#   1. INPUT_TAG        (workflow_call — tag/release created by the caller)
#   2. INPUT_VERSION    (dispatch with explicit version)
#   3. INPUT_BUMP       (manual dispatch — explicitly selected semver bump)

set -euo pipefail

SEMVER_RE='^v[0-9]+\.[0-9]+\.[0-9]+$'

die() { echo "::error::$*" >&2; exit 1; }

ref_sha="$(git rev-parse HEAD)"
latest="$(git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || true)"

# version_gt A B → true if A sorts after B (semver)
version_gt() {
    [ "$1" != "$2" ] && [ "$(printf '%s\n%s\n' "${1#v}" "${2#v}" | sort -V | head -n1)" = "${2#v}" ]
}

# apply_bump LATEST BUMP → prints the next tag
apply_bump() {
    local cur="$1" bump="$2"
    local major minor patch
    IFS=. read -r major minor patch <<< "${cur#v}"
    case "$bump" in
        major) echo "v$((major + 1)).0.0" ;;
        minor) echo "v${major}.$((minor + 1)).0" ;;
        patch) echo "v${major}.${minor}.$((patch + 1))" ;;
        *) die "unknown bump '$bump' (expected patch|minor|major)" ;;
    esac
}

tag=""

if [ -n "${INPUT_TAG:-}" ]; then
    # workflow_call path — caller (release-please) decided the version.
    # The tag may not exist yet: draft releases don't materialize git tags.
    tag="$INPUT_TAG"
    [[ "$tag" =~ $SEMVER_RE ]] || die "invalid release_tag '$tag' (expected vX.Y.Z)"

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
    bump="${INPUT_BUMP:-}"
    [ -n "$bump" ] || die "pass an explicit version or bump"
    [ -n "$latest" ] || die "no prior release tag; pass an explicit version"
    tag="$(apply_bump "$latest" "$bump")"
    git rev-parse -q --verify "refs/tags/$tag" >/dev/null 2>&1 && die "tag $tag already exists"
fi

echo "resolved release tag: $tag (at $ref_sha, latest: ${latest:-none})"
echo "release_tag=$tag" >> "$GITHUB_OUTPUT"
echo "ref_sha=$ref_sha" >> "$GITHUB_OUTPUT"

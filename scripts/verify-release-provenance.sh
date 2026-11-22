#!/usr/bin/env bash
set -euo pipefail

REPO=AmadeusITGroup/gh-app-auth
SIGNER="$REPO/.github/workflows/attest-release.yml"

fail() { printf '%s\n' "$*" >&2; exit 1; }

release="$(gh api "repos/$REPO/releases/latest")"
jq -e '
  type == "object" and .draft == false and .prerelease == false and
  (.tag_name | type == "string") and
  (.assets | type == "array" and length > 0 and
    all(.[]; .name | type == "string" and test("\\A[A-Za-z0-9][A-Za-z0-9._+-]*\\z")) and
    (map(.name) | length == (unique | length)))
' <<< "$release" > /dev/null || fail "invalid release metadata"

tag="$(jq -r '.tag_name' <<< "$release")"
[[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z.-]+)*$ ]] || fail "invalid release tag"
source_sha="$(gh api "repos/$REPO/commits/$tag" --jq .sha)"
[[ "$source_sha" =~ ^[0-9a-f]{40}$ ]] || fail "invalid release source commit"
mapfile -t names < <(jq -r '.assets[].name' <<< "$release")

download_dir="$(mktemp -d)"
trap 'rm -f -- "$download_dir"/*; rmdir -- "$download_dir"' EXIT
gh release download "$tag" --repo "$REPO" --dir "$download_dir"
shopt -s nullglob dotglob
downloaded=("$download_dir"/*)
[[ "${#downloaded[@]}" -eq "${#names[@]}" ]] || fail "release asset set changed during download"

verified=0
payloads=0
for name in "${names[@]}"; do
  artifact="$download_dir/$name"
  [[ -f "$artifact" && ! -L "$artifact" && -s "$artifact" ]] || fail "missing or invalid release asset: $name"
  [[ "$name" != checksums.txt ]] || continue
  gh attestation verify "$artifact" \
    --repo "$REPO" \
    --signer-workflow "$SIGNER" \
    --source-ref refs/heads/main \
    --source-digest "$source_sha" \
    --signer-digest "$source_sha" \
    --deny-self-hosted-runners \
    --predicate-type https://slsa.dev/provenance/v1
  verified=$((verified + 1))
  [[ "$name" == gh-app-auth-rpm-signing.pub ]] || payloads=$((payloads + 1))
done

[[ "$payloads" -gt 0 ]] || fail "release has no payload artifacts"
printf 'Verified provenance for %s assets in %s at %s\n' "$verified" "$tag" "$source_sha"

#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SUBJECT="$SCRIPT_DIR/verify-release-provenance.sh"
REPO=AmadeusITGroup/gh-app-auth
SHA=0123456789abcdef0123456789abcdef01234567

PASS=0
FAIL=0
WORKDIR="$(mktemp -d)"
BIN="$WORKDIR/bin"
mkdir -p "$BIN"

cleanup() {
  rm -f "$BIN/gh" "$WORKDIR/gh.log" "$WORKDIR/last-dir" "$WORKDIR/release.json" "$WORKDIR/executed"
  rmdir "$BIN" "$WORKDIR" || :
}
trap cleanup EXIT

cat > "$BIN/gh" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$GH_LOG"
case "$1 $2" in
  "api repos/AmadeusITGroup/gh-app-auth/releases/latest")
    [ -z "${GH_FAIL_API:-}" ] || exit 1
    cat "$FIXTURE_RELEASE"
    ;;
  "api repos/AmadeusITGroup/gh-app-auth/commits/"*)
    printf '%s\n' "$FIXTURE_SHA"
    ;;
  "release download")
    [ -z "${GH_FAIL_DOWNLOAD:-}" ] || exit 1
    dir=""
    while [ "$#" -gt 0 ]; do
      [ "$1" != --dir ] || dir="$2"
      shift
    done
    printf '%s\n' "$dir" > "$LAST_DIR"
    while IFS= read -r name; do
      [ -n "$name" ] || continue
      case "$name" in
        "$EMPTY_NAME") : > "$dir/$name" ;;
        "$SYMLINK_NAME") ln -s /dev/null "$dir/$name" ;;
        "$EXEC_NAME") printf '#!/bin/sh\ntouch "%s"\n' "$EXEC_MARKER" > "$dir/$name"; chmod +x "$dir/$name" ;;
        *) printf 'data\n' > "$dir/$name" ;;
      esac
    done <<< "$DOWNLOAD_NAMES"
    ;;
  "attestation verify")
    [ -z "${GH_FAIL_VERIFY:-}" ] || exit 1
    ;;
  *) exit 99 ;;
esac
STUB
chmod +x "$BIN/gh"

release_json() {
  local tag="$1" draft="$2" prerelease="$3"
  shift 3
  jq -n --arg tag "$tag" --argjson draft "$draft" --argjson prerelease "$prerelease" \
    '{tag_name: $tag, draft: $draft, prerelease: $prerelease, assets: ($ARGS.positional | map({name: .}))}' --args -- "$@"
}

run_case() {
  export GH_LOG="$WORKDIR/gh.log" LAST_DIR="$WORKDIR/last-dir" FIXTURE_RELEASE="$WORKDIR/release.json"
  export FIXTURE_SHA="${FIXTURE_SHA_VALUE:-$SHA}" EXEC_MARKER="$WORKDIR/executed"
  : > "$GH_LOG"
  rm -f "$EXEC_MARKER" "$LAST_DIR"
  if OUT="$(PATH="$BIN:$PATH" bash "$SUBJECT" 2>&1)"; then RC=0; else RC=$?; fi
}

reset_env() {
  unset GH_FAIL_API GH_FAIL_DOWNLOAD GH_FAIL_VERIFY
  export EMPTY_NAME="" SYMLINK_NAME="" EXEC_NAME="" DOWNLOAD_NAMES="" FIXTURE_SHA_VALUE=""
}

check() {
  local name="$1"; shift
  if "$@"; then
    PASS=$((PASS + 1)); echo "  ok   $name"
  else
    FAIL=$((FAIL + 1)); echo "  FAIL $name  (rc=$RC out: $OUT)"
  fi
}

failed()          { [ "$RC" -ne 0 ]; }
no_success()      { ! grep -q 'Verified provenance' <<< "$OUT"; }
no_download()     { ! grep -q '^release download' "$GH_LOG"; }
no_verify()       { ! grep -q '^attestation verify' "$GH_LOG"; }
failed_early()    { failed && no_success && no_download && no_verify; }
failed_no_green() { failed && no_success; }
verify_count()    { [ "$(grep -c '^attestation verify' "$GH_LOG")" -eq "$1" ]; }
success_line()    { [ "$RC" -eq 0 ] && grep -qxF "Verified provenance for 3 assets in v1.2.3 at $SHA" <<< "$OUT"; }
manifest_unverified() { ! grep '^attestation verify' "$GH_LOG" | grep -q 'checksums.txt'; }
not_executed()    { [ ! -e "$EXEC_MARKER" ]; }
download_command() { grep -qxF "release download v1.2.3 --repo $REPO --dir $dir" "$GH_LOG"; }
fixture_has_name() { jq -e --arg bad "$1" '.assets[1].name == $bad' "$WORKDIR/release.json" > /dev/null; }

standard_names=$'checksums.txt\nlinux-amd64\nexample.deb\ngh-app-auth-rpm-signing.pub'

reset_env
release_json v1.2.3 false false checksums.txt linux-amd64 example.deb gh-app-auth-rpm-signing.pub > "$WORKDIR/release.json"
DOWNLOAD_NAMES="$standard_names"
EXEC_NAME=example.deb
run_case
dir="$(cat "$LAST_DIR")"
expected() {
  printf 'attestation verify %s/%s --repo %s --signer-workflow %s/.github/workflows/attest-release.yml --source-ref refs/heads/main --source-digest %s --signer-digest %s --deny-self-hosted-runners --predicate-type https://slsa.dev/provenance/v1\n' "$dir" "$1" "$REPO" "$REPO" "$SHA" "$SHA"
}
has_exact() { grep -qxF -- "$(expected "$1")" "$GH_LOG"; }
echo "success path"
check "exits zero with final success line for 3 assets" success_line
check "verifies exactly 3 assets" verify_count 3
check "downloads the tag from the canonical repo into the captured directory" download_command
check "policy arguments for linux-amd64" has_exact linux-amd64
check "policy arguments for example.deb" has_exact example.deb
check "policy arguments for gh-app-auth-rpm-signing.pub" has_exact gh-app-auth-rpm-signing.pub
check "checksums.txt is never verified" manifest_unverified
check "downloaded artifacts are never executed" not_executed

echo "metadata failures"
reset_env
release_json v1.2.3 false false linux-amd64 > "$WORKDIR/release.json"
GH_FAIL_API=1 run_case
check "release API failure propagates before download" failed_early

reset_env
printf '{not json' > "$WORKDIR/release.json"
run_case
check "malformed JSON fails early" failed_early

reset_env
release_json v1.2.3 false false > "$WORKDIR/release.json"
run_case
check "empty assets fail early" failed_early

reset_env
jq -n '{draft: false, prerelease: false, assets: [{name: "linux-amd64"}]}' > "$WORKDIR/release.json"
run_case
check "missing tag fails early" failed_early

reset_env
release_json v1.2.3 true false linux-amd64 > "$WORKDIR/release.json"
run_case
check "draft release fails early" failed_early

reset_env
release_json v1.2.3 false true linux-amd64 > "$WORKDIR/release.json"
run_case
check "prerelease fails early" failed_early

reset_env
release_json 1.2.3-bad false false linux-amd64 > "$WORKDIR/release.json"
run_case
check "invalid tag fails early" failed_early

echo "unsafe and duplicate names"
for bad in '../escape' '/abs' '--flag' $'line\nbreak' $'linux-amd64\n'; do
  reset_env
  release_json v1.2.3 false false linux-amd64 "$bad" > "$WORKDIR/release.json"
  fixture_has_name "$bad"
  run_case
  check "unsafe asset name fails early: $(printf '%q' "$bad")" failed_early
done

reset_env
release_json v1.2.3 false false linux-amd64 linux-amd64 > "$WORKDIR/release.json"
run_case
check "duplicate asset names fail early" failed_early

reset_env
release_json v1.2.3 false false linux-amd64 > "$WORKDIR/release.json"
FIXTURE_SHA_VALUE=notasha
run_case
check "invalid source SHA fails early" failed_early

echo "download and file checks"
reset_env
release_json v1.2.3 false false linux-amd64 > "$WORKDIR/release.json"
DOWNLOAD_NAMES=linux-amd64
GH_FAIL_DOWNLOAD=1 run_case
check "download failure fails" failed_no_green

reset_env
release_json v1.2.3 false false linux-amd64 example.deb > "$WORKDIR/release.json"
DOWNLOAD_NAMES=linux-amd64
run_case
check "missing downloaded asset fails" failed_no_green

reset_env
release_json v1.2.3 false false linux-amd64 example.deb > "$WORKDIR/release.json"
DOWNLOAD_NAMES=$'linux-amd64\nexample.deb'
EMPTY_NAME=example.deb
run_case
check "empty downloaded asset fails" failed_no_green

reset_env
release_json v1.2.3 false false linux-amd64 example.deb > "$WORKDIR/release.json"
DOWNLOAD_NAMES=$'linux-amd64\nexample.deb'
SYMLINK_NAME=example.deb
run_case
check "symlink asset fails" failed_no_green

reset_env
release_json v1.2.3 false false linux-amd64 > "$WORKDIR/release.json"
DOWNLOAD_NAMES=$'linux-amd64\nunexpected-extra'
run_case
check "unexpected extra download fails" failed_no_green

echo "policy outcomes"
reset_env
release_json v1.2.3 false false checksums.txt gh-app-auth-rpm-signing.pub > "$WORKDIR/release.json"
DOWNLOAD_NAMES=$'checksums.txt\ngh-app-auth-rpm-signing.pub'
run_case
check "checksums and public key only is not a false green" failed_no_green

reset_env
release_json v1.2.3 false false linux-amd64 > "$WORKDIR/release.json"
DOWNLOAD_NAMES=linux-amd64
GH_FAIL_VERIFY=1 run_case
check "attestation verification failure propagates" failed_no_green

echo
echo "passed: $PASS  failed: $FAIL"
[ "$FAIL" -eq 0 ]

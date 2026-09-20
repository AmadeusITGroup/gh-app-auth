#!/usr/bin/env bash
# e2e-coverage.sh — run the e2e suite against a coverage-instrumented binary.
#
# `go test -cover` cannot measure this suite: the tests exec the binary as a
# subprocess, so counters live outside the test process. Instead we build with
# `go build -cover -coverpkg=./...` and point GOCOVERDIR at a collection
# directory — every instrumented process (binary runs, git credential-helper
# invocations) writes counters there on exit.
#
# Outputs (all paths overridable via env):
#   E2E_COV_DIR      raw covdata directory        (.e2e-covdata)
#   E2E_COV_BINARY   instrumented binary          (<E2E_COV_DIR>/gh-app-auth-cover)
#   E2E_COV_PROFILE  textfmt coverprofile         (e2e-coverage.out)
#   E2E_COV_SUMMARY  markdown table for GH step summary (e2e-coverage-summary.md)
#
# Requires E2E_APP_ID + E2E_PRIVATE_KEY / E2E_PRIVATE_KEY_B64 — see
# docs/E2E_INFRASTRUCTURE.md.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

E2E_COV_DIR="${E2E_COV_DIR:-.e2e-covdata}"
E2E_COV_BINARY="${E2E_COV_BINARY:-$E2E_COV_DIR/gh-app-auth-cover$(go env GOEXE)}"
E2E_COV_PROFILE="${E2E_COV_PROFILE:-e2e-coverage.out}"
E2E_COV_SUMMARY="${E2E_COV_SUMMARY:-e2e-coverage-summary.md}"
TEST_TIMEOUT="${TEST_TIMEOUT:-15m}"

# Stale counters from a previous run would silently merge into the results.
rm -rf "$E2E_COV_DIR"
mkdir -p "$E2E_COV_DIR"

echo "Building coverage-instrumented binary -> $E2E_COV_BINARY"
go build -cover -coverpkg=./... -o "$E2E_COV_BINARY" .

# covdata needs absolute paths — the binary may be exec'd from temp dirs.
ABS_COV_DIR="$(cd "$E2E_COV_DIR" && pwd)"
ABS_BINARY="$(cd "$(dirname "$E2E_COV_BINARY")" && pwd)/$(basename "$E2E_COV_BINARY")"

status=0
E2E_BINARY_PATH="$ABS_BINARY" GOCOVERDIR="$ABS_COV_DIR" \
	go test -v -tags=e2e -timeout="$TEST_TIMEOUT" ./test/e2e/... || status=$?

# Emit reports even when tests failed — partial coverage is still evidence.
if ! ls "$ABS_COV_DIR"/covmeta.* >/dev/null 2>&1; then
	echo "warning: no coverage data collected (binary never ran?)" >&2
	echo "No e2e coverage data collected." > "$E2E_COV_SUMMARY"
	exit "$status"
fi

go tool covdata textfmt -i="$ABS_COV_DIR" -o "$E2E_COV_PROFILE"
TOTAL="$(go tool cover -func="$E2E_COV_PROFILE" | awk '/^total:/ {print $NF}')"

{
	echo "## E2E coverage (instrumented binary)"
	echo
	echo "Every subprocess the suite spawns writes counters to GOCOVERDIR;"
	echo "this measures real app coverage, not just the test package."
	echo
	echo "| Package | Statements |"
	echo "|---|---|"
	go tool covdata percent -i="$ABS_COV_DIR" | awk '{
		pkg = $1
		sub("github.com/AmadeusITGroup/gh-app-auth/?", "", pkg)
		if (pkg == "") pkg = "main"
		match($0, /[0-9.]+%/)
		printf "| %s | %s |\n", pkg, substr($0, RSTART, RLENGTH)
	}' | sort
	echo "| **total** | **${TOTAL:-n/a}** |"
} | tee "$E2E_COV_SUMMARY"

echo
echo "Raw covdata:      $E2E_COV_DIR"
echo "Coverprofile:     $E2E_COV_PROFILE  (go tool cover -func=$E2E_COV_PROFILE)"
echo "Markdown summary: $E2E_COV_SUMMARY"
echo
echo "To combine with unit-test coverage, generate the unit profile first:"
echo "  go test -coverprofile=coverage.out ./..."
echo "then union the two textfmt profiles per block (count>0 in either),"
echo "e.g. with gocovmerge or a small awk script keyed on 'file:range'."

exit "$status"

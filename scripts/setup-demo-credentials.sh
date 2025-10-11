#!/usr/bin/env bash
set -euo pipefail

###
# setup-demo-credentials.sh
# -------------------------
# Demonstrates configuring gh-app-auth with a mix of GitHub Apps and Personal Access Tokens
# targeting repositories across multiple organizations plus Bitbucket Server.
#
# Required environment variables (examples below use placeholder values):
#   ENTERPRISE_APP_ID=123456
#   ENTERPRISE_KEY=~/keys/enterprise-app.pem
#   ENTERPRISE_INSTALLATION_ID=987654321        # optional if auto-detect works
#   PERSONAL_GH_PAT=ghp_example_personal_token
#   BITBUCKET_PAT=bb_example_pat_token
#   BITBUCKET_USERNAME=your-username            # Bitbucket HTTP username
#
# Optional environment variables:
#   ENTERPRISE_PATTERNS="github.com/your-org/,github.com/your-team/"
#   PERSONAL_REPO_PATTERN="github.com/your-personal-org/"
#   BITBUCKET_PATTERN="bitbucket.example.com/"
#
# Usage example:
#   ENTERPRISE_APP_ID=123456 \
#   ENTERPRISE_KEY=~/keys/enterprise-app.pem \
#   ENTERPRISE_INSTALLATION_ID=987654321 \
#   PERSONAL_GH_PAT=ghp_your_personal_token \
#   BITBUCKET_PAT=bbpat_your_bitbucket_token \
#   BITBUCKET_USERNAME=your-username \
#   ./scripts/setup-demo-credentials.sh
###

log() {
  printf '\n➡️  %s\n' "$1"
}

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    printf '❌ Missing required environment variable: %s\n' "$name" >&2
    MISSING_ENV=true
  fi
}

ensure_dependencies() {
  for bin in gh; do
    if ! command -v "$bin" >/dev/null 2>&1; then
      printf '❌ Required command "%s" not found in PATH.\n' "$bin" >&2
      exit 1
    fi
  done
  if ! gh extension list 2>/dev/null | grep -qi 'gh-app-auth'; then
    printf '❌ gh-app-auth extension is not installed. Run "gh extension install AmadeusITGroup/gh-app-auth" first.\n' >&2
    exit 1
  fi
}

join_patterns() {
  local raw="$1"
  # Collapse whitespace and ensure comma separation
  echo "$raw" | tr '\n' ',' | sed 's/[[:space:]]//g; s/,,*/,/g; s/,$//'
}

setup_enterprise_app() {
  local patterns
  patterns=$(join_patterns "${ENTERPRISE_PATTERNS}")

  log "Configuring Enterprise GitHub App for patterns: ${patterns}"
  gh app-auth setup \
    --app-id "$ENTERPRISE_APP_ID" \
    --key-file "$ENTERPRISE_KEY" \
    ${ENTERPRISE_INSTALLATION_ID:+--installation-id "$ENTERPRISE_INSTALLATION_ID"} \
    --patterns "$patterns" \
    --name "Enterprise App" \
    --priority 50
}

setup_personal_pat() {
  local patterns
  patterns=$(join_patterns "${PERSONAL_REPO_PATTERN}")

  log "Configuring Personal GitHub PAT for pattern: ${patterns}"
  gh app-auth setup \
    --pat "$PERSONAL_GH_PAT" \
    --patterns "$patterns" \
    --name "Personal Templates PAT" \
    --priority 80
}

setup_bitbucket_pat() {
  local patterns
  patterns=$(join_patterns "${BITBUCKET_PATTERN}")

  log "Configuring Bitbucket PAT for pattern: ${patterns} (username: ${BITBUCKET_USERNAME})"
  gh app-auth setup \
    --pat "$BITBUCKET_PAT" \
    --patterns "$patterns" \
    --username "$BITBUCKET_USERNAME" \
    --name "Bitbucket PAT" \
    --priority 40
}

main() {
  ensure_dependencies

  ENTERPRISE_PATTERNS=${ENTERPRISE_PATTERNS:-"github.com/your-org/,github.com/your-team/"}
  PERSONAL_REPO_PATTERN=${PERSONAL_REPO_PATTERN:-"github.com/your-personal-org/"}
  BITBUCKET_PATTERN=${BITBUCKET_PATTERN:-"bitbucket.example.com/"}

  MISSING_ENV=false
  require_env ENTERPRISE_APP_ID
  require_env ENTERPRISE_KEY
  require_env PERSONAL_GH_PAT
  require_env BITBUCKET_PAT
  require_env BITBUCKET_USERNAME
  if [[ "$MISSING_ENV" == true ]]; then
    exit 1
  fi

  setup_enterprise_app
  setup_personal_pat
  setup_bitbucket_pat

  log "Demo credential setup completed!"
  gh app-auth list
}

main "$@"

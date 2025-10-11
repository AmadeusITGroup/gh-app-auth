#!/usr/bin/env bash
#
# Cleanup E2E Test Environment
# Removes all test configurations, repositories, and local files
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  E2E Test Environment Cleanup${NC}"
echo -e "${BLUE}========================================${NC}"
echo

echo -e "${YELLOW}⚠️  WARNING ⚠️${NC}"
echo
echo "This script will:"
echo "  • Remove gh-app-auth configurations"
echo "  • Clean git credential helper settings"
echo "  • Optionally delete test repositories"
echo "  • Optionally delete GitHub App"
echo "  • Clean local temporary files"
echo

read -p "Continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleanup cancelled"
    exit 0
fi

# Function to cleanup with confirmation
cleanup_with_confirm() {
    local what=$1
    local command=$2
    
    echo
    echo -e "${BLUE}Cleanup: $what${NC}"
    read -p "Remove $what? (y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if eval "$command"; then
            echo -e "${GREEN}✓ Removed${NC}"
        else
            echo -e "${RED}✗ Failed${NC}"
        fi
    else
        echo -e "${YELLOW}⊘ Skipped${NC}"
    fi
}

echo
echo -e "${BLUE}=== gh-app-auth Configuration ===${NC}"

# List current apps
echo
echo "Currently configured apps:"
gh app-auth list 2>/dev/null || echo "  (none)"
echo

# Remove all app configurations
if gh app-auth list --json 2>/dev/null | jq -e 'length > 0' &> /dev/null; then
    read -p "Remove all gh-app-auth configurations? (y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Removing app configurations..."
        gh app-auth list --json | jq -r '.[].name' | while read name; do
            echo -n "  Removing $name... "
            if gh app-auth remove --name "$name" &> /dev/null; then
                echo -e "${GREEN}✓${NC}"
            else
                echo -e "${RED}✗${NC}"
            fi
        done
    fi
else
    echo "No app configurations to remove"
fi

# Clean git credential config
echo
echo -e "${BLUE}=== Git Credential Helper ===${NC}"
echo

git_creds=$(git config --global --get-regexp 'credential.*gh app-auth' || true)
if [[ -n "$git_creds" ]]; then
    echo "Current git credential helpers:"
    echo "$git_creds"
    echo
    
    read -p "Clean git credential helper config? (y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -n "Cleaning git config... "
        if gh app-auth gitconfig --clean --global &> /dev/null; then
            echo -e "${GREEN}✓${NC}"
        else
            echo -e "${YELLOW}⚠ Manual cleanup may be needed${NC}"
            echo "Run: git config --global --unset-all credential.<url>.helper"
        fi
    fi
else
    echo "No git credential helpers to clean"
fi

# Clean test repositories
if [[ -n "$TEST_ORG" ]]; then
    echo
    echo -e "${BLUE}=== Test Repositories ===${NC}"
    echo
    echo "Organization: $TEST_ORG"
    echo
    
    # List test repositories
    echo "Test repositories:"
    for repo in public-test-repo private-test-repo submodule-parent; do
        if gh api "/repos/$TEST_ORG/$repo" &> /dev/null; then
            echo "  • $TEST_ORG/$repo"
        fi
    done
    echo
    
    read -p "Delete test repositories? (y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        for repo in public-test-repo private-test-repo submodule-parent; do
            echo -n "  Deleting $repo... "
            if gh repo delete "$TEST_ORG/$repo" --yes &> /dev/null; then
                echo -e "${GREEN}✓${NC}"
            else
                echo -e "${YELLOW}⊘ Not found or already deleted${NC}"
            fi
        done
    fi
else
    echo
    echo -e "${YELLOW}TEST_ORG not set, skipping repository cleanup${NC}"
    echo "To clean repositories manually:"
    echo "  export TEST_ORG=\"your-org\""
    echo "  gh repo delete \$TEST_ORG/public-test-repo --yes"
    echo "  gh repo delete \$TEST_ORG/private-test-repo --yes"
    echo "  gh repo delete \$TEST_ORG/submodule-parent --yes"
fi

# GitHub App (manual instructions)
echo
echo -e "${BLUE}=== GitHub App ===${NC}"
echo
echo -e "${YELLOW}GitHub App cleanup must be done manually:${NC}"
echo
if [[ -n "$TEST_ORG" ]]; then
    echo "1. Uninstall app:"
    echo "   https://github.com/organizations/$TEST_ORG/settings/installations"
    echo
    echo "2. Delete app:"
    echo "   https://github.com/organizations/$TEST_ORG/settings/apps"
else
    echo "1. Go to: https://github.com/organizations/YOUR-ORG/settings/installations"
    echo "2. Click Configure next to your test app"
    echo "3. Scroll down and click Uninstall"
    echo
    echo "4. Go to: https://github.com/organizations/YOUR-ORG/settings/apps"
    echo "5. Click on your test app"
    echo "6. Scroll down and click 'Delete GitHub App'"
fi
echo

# Clean local files
echo
echo -e "${BLUE}=== Local Files ===${NC}"
echo

cleanup_with_confirm "Private key files" \
    "rm -f ~/.ssh/github-apps/test-app*.pem"

cleanup_with_confirm "Temporary test directories" \
    "rm -rf /tmp/gh-app-auth-* /tmp/*test-repo* /tmp/*submodule-parent*"

cleanup_with_confirm "Extension config directory" \
    "rm -rf ~/.config/gh/extensions/gh-app-auth/"

# Summary
echo
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✓ Cleanup complete${NC}"
echo
echo -e "${BLUE}Remaining manual steps (if applicable):${NC}"
echo "  1. Uninstall and delete GitHub App (see URLs above)"
echo "  2. Delete test organization (if dedicated test org):"
echo "     https://github.com/organizations/YOUR-ORG/settings/profile"
echo
echo -e "${BLUE}To start over:${NC}"
echo "  1. Follow: docs/E2E_TESTING_TUTORIAL.md"
echo "  2. Or run: ./test/e2e/scripts/setup-wizard.sh"
echo

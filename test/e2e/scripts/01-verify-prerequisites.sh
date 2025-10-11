#!/usr/bin/env bash
#
# Verify Prerequisites for E2E Testing
# Checks that all required tools and permissions are available
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  gh-app-auth E2E Prerequisites Check${NC}"
echo -e "${BLUE}========================================${NC}"
echo

# Track failures
FAILURES=0

# Function to check command exists
check_command() {
    local cmd=$1
    local name=$2
    local install_hint=$3
    
    echo -n "Checking $name... "
    if command -v "$cmd" &> /dev/null; then
        local version=$($cmd --version 2>&1 | head -n 1)
        echo -e "${GREEN}✓${NC} $version"
        return 0
    else
        echo -e "${RED}✗ Not found${NC}"
        echo -e "  ${YELLOW}Install: $install_hint${NC}"
        ((FAILURES++))
        return 1
    fi
}

# Function to check GitHub CLI authentication
check_gh_auth() {
    echo -n "Checking GitHub CLI authentication... "
    if gh auth status &> /dev/null; then
        local username=$(gh api user --jq '.login' 2>/dev/null || echo "unknown")
        echo -e "${GREEN}✓${NC} Logged in as $username"
        return 0
    else
        echo -e "${RED}✗ Not authenticated${NC}"
        echo -e "  ${YELLOW}Run: gh auth login${NC}"
        ((FAILURES++))
        return 1
    fi
}

# Function to check gh-app-auth extension
check_extension() {
    echo -n "Checking gh-app-auth extension... "
    if gh extension list | grep -q "app-auth"; then
        echo -e "${GREEN}✓${NC} Installed"
        return 0
    else
        echo -e "${RED}✗ Not installed${NC}"
        echo -e "  ${YELLOW}Install: gh extension install AmadeusITGroup/gh-app-auth${NC}"
        ((FAILURES++))
        return 1
    fi
}

# Function to check network connectivity
check_connectivity() {
    echo -n "Checking GitHub API connectivity... "
    if curl -s --connect-timeout 5 https://api.github.com > /dev/null; then
        echo -e "${GREEN}✓${NC} Connected"
        return 0
    else
        echo -e "${RED}✗ Cannot reach GitHub${NC}"
        echo -e "  ${YELLOW}Check your internet connection${NC}"
        ((FAILURES++))
        return 1
    fi
}

# Function to check permissions
check_permissions() {
    echo -n "Checking GitHub permissions... "
    local scopes=$(gh api /user -i 2>&1 | grep -i "x-oauth-scopes:" | cut -d: -f2 | tr -d ' ')
    
    if [[ -n "$scopes" ]]; then
        echo -e "${GREEN}✓${NC} OAuth scopes: $scopes"
        
        # Check for required scopes
        if echo "$scopes" | grep -q "repo"; then
            return 0
        else
            echo -e "  ${YELLOW}Warning: 'repo' scope recommended for full functionality${NC}"
        fi
    else
        echo -e "${YELLOW}⚠${NC} Could not determine scopes (might be fine)"
    fi
    
    return 0
}

# Function to check OS keyring availability
check_keyring() {
    echo -n "Checking OS keyring availability... "
    
    case "$(uname -s)" in
        Darwin*)
            echo -e "${GREEN}✓${NC} macOS Keychain available"
            ;;
        Linux*)
            if command -v secret-tool &> /dev/null || command -v gnome-keyring-daemon &> /dev/null; then
                echo -e "${GREEN}✓${NC} Linux Secret Service available"
            else
                echo -e "${YELLOW}⚠${NC} Keyring not detected (will use filesystem fallback)"
            fi
            ;;
        MINGW*|MSYS*|CYGWIN*)
            echo -e "${GREEN}✓${NC} Windows Credential Manager available"
            ;;
        *)
            echo -e "${YELLOW}⚠${NC} Unknown OS (will use filesystem fallback)"
            ;;
    esac
}

# Run all checks
echo -e "${BLUE}=== Core Tools ===${NC}"
check_command "gh" "GitHub CLI" "brew install gh  OR  sudo apt install gh"
check_command "git" "Git" "brew install git  OR  sudo apt install git"
check_command "jq" "jq (JSON processor)" "brew install jq  OR  sudo apt install jq"
check_command "curl" "curl" "brew install curl  OR  sudo apt install curl"

echo
echo -e "${BLUE}=== GitHub Configuration ===${NC}"
check_gh_auth
check_extension
check_connectivity
check_permissions

echo
echo -e "${BLUE}=== System Configuration ===${NC}"
check_keyring

# Check if config directory exists
echo -n "Checking gh-app-auth config directory... "
config_dir="$HOME/.config/gh/extensions/gh-app-auth"
if [[ -d "$config_dir" ]]; then
    echo -e "${GREEN}✓${NC} Exists"
else
    echo -e "${YELLOW}⚠${NC} Will be created on first setup"
fi

# Summary
echo
echo -e "${BLUE}========================================${NC}"
if [[ $FAILURES -eq 0 ]]; then
    echo -e "${GREEN}✓ All prerequisites met!${NC}"
    echo
    echo -e "${BLUE}Next steps:${NC}"
    echo "  1. Set TEST_ORG environment variable:"
    echo "     export TEST_ORG=\"your-org-name\""
    echo
    echo "  2. Run setup wizard:"
    echo "     ./test/e2e/scripts/setup-wizard.sh"
    echo
    echo "  3. Or follow manual steps in:"
    echo "     docs/E2E_TESTING_TUTORIAL.md"
    echo
    exit 0
else
    echo -e "${RED}✗ $FAILURES prerequisite(s) failed${NC}"
    echo
    echo -e "${YELLOW}Please install missing tools and try again.${NC}"
    echo "See: docs/E2E_TESTING_TUTORIAL.md"
    echo
    exit 1
fi

#!/usr/bin/env bash
#
# Interactive Setup Wizard
# Guides users through complete E2E test environment setup
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

clear

cat << "EOF"
╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║       gh-app-auth E2E Testing Setup Wizard                     ║
║                                                                ║
║       Interactive guide to set up your test environment        ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝
EOF

echo
echo -e "${CYAN}This wizard will guide you through:${NC}"
echo "  1. Prerequisites verification"
echo "  2. Test repository creation"
echo "  3. GitHub App setup guidance"
echo "  4. gh-app-auth configuration"
echo "  5. Validation tests"
echo
echo -e "${YELLOW}Time required: 20-30 minutes${NC}"
echo

read -p "Press Enter to begin..."
clear

#
# Step 1: Prerequisites
#
echo -e "${BLUE}${BOLD}Step 1: Prerequisites Verification${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

echo "Checking prerequisites..."
echo

if ./test/e2e/scripts/01-verify-prerequisites.sh; then
    echo
    echo -e "${GREEN}✓ Prerequisites verified${NC}"
else
    echo
    echo -e "${RED}Prerequisites check failed${NC}"
    echo "Please install missing tools and run this wizard again."
    exit 1
fi

read -p "Press Enter to continue..."
clear

#
# Step 2: Organization Setup
#
echo -e "${BLUE}${BOLD}Step 2: Test Organization${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

echo -e "${CYAN}You need a GitHub organization for testing.${NC}"
echo
echo "Options:"
echo "  a) Use existing organization"
echo "  b) Create new organization"
echo

read -p "Your choice (a/b): " -n 1 -r org_choice
echo
echo

if [[ $org_choice == "b" ]]; then
    echo -e "${YELLOW}Creating a new organization...${NC}"
    echo
    echo "1. Opening browser to: https://github.com/settings/organizations"
    echo "2. Click 'New organization'"
    echo "3. Choose 'Create a free organization'"
    echo "4. Enter organization name (e.g., 'gh-app-auth-testing-yourname')"
    echo "5. Complete the setup wizard"
    echo
    
    if command -v open &> /dev/null; then
        open "https://github.com/settings/organizations"
    elif command -v xdg-open &> /dev/null; then
        xdg-open "https://github.com/settings/organizations"
    else
        echo "Visit: https://github.com/settings/organizations"
    fi
    
    echo
    read -p "Press Enter when you've created the organization..."
fi

echo
read -p "Enter your test organization name: " TEST_ORG
export TEST_ORG

echo
echo -n "Verifying organization '$TEST_ORG'... "
if gh api "/orgs/$TEST_ORG" &> /dev/null; then
    echo -e "${GREEN}✓ Found${NC}"
else
    echo -e "${RED}✗ Not found${NC}"
    echo
    echo "Could not find organization '$TEST_ORG'"
    echo "Please verify the name and try again."
    exit 1
fi

read -p "Press Enter to continue..."
clear

#
# Step 3: Test Repositories
#
echo -e "${BLUE}${BOLD}Step 3: Test Repositories${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

echo -e "${CYAN}Creating 3 test repositories:${NC}"
echo "  • public-test-repo (public)"
echo "  • private-test-repo (private, requires auth)"
echo "  • submodule-parent (private, for submodule tests)"
echo

read -p "Create test repositories? (y/N): " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo
    ./test/e2e/scripts/02-create-test-repos.sh
else
    echo "Skipping repository creation..."
fi

read -p "Press Enter to continue..."
clear

#
# Step 4: GitHub App
#
echo -e "${BLUE}${BOLD}Step 4: GitHub App Setup${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

echo -e "${CYAN}Now you need to create a GitHub App${NC}"
echo

echo -e "${YELLOW}Follow these steps:${NC}"
echo
echo "1. Go to: https://github.com/organizations/$TEST_ORG/settings/apps"
echo "2. Click 'New GitHub App'"
echo
echo "3. Basic Information:"
echo "   - Name: gh-app-auth-test-app-<your-github-username>"
echo "   - Homepage: https://github.com/AmadeusITGroup/gh-app-auth"
echo "   - Webhook: Uncheck 'Active'"
echo
echo "4. Permissions:"
echo "   Repository permissions:"
echo "   - Contents: Read and write"
echo
echo "5. Where can this app be installed?"
echo "   - Select: 'Only on this account'"
echo
echo "6. Click 'Create GitHub App'"
echo

echo "Opening browser..."
if command -v open &> /dev/null; then
    open "https://github.com/organizations/$TEST_ORG/settings/apps/new"
elif command -v xdg-open &> /dev/null; then
    xdg-open "https://github.com/organizations/$TEST_ORG/settings/apps/new"
fi

echo
read -p "Press Enter when you've created the app..."

echo
read -p "Enter the App ID (found on app settings page): " APP_ID
export APP_ID

echo
echo -e "${CYAN}Generating Private Key...${NC}"
echo
echo "1. On the app settings page, scroll to 'Private keys'"
echo "2. Click 'Generate a private key'"
echo "3. A .pem file will download"
echo

read -p "Press Enter when you have the private key file..."

echo
echo -e "${CYAN}Securing private key...${NC}"
echo

# Find the downloaded key
key_file=$(ls -t ~/Downloads/*.private-key.pem 2>/dev/null | head -n 1 || echo "")

if [[ -n "$key_file" ]]; then
    echo "Found: $key_file"
    
    mkdir -p ~/.ssh/github-apps
    chmod 700 ~/.ssh/github-apps
    
    cp "$key_file" ~/.ssh/github-apps/test-app.pem
    chmod 600 ~/.ssh/github-apps/test-app.pem
    
    echo -e "${GREEN}✓ Private key secured at: ~/.ssh/github-apps/test-app.pem${NC}"
else
    echo -e "${YELLOW}Could not find private key in Downloads${NC}"
    read -p "Enter the full path to your private key file: " key_file
    
    if [[ -f "$key_file" ]]; then
        mkdir -p ~/.ssh/github-apps
        chmod 700 ~/.ssh/github-apps
        cp "$key_file" ~/.ssh/github-apps/test-app.pem
        chmod 600 ~/.ssh/github-apps/test-app.pem
        echo -e "${GREEN}✓ Private key secured${NC}"
    else
        echo -e "${RED}File not found: $key_file${NC}"
        exit 1
    fi
fi

read -p "Press Enter to continue..."
clear

#
# Step 5: Install App
#
echo -e "${BLUE}${BOLD}Step 5: Install GitHub App${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

echo -e "${CYAN}Installing the app to your organization...${NC}"
echo
echo "1. On your app settings page, click 'Install App' in the sidebar"
echo "2. Click 'Install' next to '$TEST_ORG'"
echo "3. Select 'All repositories'"
echo "4. Click 'Install'"
echo
echo "The URL will change to: .../installations/INSTALLATION_ID"
echo "Note the Installation ID from the URL"
echo

read -p "Press Enter to open installation page..."

if command -v open &> /dev/null; then
    open "https://github.com/organizations/$TEST_ORG/settings/installations"
elif command -v xdg-open &> /dev/null; then
    xdg-open "https://github.com/organizations/$TEST_ORG/settings/installations"
fi

echo
echo "Alternatively, get Installation ID via API:"
INSTALLATION_ID=$(gh api /orgs/$TEST_ORG/installation --jq '.id' 2>/dev/null || echo "")

if [[ -n "$INSTALLATION_ID" ]]; then
    echo -e "${GREEN}Found Installation ID: $INSTALLATION_ID${NC}"
    echo
    read -p "Use this Installation ID? (Y/n): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Nn]$ ]]; then
        export INSTALLATION_ID
    else
        read -p "Enter Installation ID: " INSTALLATION_ID
        export INSTALLATION_ID
    fi
else
    read -p "Enter Installation ID: " INSTALLATION_ID
    export INSTALLATION_ID
fi

read -p "Press Enter to continue..."
clear

#
# Step 6: Configure gh-app-auth
#
echo -e "${BLUE}${BOLD}Step 6: Configure gh-app-auth${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

echo -e "${CYAN}Configuring gh-app-auth with encrypted storage...${NC}"
echo

# Load private key
export GH_APP_PRIVATE_KEY="$(cat ~/.ssh/github-apps/test-app.pem)"

echo "Configuration:"
echo "  Organization: $TEST_ORG"
echo "  App ID: $APP_ID"
echo "  Installation ID: $INSTALLATION_ID"
echo "  Pattern: github.com/$TEST_ORG/*"
echo

gh app-auth setup \
  --app-id "$APP_ID" \
  --installation-id "$INSTALLATION_ID" \
  --patterns "github.com/$TEST_ORG/*" \
  --name "test-app"

unset GH_APP_PRIVATE_KEY

echo
echo -e "${GREEN}✓ gh-app-auth configured${NC}"
echo

# Configure git
echo "Configuring git credential helper..."
gh app-auth gitconfig --sync --global

echo -e "${GREEN}✓ Git credential helper configured${NC}"

read -p "Press Enter to continue..."
clear

#
# Step 7: Validation
#
echo -e "${BLUE}${BOLD}Step 7: Validation Tests${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

echo -e "${CYAN}Running validation tests...${NC}"
echo

if ./test/e2e/scripts/03-validate-basic-functionality.sh; then
    validation_success=true
else
    validation_success=false
fi

read -p "Press Enter to continue..."
clear

#
# Summary
#
cat << "EOF"
╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║                    Setup Complete!                             ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝
EOF

echo
echo -e "${BLUE}${BOLD}Configuration Summary:${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo
echo "  Organization: $TEST_ORG"
echo "  App ID: $APP_ID"
echo "  Installation ID: $INSTALLATION_ID"
echo "  Private Key: ~/.ssh/github-apps/test-app.pem"
echo

if [[ $validation_success == true ]]; then
    echo -e "${GREEN}✓ All validation tests passed${NC}"
else
    echo -e "${YELLOW}⚠ Some validation tests failed${NC}"
    echo "  Review output above for details"
fi

echo
echo -e "${BLUE}${BOLD}Next Steps:${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo
echo "1. Run advanced tests:"
echo "   ./test/e2e/scripts/04-run-advanced-tests.sh"
echo
echo "2. Try real workflows:"
echo "   git clone https://github.com/$TEST_ORG/private-test-repo"
echo
echo "3. Review documentation:"
echo "   docs/E2E_TESTING_TUTORIAL.md"
echo "   docs/TOKEN_CACHING.md"
echo
echo "4. When done testing, cleanup:"
echo "   ./test/e2e/scripts/99-cleanup.sh"
echo

# Save environment for future use
cat > /tmp/gh-app-auth-e2e-env.sh << ENVEOF
# E2E Test Environment Variables
# Source this file to restore environment:
#   source /tmp/gh-app-auth-e2e-env.sh

export TEST_ORG="$TEST_ORG"
export APP_ID="$APP_ID"
export INSTALLATION_ID="$INSTALLATION_ID"
ENVEOF

echo -e "${CYAN}Environment variables saved to:${NC}"
echo "  /tmp/gh-app-auth-e2e-env.sh"
echo
echo -e "${CYAN}To restore environment in a new shell:${NC}"
echo "  source /tmp/gh-app-auth-e2e-env.sh"
echo

echo -e "${GREEN}${BOLD}Happy testing!${NC}"
echo

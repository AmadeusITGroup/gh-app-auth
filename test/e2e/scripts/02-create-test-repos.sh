#!/usr/bin/env bash
#
# Create Test Repositories
# Creates public, private, and submodule test repositories
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Create E2E Test Repositories${NC}"
echo -e "${BLUE}========================================${NC}"
echo

# Check TEST_ORG is set
if [[ -z "$TEST_ORG" ]]; then
    echo -e "${RED}Error: TEST_ORG environment variable not set${NC}"
    echo
    echo "Usage:"
    echo "  export TEST_ORG=\"your-organization-name\""
    echo "  $0"
    exit 1
fi

echo -e "${BLUE}Organization: ${GREEN}$TEST_ORG${NC}"
echo

# Verify organization exists
echo -n "Verifying organization exists... "
if gh api "/orgs/$TEST_ORG" &> /dev/null; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo
    echo -e "${RED}Organization '$TEST_ORG' not found or not accessible${NC}"
    echo
    echo "Create an organization:"
    echo "  1. Go to: https://github.com/settings/organizations"
    echo "  2. Click 'New organization'"
    echo "  3. Follow the setup wizard"
    exit 1
fi

# Function to create repository
create_repo() {
    local name=$1
    local visibility=$2
    local description=$3
    
    echo -e "${BLUE}Creating $name ($visibility)...${NC}"
    
    # Check if repo already exists
    if gh api "/repos/$TEST_ORG/$name" &> /dev/null; then
        echo -e "  ${YELLOW}⚠ Repository already exists${NC}"
        read -p "  Delete and recreate? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo -n "  Deleting existing repository... "
            gh repo delete "$TEST_ORG/$name" --yes
            echo -e "${GREEN}✓${NC}"
        else
            echo "  Skipping $name"
            return 0
        fi
    fi
    
    # Create repository
    echo -n "  Creating repository... "
    local vis_flag
    if [[ "$visibility" == "public" ]]; then
        vis_flag="--public"
    else
        vis_flag="--private"
    fi
    
    gh repo create "$TEST_ORG/$name" \
        $vis_flag \
        --description "$description" \
        &> /dev/null
    echo -e "${GREEN}✓${NC}"
    
    # Clone repository
    echo -n "  Cloning repository... "
    local temp_dir="/tmp/gh-app-auth-e2e-$$"
    mkdir -p "$temp_dir"
    cd "$temp_dir"
    
    gh repo clone "$TEST_ORG/$name" &> /dev/null
    cd "$name"
    echo -e "${GREEN}✓${NC}"
    
    # Create initial content
    echo -n "  Creating initial content... "
    cat > README.md <<EOF
# $name

Test repository for gh-app-auth E2E testing.

- **Visibility**: $visibility
- **Purpose**: $description
- **Created**: $(date)

## Usage

This repository is part of the gh-app-auth E2E test suite.

See: https://github.com/AmadeusITGroup/gh-app-auth
EOF
    
    git add README.md
    git commit -m "Initial commit" &> /dev/null
    echo -e "${GREEN}✓${NC}"
    
    # Push to GitHub
    echo -n "  Pushing to GitHub... "
    git push -u origin main &> /dev/null
    echo -e "${GREEN}✓${NC}"
    
    # Clean up
    cd /tmp
    rm -rf "$temp_dir"
    
    echo -e "  ${GREEN}✓ $name created successfully${NC}"
    echo -e "  URL: ${BLUE}https://github.com/$TEST_ORG/$name${NC}"
    echo
}

# Create repositories
echo -e "${BLUE}=== Creating Test Repositories ===${NC}"
echo

create_repo "public-test-repo" "public" "Public repository for basic E2E testing"
create_repo "private-test-repo" "private" "Private repository requiring authentication"
create_repo "submodule-parent" "private" "Parent repository for submodule testing"

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✓ All test repositories created${NC}"
echo
echo -e "${BLUE}Created repositories:${NC}"
echo "  1. $TEST_ORG/public-test-repo (public)"
echo "  2. $TEST_ORG/private-test-repo (private)"
echo "  3. $TEST_ORG/submodule-parent (private)"
echo
echo -e "${BLUE}View repositories:${NC}"
echo "  gh repo list $TEST_ORG"
echo
echo -e "${BLUE}Next step:${NC}"
echo "  Create a GitHub App for this organization"
echo "  See: docs/E2E_TESTING_TUTORIAL.md#step-3-create-github-app"
echo

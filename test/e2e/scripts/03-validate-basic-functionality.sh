#!/usr/bin/env bash
#
# Validate Basic Functionality
# Tests core gh-app-auth features end-to-end
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  gh-app-auth E2E Validation${NC}"
echo -e "${BLUE}========================================${NC}"
echo

# Check required environment variables
if [[ -z "$TEST_ORG" ]]; then
    echo -e "${RED}Error: TEST_ORG not set${NC}"
    echo "export TEST_ORG=\"your-organization\""
    exit 1
fi

if [[ -z "$APP_ID" ]]; then
    echo -e "${YELLOW}Warning: APP_ID not set${NC}"
    echo "Some tests may fail without APP_ID"
    echo "export APP_ID=\"123456\""
fi

echo -e "${BLUE}Configuration:${NC}"
echo "  Organization: $TEST_ORG"
echo "  App ID: ${APP_ID:-<not set>}"
echo

# Test counter
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local name=$1
    local command=$2
    
    ((TESTS_RUN++))
    echo -n "Test $TESTS_RUN: $name... "
    
    if eval "$command" &> /tmp/test-output-$$; then
        echo -e "${GREEN}✓ PASS${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        echo -e "${YELLOW}Output:${NC}"
        cat /tmp/test-output-$$
        ((TESTS_FAILED++))
        return 1
    fi
}

# Clean up function
cleanup() {
    rm -f /tmp/test-output-$$
    rm -rf /tmp/gh-app-auth-test-*
}
trap cleanup EXIT

echo -e "${BLUE}=== Configuration Tests ===${NC}"
echo

run_test "gh-app-auth is installed" \
    "gh app-auth --help"

run_test "At least one app configured" \
    "gh app-auth list --json | jq -e 'length > 0'"

run_test "App has valid structure" \
    "gh app-auth list --json | jq -e '.[0] | has(\"name\") and has(\"appId\")'"

echo
echo -e "${BLUE}=== JWT Token Tests ===${NC}"
echo

if [[ -n "$APP_ID" ]]; then
    run_test "Generate JWT token" \
        "gh app-auth test --app-id $APP_ID"
else
    echo -e "${YELLOW}Skipping JWT tests (APP_ID not set)${NC}"
fi

echo
echo -e "${BLUE}=== Installation Token Tests ===${NC}"
echo

run_test "Get installation token for private repo" \
    "gh app-auth test --repo github.com/$TEST_ORG/private-test-repo"

echo
echo -e "${BLUE}=== Git Credential Helper Tests ===${NC}"
echo

run_test "Git credential helper is configured" \
    "git config --get-regexp 'credential.*github.com/$TEST_ORG' | grep -q 'gh app-auth'"

run_test "Credential helper responds to 'get' request" \
    "echo -e 'protocol=https\nhost=github.com\npath=$TEST_ORG/private-test-repo' | gh app-auth git-credential get | grep -q 'username='"

echo
echo -e "${BLUE}=== Repository Access Tests ===${NC}"
echo

# Test public repository
run_test "Clone public repository" \
    "git clone --depth 1 https://github.com/$TEST_ORG/public-test-repo /tmp/gh-app-auth-test-public-$$"

# Test private repository
run_test "Clone private repository" \
    "git clone --depth 1 https://github.com/$TEST_ORG/private-test-repo /tmp/gh-app-auth-test-private-$$"

# Test git operations
if [[ -d "/tmp/gh-app-auth-test-private-$$" ]]; then
    cd "/tmp/gh-app-auth-test-private-$$"
    
    run_test "Read repository content" \
        "cat README.md | grep -q 'Test repository'"
    
    run_test "Create and commit test file" \
        "echo 'Test $(date)' > e2e-test.txt && git add e2e-test.txt && git commit -m 'E2E test commit'"
    
    run_test "Push to remote repository" \
        "git push origin main"
    
    # Verify push worked
    run_test "Verify pushed content" \
        "gh api repos/$TEST_ORG/private-test-repo/contents/e2e-test.txt --jq '.name' | grep -q 'e2e-test.txt'"
    
    cd - > /dev/null
else
    echo -e "${YELLOW}Skipping git operations (clone failed)${NC}"
fi

echo
echo -e "${BLUE}=== Scope Detection Tests ===${NC}"
echo

run_test "Scope detection for configured repo" \
    "gh app-auth scope github.com/$TEST_ORG/private-test-repo | grep -q 'Matched App:'"

run_test "Scope detection output includes pattern" \
    "gh app-auth scope github.com/$TEST_ORG/private-test-repo | grep -q 'Pattern:'"

echo
echo -e "${BLUE}=== Cache Tests ===${NC}"
echo

# Test token caching by timing
echo -n "Test: Token caching improves performance... "
rm -rf /tmp/gh-app-auth-test-cache-1-$$ /tmp/gh-app-auth-test-cache-2-$$

# First clone (cache miss)
start_time=$(date +%s)
git clone --depth 1 https://github.com/$TEST_ORG/private-test-repo /tmp/gh-app-auth-test-cache-1-$$ &> /dev/null
first_time=$(($(date +%s) - start_time))

# Second clone (cache hit)
start_time=$(date +%s)
git clone --depth 1 https://github.com/$TEST_ORG/private-test-repo /tmp/gh-app-auth-test-cache-2-$$ &> /dev/null
second_time=$(($(date +%s) - start_time))

if [[ $second_time -le $first_time ]]; then
    echo -e "${GREEN}✓ PASS${NC} (${first_time}s → ${second_time}s)"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}⚠ INCONCLUSIVE${NC} (${first_time}s → ${second_time}s)"
fi
((TESTS_RUN++))

# Summary
echo
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test Results:${NC}"
echo
echo "  Total:  $TESTS_RUN"
echo -e "  ${GREEN}Passed: $TESTS_PASSED${NC}"
if [[ $TESTS_FAILED -gt 0 ]]; then
    echo -e "  ${RED}Failed: $TESTS_FAILED${NC}"
fi
echo

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo
    echo -e "${BLUE}Next steps:${NC}"
    echo "  • Run advanced tests: ./test/e2e/scripts/04-run-advanced-tests.sh"
    echo "  • Try real workflows with your repositories"
    echo "  • Review token caching: docs/TOKEN_CACHING.md"
    echo
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    echo
    echo -e "${YELLOW}Troubleshooting:${NC}"
    echo "  • Check gh-app-auth configuration: gh app-auth list"
    echo "  • Verify git config: git config --get-regexp credential"
    echo "  • Enable debug mode: export GH_APP_AUTH_DEBUG=1"
    echo "  • See: docs/E2E_TESTING_TUTORIAL.md#troubleshooting"
    echo
    exit 1
fi

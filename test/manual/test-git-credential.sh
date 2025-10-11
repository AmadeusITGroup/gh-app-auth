#!/bin/bash
# Manual test script for git credential helper
# This simulates how git calls the credential helper

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Find the binary
BINARY="${1:-./gh-app-auth}"

if [ ! -f "$BINARY" ]; then
    echo -e "${RED}Error: Binary not found at $BINARY${NC}"
    echo "Usage: $0 [path-to-binary]"
    echo "Example: $0 ./gh-app-auth"
    exit 1
fi

echo -e "${BLUE}=== Git Credential Helper Manual Test ===${NC}\n"

# Test 1: No config file
echo -e "${YELLOW}Test 1: No config file (should exit silently)${NC}"
export GH_APP_AUTH_CONFIG="/tmp/nonexistent-config-$$.yml"
echo -e "protocol=https\nhost=github.com\npath=myorg/myrepo\n" | "$BINARY" git-credential get
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Passed: Exited silently with no config${NC}\n"
else
    echo -e "${RED}✗ Failed: Should exit successfully${NC}\n"
fi
unset GH_APP_AUTH_CONFIG

# Test 2: Host only (no path)
echo -e "${YELLOW}Test 2: Host only - no path (should exit silently)${NC}"
echo -e "protocol=https\nhost=github.com\n" | "$BINARY" git-credential get
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Passed: Handled host-only request${NC}\n"
else
    echo -e "${RED}✗ Failed: Should handle host-only gracefully${NC}\n"
fi

# Test 3: Full URL with existing config
echo -e "${YELLOW}Test 3: Full URL with config (will fail without valid key, but tests flow)${NC}"
OUTPUT=$(timeout 5 bash -c "echo -e 'protocol=https\nhost=github.com\npath=myorg/myrepo\n' | '$BINARY' git-credential get 2>&1")
EXIT_CODE=$?
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✓ Passed: Successfully authenticated${NC}\n"
elif [ $EXIT_CODE -eq 1 ]; then
    echo -e "${YELLOW}⚠ Expected: No matching app or authentication failed${NC}"
    echo -e "${YELLOW}  (This is normal if no valid GitHub App is configured)${NC}\n"
else
    echo -e "${RED}✗ Failed: Unexpected exit code $EXIT_CODE${NC}\n"
fi

# Test 4: Store operation
echo -e "${YELLOW}Test 4: Store operation (should succeed silently)${NC}"
echo -e "protocol=https\nhost=github.com\npath=myorg/myrepo\nusername=test\npassword=token\n" | "$BINARY" git-credential store
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Passed: Store operation succeeded${NC}\n"
else
    echo -e "${RED}✗ Failed: Store should succeed${NC}\n"
fi

# Test 5: Erase operation
echo -e "${YELLOW}Test 5: Erase operation (should succeed silently)${NC}"
echo -e "protocol=https\nhost=github.com\npath=myorg/myrepo\n" | "$BINARY" git-credential erase
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Passed: Erase operation succeeded${NC}\n"
else
    echo -e "${RED}✗ Failed: Erase should succeed${NC}\n"
fi

# Test 6: Invalid operation
echo -e "${YELLOW}Test 6: Invalid operation (should fail)${NC}"
echo -e "protocol=https\nhost=github.com\n" | "$BINARY" git-credential invalid 2>/dev/null
if [ $? -ne 0 ]; then
    echo -e "${GREEN}✓ Passed: Rejected invalid operation${NC}\n"
else
    echo -e "${RED}✗ Failed: Should reject invalid operation${NC}\n"
fi

# Test 7: Multi-stage protocol simulation
echo -e "${YELLOW}Test 7: Multi-stage protocol (simulating git's behavior)${NC}"
echo -e "${BLUE}  Stage 1: Git asks for host only${NC}"
echo -e "protocol=https\nhost=github.com\n" | "$BINARY" git-credential get
if [ $? -eq 0 ]; then
    echo -e "${GREEN}  ✓ Stage 1 passed${NC}"
fi

echo -e "${BLUE}  Stage 2: Git asks with full path${NC}"
echo -e "protocol=https\nhost=github.com\npath=myorg/myrepo\n" | "$BINARY" git-credential get >/dev/null 2>&1
EXIT_CODE=$?
if [ $EXIT_CODE -eq 0 ] || [ $EXIT_CODE -eq 1 ]; then
    echo -e "${GREEN}  ✓ Stage 2 passed (exit code: $EXIT_CODE)${NC}\n"
else
    echo -e "${RED}  ✗ Stage 2 failed with unexpected exit code: $EXIT_CODE${NC}\n"
fi

# Test 8: Different URL formats
echo -e "${YELLOW}Test 8: Different URL formats${NC}"

echo -e "${BLUE}  Testing: github.com/org/repo${NC}"
echo -e "protocol=https\nhost=github.com\npath=org/repo\n" | "$BINARY" git-credential get >/dev/null 2>&1
echo -e "${GREEN}  ✓ Handled standard format${NC}"

echo -e "${BLUE}  Testing: github.com/org/repo.git${NC}"
echo -e "protocol=https\nhost=github.com\npath=org/repo.git\n" | "$BINARY" git-credential get >/dev/null 2>&1
echo -e "${GREEN}  ✓ Handled .git suffix${NC}"

echo -e "${BLUE}  Testing: enterprise GitHub${NC}"
echo -e "protocol=https\nhost=github.example.com\npath=org/repo\n" | "$BINARY" git-credential get >/dev/null 2>&1
echo -e "${GREEN}  ✓ Handled enterprise GitHub${NC}\n"

echo -e "${BLUE}=== All manual tests completed ===${NC}"
echo -e "\n${YELLOW}Note:${NC} Some tests may show 'expected failures' when no valid"
echo -e "GitHub App is configured. This is normal behavior."
echo -e "\nTo test with a real GitHub App:"
echo -e "  1. Run: gh app-auth setup --app-id <id> --key-file <key> --patterns 'github.com/myorg/*'"
echo -e "  2. Run: $0 $BINARY"
echo -e "  3. Test with: echo -e 'protocol=https\\nhost=github.com\\npath=myorg/myrepo\\n' | $BINARY git-credential get"

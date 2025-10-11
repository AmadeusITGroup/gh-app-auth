#!/bin/bash
# Test script for diagnostic logging functionality

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

echo -e "${BLUE}=== Diagnostic Logging Test ===${NC}\n"

# Create temporary log file
LOG_FILE=$(mktemp /tmp/gh-app-auth-test-XXXXXX.log)
echo -e "${YELLOW}Using log file: $LOG_FILE${NC}\n"

# Test 1: Logging disabled (default)
echo -e "${YELLOW}Test 1: Logging disabled by default${NC}"
unset GH_APP_AUTH_DEBUG_LOG
echo -e "protocol=https\nhost=github.com\npath=myorg/myrepo\n" | "$BINARY" git-credential get 2>/dev/null || true
if [ ! -f "$LOG_FILE" ] || [ ! -s "$LOG_FILE" ]; then
    echo -e "${GREEN}✓ Passed: No logging when disabled${NC}\n"
else
    echo -e "${RED}✗ Failed: Found logs when disabled${NC}\n"
fi

# Test 2: Enable logging with custom file
echo -e "${YELLOW}Test 2: Enable logging with custom file${NC}"
export GH_APP_AUTH_DEBUG_LOG="$LOG_FILE"

# Test different operations
echo -e "${BLUE}  Running get operation...${NC}"
echo -e "protocol=https\nhost=github.com\npath=myorg/myrepo\n" | "$BINARY" git-credential get 2>/dev/null || true

echo -e "${BLUE}  Running store operation...${NC}"
echo -e "protocol=https\nhost=github.com\npath=myorg/myrepo\nusername=test\npassword=fake\n" | "$BINARY" git-credential store 2>/dev/null || true

echo -e "${BLUE}  Running erase operation...${NC}"
echo -e "protocol=https\nhost=github.com\npath=myorg/myrepo\n" | "$BINARY" git-credential erase 2>/dev/null || true

# Check if log file exists and has content
if [ -s "$LOG_FILE" ]; then
    echo -e "${GREEN}✓ Passed: Log file created and has content${NC}"
    echo -e "${BLUE}  Log file size: $(wc -c < "$LOG_FILE") bytes${NC}\n"
else
    echo -e "${RED}✗ Failed: Log file empty or doesn't exist${NC}\n"
    exit 1
fi

# Test 3: Verify log structure
echo -e "${YELLOW}Test 3: Verify log structure and content${NC}"

# Check for session events
if grep -q "SESSION_START" "$LOG_FILE"; then
    echo -e "${GREEN}✓ Found SESSION_START${NC}"
else
    echo -e "${RED}✗ Missing SESSION_START${NC}"
fi

if grep -q "SESSION_END" "$LOG_FILE"; then
    echo -e "${GREEN}✓ Found SESSION_END${NC}"
else
    echo -e "${RED}✗ Missing SESSION_END${NC}"
fi

# Check for flow events
if grep -q "FLOW_START" "$LOG_FILE"; then
    echo -e "${GREEN}✓ Found FLOW_START events${NC}"
else
    echo -e "${RED}✗ Missing FLOW_START events${NC}"
fi

if grep -q "FLOW_STEP" "$LOG_FILE"; then
    echo -e "${GREEN}✓ Found FLOW_STEP events${NC}"
else
    echo -e "${RED}✗ Missing FLOW_STEP events${NC}"
fi

if grep -q "FLOW_SUCCESS\|FLOW_ERROR" "$LOG_FILE"; then
    echo -e "${GREEN}✓ Found FLOW_SUCCESS or FLOW_ERROR events${NC}"
else
    echo -e "${RED}✗ Missing FLOW_SUCCESS/FLOW_ERROR events${NC}"
fi

echo ""

# Test 4: Verify timestamp format
echo -e "${YELLOW}Test 4: Verify timestamp format${NC}"
if grep -qE "^\[[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}" "$LOG_FILE"; then
    echo -e "${GREEN}✓ Timestamps are properly formatted${NC}"
else
    echo -e "${RED}✗ Invalid timestamp format${NC}"
fi

# Test 5: Verify operation IDs
echo -e "${YELLOW}Test 5: Verify operation IDs${NC}"
if grep -qE "session_[0-9]+_[0-9]+_op[0-9]+" "$LOG_FILE"; then
    echo -e "${GREEN}✓ Operation IDs are properly formatted${NC}"
else
    echo -e "${RED}✗ Invalid operation ID format${NC}"
fi

# Test 6: Security - verify no clear tokens
echo -e "${YELLOW}Test 6: Security - verify no clear tokens logged${NC}"
if grep -qE "(password|token)=[^=]" "$LOG_FILE" && ! grep -q "token_hash=sha256:" "$LOG_FILE"; then
    echo -e "${RED}✗ Found potential clear text tokens${NC}"
    echo -e "${RED}  Suspicious lines:${NC}"
    grep -E "(password|token)=" "$LOG_FILE" | head -3
else
    echo -e "${GREEN}✓ No clear text tokens found${NC}"
fi

if grep -q "token_hash=sha256:" "$LOG_FILE"; then
    echo -e "${GREEN}✓ Found hashed tokens (secure)${NC}"
fi

echo ""

# Test 7: Multi-stage protocol logging
echo -e "${YELLOW}Test 7: Multi-stage protocol logging${NC}"

# Clear log for clean test
> "$LOG_FILE"

echo -e "${BLUE}  Stage 1: Host only${NC}"
echo -e "protocol=https\nhost=github.com\n" | "$BINARY" git-credential get 2>/dev/null || true

echo -e "${BLUE}  Stage 2: Full path${NC}"  
echo -e "protocol=https\nhost=github.com\npath=myorg/myrepo\n" | "$BINARY" git-credential get 2>/dev/null || true

# Check if both stages are logged
STAGE1_COUNT=$(grep -c "no_path_exit\|host_only" "$LOG_FILE" || echo "0")
STAGE2_COUNT=$(grep -c "parse_input.*path=" "$LOG_FILE" || echo "0")

if [ "$STAGE1_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ Stage 1 (host only) logged${NC}"
else
    echo -e "${YELLOW}⚠ Stage 1 not detected (may be normal)${NC}"
fi

if [ "$STAGE2_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ Stage 2 (full path) logged${NC}"
else
    echo -e "${RED}✗ Stage 2 not logged${NC}"
fi

echo ""

# Test 8: Log file permissions
echo -e "${YELLOW}Test 8: Log file permissions${NC}"
PERMS=$(stat -c "%a" "$LOG_FILE" 2>/dev/null || stat -f "%A" "$LOG_FILE" 2>/dev/null || echo "unknown")
if [ "$PERMS" = "600" ] || [ "$PERMS" = "0600" ]; then
    echo -e "${GREEN}✓ Log file has secure permissions (600)${NC}"
else
    echo -e "${YELLOW}⚠ Log file permissions: $PERMS (consider 600 for security)${NC}"
fi

echo ""

# Display sample log entries
echo -e "${YELLOW}Sample Log Entries:${NC}"
echo -e "${BLUE}--- First 10 lines ---${NC}"
head -10 "$LOG_FILE" | sed 's/^/  /'

echo -e "\n${BLUE}--- Flow steps ---${NC}"
grep "FLOW_STEP" "$LOG_FILE" | head -5 | sed 's/^/  /'

echo -e "\n${BLUE}--- Security examples ---${NC}"
grep -E "(token_hash|<redacted>)" "$LOG_FILE" | head -3 | sed 's/^/  /'

echo ""

# Summary
echo -e "${BLUE}=== Test Summary ===${NC}"
echo -e "Log file: $LOG_FILE"
echo -e "Total log entries: $(wc -l < "$LOG_FILE")"
echo -e "Log file size: $(wc -c < "$LOG_FILE") bytes"
echo -e "Unique sessions: $(grep -o 'session_[0-9]*_[0-9]*' "$LOG_FILE" | sort -u | wc -l)"
echo -e "Flow operations: $(grep -c "FLOW_START" "$LOG_FILE")"

echo -e "\n${GREEN}Diagnostic logging test completed!${NC}"
echo -e "\nTo analyze the full log:"
echo -e "  cat $LOG_FILE"
echo -e "\nTo extract a specific session:"
echo -e "  SESSION_ID=\$(grep 'SESSION_START' $LOG_FILE | tail -1 | grep -o 'session_[0-9]*_[0-9]*')"
echo -e "  grep \"\$SESSION_ID\" $LOG_FILE"
echo -e "\nTo clean up:"
echo -e "  rm $LOG_FILE"

# Keep log file for manual inspection
echo -e "\n${YELLOW}Note: Log file preserved for manual inspection${NC}"

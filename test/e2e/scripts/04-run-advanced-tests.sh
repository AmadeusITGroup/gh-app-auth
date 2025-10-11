#!/usr/bin/env bash
#
# Advanced E2E Tests
# Tests advanced scenarios: submodules, pattern priority, multi-org, etc.
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Advanced E2E Tests${NC}"
echo -e "${BLUE}========================================${NC}"
echo

# Check environment
if [[ -z "$TEST_ORG" ]]; then
    echo -e "${RED}Error: TEST_ORG not set${NC}"
    exit 1
fi

echo -e "${BLUE}Configuration:${NC}"
echo "  Organization: $TEST_ORG"
echo

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
run_test() {
    local name=$1
    shift
    local description=$1
    shift
    
    ((TESTS_RUN++))
    echo
    echo -e "${BLUE}Test $TESTS_RUN: $name${NC}"
    echo -e "${YELLOW}$description${NC}"
    echo
    
    if "$@"; then
        echo -e "${GREEN}✓ PASS${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Cleanup
cleanup() {
    rm -rf /tmp/gh-app-auth-advanced-test-*
}
trap cleanup EXIT

#
# Test 1: Submodule Support
#
test_submodules() {
    local parent_dir="/tmp/gh-app-auth-advanced-test-submodule-$$"
    
    echo "Setting up submodule test environment..."
    
    # Clone parent
    if ! git clone --depth 1 https://github.com/$TEST_ORG/submodule-parent "$parent_dir"; then
        echo "Failed to clone parent repository"
        return 1
    fi
    
    cd "$parent_dir"
    
    # Check if submodule already exists
    if [[ -f .gitmodules ]]; then
        echo "Submodule already configured, testing..."
    else
        echo "Adding submodule..."
        if ! git submodule add https://github.com/$TEST_ORG/private-test-repo submodule; then
            echo "Failed to add submodule"
            return 1
        fi
        
        git commit -m "Add submodule for E2E testing"
        git push origin main
    fi
    
    # Test recursive clone
    cd /tmp
    rm -rf "$parent_dir-clone"
    
    echo "Testing recursive clone..."
    if ! git clone --recurse-submodules https://github.com/$TEST_ORG/submodule-parent "$parent_dir-clone"; then
        echo "Failed to clone with submodules"
        return 1
    fi
    
    # Verify submodule contents
    if [[ -f "$parent_dir-clone/submodule/README.md" ]]; then
        echo "Submodule contents verified"
        return 0
    else
        echo "Submodule contents missing"
        return 1
    fi
}

run_test "Submodule Support" \
    "Verify gh-app-auth works with git submodules" \
    test_submodules

#
# Test 2: Pattern Matching
#
test_pattern_matching() {
    echo "Testing pattern matching..."
    
    # Test exact match
    local output=$(gh app-auth scope github.com/$TEST_ORG/private-test-repo)
    if ! echo "$output" | grep -q "Matched App:"; then
        echo "Failed to match configured repository"
        return 1
    fi
    
    # Test prefix match
    output=$(gh app-auth scope github.com/$TEST_ORG/any-other-repo)
    if ! echo "$output" | grep -q "Matched App:"; then
        echo "Failed to match repository with prefix pattern"
        return 1
    fi
    
    # Test scope detection
    if ! echo "$output" | grep -q "Scope:"; then
        echo "Scope information missing"
        return 1
    fi
    
    echo "Pattern matching verified"
    return 0
}

run_test "Pattern Matching" \
    "Verify URL prefix pattern matching works correctly" \
    test_pattern_matching

#
# Test 3: Concurrent Operations
#
test_concurrent_operations() {
    echo "Testing concurrent git operations..."
    
    local temp_base="/tmp/gh-app-auth-advanced-test-concurrent-$$"
    mkdir -p "$temp_base"
    
    # Launch 3 concurrent clones
    local pids=()
    for i in 1 2 3; do
        (git clone --depth 1 https://github.com/$TEST_ORG/private-test-repo "$temp_base/clone-$i" &> /dev/null) &
        pids+=($!)
    done
    
    # Wait for all to complete
    local failed=0
    for pid in "${pids[@]}"; do
        if ! wait "$pid"; then
            ((failed++))
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        echo "All concurrent operations succeeded"
        return 0
    else
        echo "$failed concurrent operations failed"
        return 1
    fi
}

run_test "Concurrent Operations" \
    "Verify gh-app-auth handles concurrent git operations" \
    test_concurrent_operations

#
# Test 4: Large File Operations
#
test_large_operations() {
    echo "Testing operations with larger file..."
    
    local work_dir="/tmp/gh-app-auth-advanced-test-large-$$"
    git clone --depth 1 https://github.com/$TEST_ORG/private-test-repo "$work_dir"
    cd "$work_dir"
    
    # Create a moderately sized file (1MB)
    dd if=/dev/zero of=large-file.bin bs=1024 count=1024 &> /dev/null
    
    git add large-file.bin
    git commit -m "Add large file for testing"
    
    if git push origin main; then
        echo "Large file push succeeded"
        
        # Verify
        if gh api repos/$TEST_ORG/private-test-repo/contents/large-file.bin &> /dev/null; then
            echo "Large file verified on remote"
            return 0
        fi
    fi
    
    echo "Large file operations failed"
    return 1
}

run_test "Large File Operations" \
    "Verify gh-app-auth works with larger files" \
    test_large_operations

#
# Test 5: Token Cache Performance
#
test_cache_performance() {
    echo "Measuring cache performance..."
    
    local base_dir="/tmp/gh-app-auth-advanced-test-cache-$$"
    
    # First operation (cache miss)
    echo "First clone (cache miss expected)..."
    local start=$(date +%s%N)
    git clone --depth 1 https://github.com/$TEST_ORG/private-test-repo "$base_dir-1" &> /dev/null
    local time1=$(( ($(date +%s%N) - start) / 1000000 ))
    
    # Wait a moment
    sleep 1
    
    # Second operation (cache hit)
    echo "Second clone (cache hit expected)..."
    start=$(date +%s%N)
    git clone --depth 1 https://github.com/$TEST_ORG/private-test-repo "$base_dir-2" &> /dev/null
    local time2=$(( ($(date +%s%N) - start) / 1000000 ))
    
    echo "First clone: ${time1}ms"
    echo "Second clone: ${time2}ms"
    
    # Cache hit should be faster or roughly equal
    if [[ $time2 -le $((time1 + 500)) ]]; then
        echo "Cache performance acceptable"
        return 0
    else
        echo "Cache may not be working (second operation slower)"
        return 1
    fi
}

run_test "Token Cache Performance" \
    "Verify token caching improves performance" \
    test_cache_performance

#
# Test 6: Error Handling
#
test_error_handling() {
    echo "Testing error handling..."
    
    # Test invalid repository
    if gh app-auth test --repo github.com/$TEST_ORG/nonexistent-repo-12345 &> /dev/null; then
        echo "Should have failed for nonexistent repository"
        return 1
    fi
    
    echo "Error handling verified"
    return 0
}

run_test "Error Handling" \
    "Verify graceful error handling" \
    test_error_handling

# Summary
echo
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Advanced Test Results:${NC}"
echo
echo "  Total:  $TESTS_RUN"
echo -e "  ${GREEN}Passed: $TESTS_PASSED${NC}"
if [[ $TESTS_FAILED -gt 0 ]]; then
    echo -e "  ${RED}Failed: $TESTS_FAILED${NC}"
fi
echo

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "${GREEN}✓ All advanced tests passed!${NC}"
    echo
    echo -e "${BLUE}Your gh-app-auth setup is fully functional.${NC}"
    echo
    echo "You can now:"
    echo "  • Use gh-app-auth in production workflows"
    echo "  • Integrate into CI/CD pipelines"
    echo "  • Configure additional organizations/apps"
    echo
    exit 0
else
    echo -e "${YELLOW}⚠ Some advanced tests failed${NC}"
    echo
    echo "Basic functionality may still work."
    echo "Review failed tests and see: docs/E2E_TESTING_TUTORIAL.md"
    echo
    exit 1
fi

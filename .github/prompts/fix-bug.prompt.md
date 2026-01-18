---
mode: agent
description: "Debug and fix a bug in gh-app-auth"
---

# Debug and Fix Bug

Systematically investigate and fix a bug in gh-app-auth.

## Investigation Process

1. **Understand the Bug**
   - What is the expected behavior?
   - What is the actual behavior?
   - When does it occur? (always, intermittent, specific conditions)
   - Can you reproduce it?

2. **Gather Information**
   - Check debug logs: `export GH_APP_AUTH_DEBUG_LOG=1`
   - Run with verbose: `gh --debug app-auth ...`
   - Check git trace: `GIT_TRACE=1 git clone ...`

3. **Locate the Issue**
   - Use grep to find relevant code
   - Check recent changes in git history
   - Review related test files for expected behavior

4. **Implement Fix**
   - Make minimal changes to fix the root cause
   - Don't fix symptoms - find the actual bug
   - Consider edge cases

5. **Verify Fix**
   - Write a test that reproduces the bug
   - Ensure the test fails before fix, passes after
   - Run full test suite: `make test`
   - Run quality checks: `make quality`

## Common Bug Categories

### Authentication Failures

- Check pattern matching in `pkg/matcher/`
- Verify token generation in `pkg/jwt/`
- Check keyring access in `pkg/secrets/`

### Configuration Issues

- Check YAML parsing in `pkg/config/`
- Verify path expansion
- Check environment variable handling

### Git Credential Problems

- Check stdin parsing in `cmd/git-credential.go`
- Verify output format matches git protocol
- Test multi-stage credential flow

### Token Expiration

- Check cache TTL in `pkg/cache/`
- Verify expiration detection
- Test automatic refresh

## Bug Fix Template

```go
// Before: describe the bug
// After: describe the fix

func fixedFunction() error {
    // Fix implementation
}
```

## Commit Message Format

```
fix: brief description of the fix

Fixes #issue-number (if applicable)

- Root cause: explain what was wrong
- Solution: explain the fix
```

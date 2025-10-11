# Troubleshooting Guide

Common issues and fixes for gh-app-auth, covering both GitHub Apps and Personal Access Tokens (PATs).

## Table of Contents

1. [General Diagnostics](#general-diagnostics)
2. [Credential Selection Issues](#credential-selection-issues)
3. [Bitbucket / Non-GitHub Hosts](#bitbucket--non-github-hosts)
4. [Git Credential Helper Problems](#git-credential-helper-problems)
5. [CI/CD Pitfalls](#cicd-pitfalls)
6. [Cleanup & Recovery](#cleanup--recovery)

---

## General Diagnostics

| Symptom | Possible Cause | Fix |
|---------|----------------|-----|
| `gh app-auth list` fails with “invalid configuration” | YAML edited manually and is malformed | Run `gh app-auth setup` to recreate, or fix the file and rerun `gh app-auth list`. |
| `gh app-auth test` returns 404 | Repo not accessible to the configured GitHub App | Check installation permissions for that org/repo. |
| `gh app-auth test` times out | Corporate proxies or firewall | Configure proxy for `gh` (see GitHub CLI docs) and retry. |

### Enable Debug Logging

```bash
gh --debug app-auth git-credential --pattern github.com/myorg/ get <<<'protocol=https
host=github.com'
```

- Logs are stored in `/tmp/gh-app-auth.*.log` (Linux/macOS) or `%TEMP%\gh-app-auth.*.log` (Windows).
- Use `grep FLOW_` to follow authentication steps.

---

## Credential Selection Issues

### 1. Git still prompts for username/password

- Run `gh app-auth list` and ensure your repo URL matches one of the `patterns`.
- Rerun `gh app-auth gitconfig --sync --global` (or `--local`).
- Verify `git config --show-origin --get-regexp credential` lists gh-app-auth helpers.

### 2. Wrong credential picked (PAT instead of App or vice versa)

- Remember that matching prefers the **longest prefix**, then the highest `priority`.
- Inspect `~/.config/gh/extensions/gh-app-auth/config.yml` to confirm pattern specificity.
- Adjust `--priority` (higher overrides) or use more specific patterns.
- Use `gh app-auth scope --repo <url>` to see which credential would be selected.

### 3. “No credential found” errors

- Ensure at least one App or PAT exists (`gh app-auth list`).
- Confirm the pattern includes a trailing slash (e.g., `github.com/myorg/`).
- For custom domains, include protocol-less host (e.g., `git.mycompany.com/`).

---

## Bitbucket / Non-GitHub Hosts

| Symptom | Fix |
|---------|-----|
| Bitbucket rejects credentials (HTTP 401) | Configure PAT with `--username <bitbucket_user>` and rerun `gh app-auth gitconfig --sync`. |
| Bitbucket uses SSH by default | Force HTTPS URLs (e.g., `https://bitbucket.example.com/scm/team/repo.git`). |
| Bitbucket PAT not found | Check `gh app-auth list --json` to confirm PAT entry exists and has matching pattern. |

### Example PAT Entry

```bash
gh app-auth setup \
  --pat bbpat_xxx \
  --username your-username \
  --patterns "bitbucket.example.com/" \
  --name "Bitbucket PAT" \
  --priority 40
```

---

## Git Credential Helper Problems

### “gh-app-auth executable not found”

- Ensure the extension is installed: `gh extension list | grep gh-app-auth`.
- Reinstall if needed: `gh extension install AmadeusITGroup/gh-app-auth`.

### Git helper conflicts with other helpers

- Run `git config --global --unset-all credential.helper` to clear legacy helpers before syncing.
- Use `gh app-auth gitconfig --clean` and `--sync` to rebuild.

### Credential cache not updating

- After editing config, rerun `gh app-auth gitconfig --sync` to propagate changes.
- For local repos, use `--local` to override global settings.

---

## CI/CD Pitfalls

| Issue | Resolution |
|-------|------------|
| Jenkins job fails after 1 hour | GitHub App tokens auto-refresh; ensure job reuses the same workspace and keyring is accessible. |
| GitHub Actions checkout still fails | Verify `gitconfig --sync` was run and `actions/checkout` uses HTTPS URLs. |
| PAT env vars exposed in logs | Provide tokens via GitHub/GitLab secrets and pass through env vars; `gh app-auth setup --pat ...` stores them securely afterward. |
| Bitbucket mirror builds | Configure both GitHub App and Bitbucket PAT in the same job; the helper picks based on host. |

---

## Cleanup & Recovery

### Remove All Credentials

```bash
gh app-auth gitconfig --clean --global
gh app-auth remove --all --force     # remove Apps
gh app-auth remove --all-pats --force # remove PATs
```

### Reset Configuration

1. Delete/rename `~/.config/gh/extensions/gh-app-auth/config.yml`.
2. Rerun `gh app-auth setup` for the desired Apps/PATs.
3. Re-sync git credential helpers.

### Inspect Stored Secrets

```bash
# macOS Keychain example
security find-generic-password -s "gh-app-auth:Org Automation App"
```

> Secrets are labeled `gh-app-auth:<name>` (matching the `name` field).

---

## Diagnostic Logging

### Enable Debug Logging

```bash
# Enable with default log location
export GH_APP_AUTH_DEBUG_LOG=1

# Or specify custom log file
export GH_APP_AUTH_DEBUG_LOG="/path/to/debug.log"

# Now run git operations
git clone https://github.com/myorg/private-repo
```

### View Logs

```bash
# Default location
tail -f ~/.config/gh/extensions/gh-app-auth/debug.log

# Search for specific events
grep "FLOW_ERROR" ~/.config/gh/extensions/gh-app-auth/debug.log
grep "app_matched" ~/.config/gh/extensions/gh-app-auth/debug.log
```

### Log Format

```
[TIMESTAMP] EVENT [OPERATION_ID] key=value...
```

**Event Types:**

| Event | Meaning |
|-------|---------|
| `SESSION_START` | New credential request started |
| `FLOW_STEP` | Step in authentication flow |
| `FLOW_SUCCESS` | Authentication succeeded |
| `FLOW_ERROR` | Authentication failed |
| `SESSION_END` | Request completed |

### Tracing a Failed Request

```bash
# 1. Enable logging
export GH_APP_AUTH_DEBUG_LOG=1

# 2. Reproduce the issue
git clone https://github.com/myorg/failing-repo

# 3. Find the session
grep "SESSION_START" ~/.config/gh/extensions/gh-app-auth/debug.log | tail -1

# 4. Extract full flow
grep "session_xxx" ~/.config/gh/extensions/gh-app-auth/debug.log
```

### Security Notes

- Tokens are **never** logged in plain text
- Token hashes (`sha256:...`) are logged for debugging
- URLs with credentials are sanitized
- Log files are created with `0600` permissions

---

## Need More Help?

- [CI/CD Integration Guide](ci-cd-guide.md)
- [Security Considerations](security.md)
- [GitHub Issues](https://github.com/AmadeusITGroup/gh-app-auth/issues)
- `gh --help app-auth`

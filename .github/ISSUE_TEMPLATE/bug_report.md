---
name: "🐛 Bug report"
about: Report a bug or unexpected behavior while using gh-app-auth
title: ''
labels: bug
assignees: ''
---

### Describe the bug

A clear and concise description of what the bug is.

### Affected version

Please run `gh-app-auth --version` and paste the output below.

### Steps to reproduce the behavior

1. Run command '...'
2. See error '....'
3. Expected result vs actual result

### Expected vs actual behavior

A clear and concise description of what you expected to happen and what actually happened.

### Configuration

Please provide your configuration (redact sensitive information):

- GitHub App ID:
- Patterns configured:
- Repository URL (if applicable):

### Environment

- OS:
- GitHub CLI version: `gh version`
- Go version (if building from source): `go version`

### Logs

Paste the activity from your command line. Redact sensitive information like tokens or private keys.

```
# Run with debug flag for verbose logs
gh-app-auth --debug [command]
```

### Additional context

Add any other context about the problem here, such as:

- Does this happen with all repositories or specific ones?
- Are you using GitHub.com or GitHub Enterprise?
- Any recent changes to your GitHub App configuration?

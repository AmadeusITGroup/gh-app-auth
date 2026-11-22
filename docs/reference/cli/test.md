---
title: gh app-auth test
description: Test gh-app-auth credentials
---

This page is generated from the output of `gh-app-auth test --help` in the checked-out source. To refresh it, run `npm run reference:generate` in `website/`.

## Help output

```text
Test authentication for a specific repository using either a GitHub App or Personal Access Token.

For GitHub Apps this command verifies that:
1. A GitHub App is configured for the repository pattern
2. JWT token generation works with the private key
3. Installation token can be retrieved
4. GitHub API access works with the token

For Personal Access Tokens this command verifies that:
1. A PAT is configured for the repository pattern
2. The token can be retrieved from secure storage
3. GitHub API access works with the token

Usage:
  gh-app-auth test [flags]

Examples:
  # Test authentication for a repository
  gh app-auth test --repo github.com/myorg/myrepo

  # Test with verbose output
  gh app-auth test --repo github.com/myorg/myrepo --verbose

  # Test current repository (if in git directory)
  gh app-auth test

Flags:
  -h, --help          help for test
  -R, --repo string   Repository to test (default: current repository)
  -v, --verbose       Show detailed test output

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output
```

## Related

- [Troubleshooting](../../troubleshoot/index.md)

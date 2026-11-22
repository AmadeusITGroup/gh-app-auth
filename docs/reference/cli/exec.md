---
title: gh app-auth exec
description: Run a command with managed GitHub authentication
---

This page is generated from the output of `gh-app-auth exec --help` in the checked-out source. To refresh it, run `npm run reference:generate` in `website/`.

## Help output

```text
Run a command with a short-lived token from a configured GitHub App
or PAT. Select credentials by repository, App ID, Client ID, or installation ID. Explicit
App selectors must match configured entries, and --repo must match the selected
App route. The token is exposed only to the child process through the environment
and is never printed by gh-app-auth.

Usage:
  gh-app-auth exec [flags] -- <command> [args...]

Examples:
  # Call the GitHub API for the current repository
  gh app-auth exec -- gh api repos/{owner}/{repo}

  # Run a gh command outside a repository
  gh app-auth exec --repo github.com/myorg/myrepo -- gh pr list

  # Run a repository-independent API command as a configured App installation
  gh app-auth exec --app-id 123456 --installation-id 789012 -- gh api /installation/repositories

Flags:
      --app-id int            Configured GitHub App ID to authenticate with
      --client-id string      Configured GitHub App Client ID to authenticate with
  -h, --help                  help for exec
      --installation-id int   GitHub App installation ID to authenticate with
  -R, --repo string           Repository to authenticate for (default: current repository unless an ID selector is used)

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output
```

## Related

- [Run a command](../../start-here/first-clone.md#run-a-command)
- [Security model](../../understand/security-model.md)

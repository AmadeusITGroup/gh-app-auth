---
title: gh app-auth gitconfig
description: Manage git credential helper configuration
---

This page is generated from the output of `gh-app-auth gitconfig --help` in the checked-out source. To refresh it, run `npm run reference:generate` in `website/`.

## Help output

```text
Manage git credential helper configuration for gh-app-auth.

This command automates the setup and cleanup of git credential helpers
based on your configured GitHub Apps and Personal Access Tokens (PATs).
It simplifies the manual process of configuring git to use gh-app-auth
for authentication.

The command can operate at two scopes:
  --global: Configure git globally (default)
  --local:  Configure git in the current repository only
  --auto:  Configure git in auto-mode (globally).
           A single GitHub App will be used to be configured automatically for each repository to clone

Usage:
  gh-app-auth gitconfig [flags]

Examples:
  # Sync git config with all configured apps
  gh app-auth gitconfig --sync

  # Clean up all gh-app-auth git configurations
  gh app-auth gitconfig --clean

  # Sync only for current repository
  gh app-auth gitconfig --sync --local

  # Enable auto-mode, 2 environment variables need to be set: GH_APP_PRIVATE_KEY_PATH and GH_APP_ID
  gh app-auth gitconfig --sync --auto

  # Clean global and check status
  gh app-auth gitconfig --clean --global

Flags:
      --auto     Configure git in auto-mode
      --clean    Remove all gh-app-auth git configurations
      --global   Configure git globally (default)
  -h, --help     help for gitconfig
      --local    Configure git in current repository only
      --sync     Sync git config with configured apps

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output
```

## Related

- [Git integration](../../guides/git-credential-helper.md)

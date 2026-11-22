---
title: gh app-auth remove
description: Remove credential configuration
---

This page is generated from the output of `gh-app-auth remove --help` in the checked-out source. To refresh it, run `npm run reference:generate` in `website/`.

## Help output

```text
Remove a configured GitHub App or Personal Access Token.

  This will remove the credential configuration and clear any cached secrets.

Usage:
  gh-app-auth remove [flags]

Aliases:
  remove, rm, delete

Examples:
  # Remove specific app by numeric App ID
  gh app-auth remove --app-id 123456

  # Remove specific app by Client ID
  gh app-auth remove --client-id Iv1.AbCdEfGhIjKlMn

  # Remove without confirmation
  gh app-auth remove --client-id Iv1.AbCdEfGhIjKlMn --force

  # Remove all configured apps
  gh app-auth remove --all

  # Remove a Personal Access Token
  gh app-auth remove --pat-name "My PAT"

  # Remove all Personal Access Tokens
  gh app-auth remove --all-pats

Flags:
      --all                Remove all configured GitHub Apps
      --all-pats           Remove all configured Personal Access Tokens
      --app-id int         GitHub App ID to remove
      --client-id string   GitHub App Client ID to remove
  -f, --force              Skip confirmation prompt
  -h, --help               help for remove
      --pat-name string    Personal Access Token name to remove

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output
```

## Related

- [Maintain credentials](../../guides/credentials.md#maintain-credentials)

---
title: gh app-auth list
description: List configured credentials
---

This page is generated from the output of `gh-app-auth list --help` in the checked-out source. To refresh it, run `npm run reference:generate` in `website/`.

## Help output

```text
List all configured GitHub Apps and Personal Access Tokens with their settings.

Shows the configured credential sources, their patterns, priorities, and status.

Usage:
  gh-app-auth list [flags]

Aliases:
  list, ls

Examples:
  # List all configured apps
  gh app-auth list

  # List with JSON output
  gh app-auth list --format json

  # Quiet output (just app IDs)
  gh app-auth list --quiet

Flags:
      --format string   Output format: table, json, yaml (default "table")
  -h, --help            help for list
  -q, --quiet           Only show app IDs
      --verify-keys     Verify that private keys are accessible

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output
```

## Related

- [Maintain credentials](../../guides/credentials.md#maintain-credentials)

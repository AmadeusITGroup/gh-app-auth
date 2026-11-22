---
title: gh app-auth token
description: Print a fresh GitHub App installation token
---

This page is generated from the output of `gh-app-auth token --help` in the checked-out source. To refresh it, run `npm run reference:generate` in `website/`.

## Help output

```text
Print a fresh installation token for a configured GitHub App.

This command does not use the active GitHub CLI account. Select the configured App
explicitly with --app-id or --client-id and provide the repository that must match
its configured route. On success, stdout contains only the token followed by a newline.

Usage:
  gh-app-auth token [flags]

Examples:
  # Print a fresh token for a configured App and repository
  gh app-auth token --app-id 123456 --repo github.com/myorg/myrepo

  # Use a Client ID and disambiguate a configured installation
  gh app-auth token \
    --client-id Iv1.AbCdEfGhIjKlMn \
    --installation-id 789012 \
    --repo github.com/myorg/myrepo

Flags:
      --app-id int            Configured GitHub App ID to authenticate with
      --client-id string      Configured GitHub App Client ID to authenticate with
  -h, --help                  help for token
      --installation-id int   Configured GitHub App installation ID to authenticate with
  -R, --repo string           Repository to authenticate for (required; must match the configured App route)

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output
```

## Related

- [Security model](../../understand/security-model.md)

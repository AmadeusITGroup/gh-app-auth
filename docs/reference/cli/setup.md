---
title: gh app-auth setup
description: Configure GitHub App authentication
---

This page is generated from the output of `gh-app-auth setup --help` in the checked-out source. To refresh it, run `npm run reference:generate` in `website/`.

> Do not put a real personal access token literal in a command or shell history. Even a variable passed with `--pat` is exposed in process arguments while the command runs. See [Personal access tokens](../../guides/credentials.md#personal-access-tokens).

## Help output

```text
Configure GitHub App authentication for specific repository patterns.

This command sets up a GitHub App for authentication with git operations.
You'll need the App ID or Client ID and private key file from your GitHub App settings.

Usage:
  gh-app-auth setup [flags]

Examples:
  # Basic setup with numeric App ID
  gh app-auth setup --app-id 123456 --key-file ~/.ssh/my-app.pem --patterns "github.com/myorg/*"

  # Basic setup with Client ID (preferred)
  gh app-auth setup --client-id Iv1.AbCdEfGhIjKlMn --key-file ~/.ssh/my-app.pem --patterns "github.com/myorg/*"

  # Setup with custom name and priority
  gh app-auth setup \
    --client-id Iv1.AbCdEfGhIjKlMn \
    --key-file ~/.ssh/my-app.pem \
    --patterns "github.com/myorg/*" \
    --name "Corporate App" \
    --priority 10

  # Multiple patterns
  gh app-auth setup \
    --client-id Iv1.AbCdEfGhIjKlMn \
    --key-file ~/.ssh/my-app.pem \
    --patterns "github.com/myorg/*,github.example.com/corp/*"

  # Setup a Personal Access Token for GitHub
  gh app-auth setup \
    --pat gh_your_token_here \
    --patterns "github.com/myorg/" \
    --name "My PAT" \
    --priority 10

  # Setup a Personal Access Token for Bitbucket (with username)
  gh app-auth setup \
    --pat your_bitbucket_token \
    --patterns "bitbucket.example.com/" \
    --username your_username \
    --name "Bitbucket PAT" \
    --priority 10

Flags:
      --app-id int            GitHub App ID (legacy; optional if --client-id is provided)
      --client-id string      GitHub App Client ID (preferred; optional if --app-id is provided)
  -h, --help                  help for setup
      --installation-id int   Installation ID (auto-detected if not provided)
      --key-file string       Path to private key file (or use GH_APP_PRIVATE_KEY env var)
      --name string           Friendly name for the GitHub App or PAT
      --pat string            Personal Access Token (for PAT setup)
      --patterns strings      Repository patterns to match (required)
      --priority int          Priority for pattern matching (higher = more priority) (default 5)
      --use-filesystem        Force filesystem storage instead of keyring
      --use-keyring           Store private key in OS keyring (default) (default true)
      --username string       Username for HTTP basic auth (optional, defaults to 'x-access-token' for GitHub)

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output
```

## Related

- [Get started](../../start-here/first-clone.md)
- [Personal access tokens](../../guides/credentials.md#personal-access-tokens)
- [Known limitations](../../troubleshoot/index.md#known-limitations)

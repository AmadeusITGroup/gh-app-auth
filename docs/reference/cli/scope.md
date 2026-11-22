---
title: gh app-auth scope
description: Manage GitHub App installation scope cache
---

This page is generated from the output of `gh-app-auth scope --help` in the checked-out source. To refresh it, run `npm run reference:generate` in `website/`.

## Help output

```text
Fetch and display GitHub App installation scope information.

The scope determines which repositories an app can access:
- "all": App has access to all repositories in the organization/account
- "selected": App has access only to specific repositories

Scope information is cached locally for 24 hours.

Usage:
  gh-app-auth scope [flags]

Flags:
  -h, --help      help for scope
      --refresh   Force refresh scope from GitHub API

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output
```

## Related

- [Credential routing](../../understand/credential-routing.mdx)

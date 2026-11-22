---
title: gh app-auth config
description: Show configuration file location and content
---

This page is generated from the output of `gh-app-auth config --help` in the checked-out source. To refresh it, run `npm run reference:generate` in `website/`.

## Help output

```text
Display the configuration file path and optionally its content.

The configuration file location follows this priority:
  1. GH_APP_AUTH_CONFIG environment variable (if set)
  2. Default: ~/.config/gh/extensions/gh-app-auth/config.yml

Usage:
  gh-app-auth config [flags]

Examples:
  # Show config file path
  gh app-auth config

  # Show config file path only
  gh app-auth config --path

  # Show config file content
  gh app-auth config --show

Flags:
  -h, --help   help for config
  -p, --path   Show only the config file path
  -s, --show   Show the config file content

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output
```

## Related

- [Configuration and environment](../configuration.md)
- [Environment variables](../configuration.md#environment-variables)

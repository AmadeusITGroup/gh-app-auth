---
title: gh app-auth migrate
description: Migrate private keys to encrypted storage
---

This page is generated from the output of `gh-app-auth migrate --help` in the checked-out source. To refresh it, run `npm run reference:generate` in `website/`.

## Help output

```text
Migrate private keys from filesystem to encrypted keyring storage.

This command helps you move your private keys from plain text files to
the OS-native encrypted keyring (Keychain on macOS, Credential Manager
on Windows, Secret Service on Linux).

The migration is safe and non-destructive - original key files are kept
as a fallback unless you specify --force.

Usage:
  gh-app-auth migrate [flags]

Examples:
  # Preview migration (dry-run)
  gh app-auth migrate --dry-run

  # Migrate keys to encrypted keyring
  gh app-auth migrate

  # Migrate to filesystem storage
  gh app-auth migrate --storage filesystem

  # Migrate and remove original key files
  gh app-auth migrate --force

Flags:
      --dry-run          Preview migration without making changes
      --force            Remove original key files after successful migration
  -h, --help             help for migrate
      --storage string   Target storage: keyring or filesystem (default "keyring")

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output
```

## Related

- [Maintain credentials](../../guides/credentials.md#maintain-credentials)

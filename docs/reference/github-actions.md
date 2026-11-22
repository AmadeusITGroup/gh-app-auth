---
title: GitHub Actions reference
description: Inputs, outputs and implementation limits of the setup-gh-app-auth and cleanup-gh-app-auth composite actions.
sidebar:
  order: 4
---

The repository ships two composite actions under `.github/actions/`. They exist for convenience;
the [GitHub Actions guide](../guides/ci/index.md#github-actions) shows the explicit, isolated approach this
documentation recommends. Read the limits below before depending on them.

Reference them by path and pin the ref, for example
`AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@<release tag or commit>`. Repository
examples use `@main`, which moves.

## Implementation limits

- **The cleanup hook is not a post-job hook.** "Register cleanup hook" writes a cleanup script, appends
  its path to `cleanup_commands.txt` (which nothing in the action ever executes), and sets a shell
  `trap` on `EXIT` inside the same step. That trap runs as soon as the step's shell exits, so it can
  **delete the credentials the setup step just configured before later steps run**, not merely leave them
  behind after the job. If you use the composite action and manage cleanup yourself, set
  `cleanup-on-exit: 'false'` and run the cleanup action with `if: always()`. The recommended
  [guides](../guides/ci/index.md#github-actions) avoid the composite action entirely.
- **Inputs are interpolated into shell scripts.** Values are expanded into the script text, so quotes
  or shell syntax in a value can break the step or run unintended commands. Even repository secrets can
  contain such characters, so passing a secret is not a guarantee of safe interpolation. The `apps-config`
  form places every private key directly in that text.
- **No filesystem key option.** The action runs `setup` with the key in `GH_APP_PRIVATE_KEY` and does not
  pass `--use-filesystem --key-file`. Where no OS keyring is available it relies on the automatic
  filesystem fallback, which is known to be unreliable. See
  [known limitations](../troubleshoot/index.md#known-limitations).
- **App ID only.** It takes `app-id`, not a Client ID.
- **Broad cleanup.** The cleanup action runs `gitconfig --clean --global` (all helper entries) and
  `remove --all`, which removes all configured Apps. It does **not** remove personal access tokens and
  does not delete files you created.
- **Not for shared machines.** Use it only on ephemeral runners.

## setup-gh-app-auth

Installs the GitHub CLI and `jq` if missing, installs the extension, registers one or several Apps,
and optionally syncs Git configuration. Use either the single-App inputs or `apps-config`.

### Inputs

| Input | Required | Default | Description |
| --- | --- | --- | --- |
| `app-id` | Single-App mode | none | Numeric GitHub App ID |
| `private-key` | Single-App mode | none | App private key in PEM format |
| `patterns` | Single-App mode | none | Comma-separated repository patterns |
| `app-name` | No | `GitHub App` | Friendly name for the App |
| `apps-config` | Multi-App mode | none | JSON array of objects with `app-id`, `private-key`, `patterns` and optional `app-name` |
| `cleanup-on-exit` | No | `true` | Registers the shell `trap` described above; set `'false'` if you manage cleanup yourself |
| `sync-git-config` | No | `true` | Runs `gh app-auth gitconfig --sync --global` after registration |
| `extension-version` | No | empty | Release to install with `--pin`; empty installs the latest |

Single-App mode needs `app-id`, `private-key` and `patterns`, and fails without all three.
`apps-config` is parsed with `jq`; each element is configured in turn, for example:

```json
[
  { "app-id": "123456", "private-key": "<PEM>", "patterns": "github.com/org-a/", "app-name": "Org A" },
  { "app-id": "234567", "private-key": "<PEM>", "patterns": "github.com/org-b/", "app-name": "Org B" }
]
```

Build it from secrets rather than writing keys into the workflow. The values above are placeholders.

### Outputs

| Output | Description |
| --- | --- |
| `config-path` | The default configuration path, `~/.config/gh/extensions/gh-app-auth/config.yml`; it ignores `GH_APP_AUTH_CONFIG` |
| `cleanup-registered` | `true` when the shell `trap` was registered |

## cleanup-gh-app-auth

Removes the Git helper entries and all configured Apps.

### Inputs

| Input | Required | Default | Description |
| --- | --- | --- | --- |
| `force` | No | `true` | Skips the confirmation prompt of `remove` |

### Outputs

This action defines no outputs.

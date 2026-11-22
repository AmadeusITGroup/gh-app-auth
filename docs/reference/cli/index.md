---
title: CLI overview
description: The gh app-auth commands, their purpose, and notes that apply to every command.
sidebar:
  order: 1
---

`gh app-auth` is the command group installed by the extension. The command pages below are generated
from the output of `--help` in the repository checkout. Run `gh app-auth <command> --help` for the
version you have installed.

## Commands

| Command | Purpose | Reference |
| --- | --- | --- |
| `setup` | Register a GitHub App or personal access token with repository patterns | [setup](setup.md) |
| `gitconfig` | Add or remove the Git credential helper entries | [gitconfig](gitconfig.md) |
| `list` | Show configured credentials (`--verify-keys`, `--format table\|json\|yaml`) | [list](list.md) |
| `test` | Exercise the credential for a repository | [test](test.md) |
| `exec` | Run a command with a short-lived token in its environment | [exec](exec.md) |
| `token` | Print a fresh installation token for a configured App | [token](token.md) |
| `config` | Show the configuration file path or content | [config](config.md) |
| `scope` | Fetch or refresh the cached installation scope | [scope](scope.md) |
| `migrate` | Move private keys between storage backends | [migrate](migrate.md) |
| `remove` | Remove Apps or PATs and their managed secrets | [remove](remove.md) |
| `completion` | Generate shell completion scripts | [completion](completion.md) |

`git-credential`, the helper that Git itself invokes, is hidden from help output. Do not call it by
hand; `gitconfig --sync` configures Git to run it. Its protocol and selection steps are in
[how credential routing works](../../understand/credential-routing.mdx).

## Flags every command shows

Every command lists `--config` and `--debug` under *Global Flags*. These flags are
registered but not wired the way their help text suggests:

- To choose a configuration file, set `GH_APP_AUTH_CONFIG`. Do not rely on `--config`.
- To capture diagnostics, set `GH_APP_AUTH_DEBUG_LOG` to an explicit absolute file path. Do not
  rely on `--debug`.

See [known limitations](../../troubleshoot/index.md#known-limitations) and
[environment variables](../configuration.md#environment-variables).

## Conventions in the examples

- `Iv1.example`, `example-org`, `.pem` paths and installation IDs are placeholders.
- Commands that contact GitHub (`test`, `exec`, `token`, `scope`) use the credentials you configured.
- `list --format` accepts `table`, `json` or `yaml`. There is no `--json` flag.
- `scope` has only `--refresh`; it has no `--repo` flag.

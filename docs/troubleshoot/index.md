---
title: Troubleshooting
description: Find your symptom, check the cause, and apply the fix for the most common gh-app-auth problems.
sidebar:
  order: 1
---

Start with the symptom. Each entry names the check that confirms the cause before the fix. For
logging, see [diagnostics](#diagnostics); for behaviors that are limits rather than bugs, see
[known limitations](#known-limitations).

## First checks

```bash
gh app-auth list --verify-keys
gh app-auth config --path
git config --show-origin --get-regexp '^credential\.'
```

These show which credentials exist and whether their keys are readable, which configuration file is
in use, and which Git credential helpers are configured.

## Symptoms

| Symptom | Likely cause | Check | Fix |
| --- | --- | --- | --- |
| Git still prompts for a username and password | The helper is not configured, or another helper answers first | `git config --show-origin --get-all credential.helper` | Run `gh app-auth gitconfig --sync`; reorder or remove the other helper |
| The wrong credential is used | Overlapping patterns, or a PAT priority higher than the App | `gh app-auth list` | Make patterns unambiguous; see [routing](../understand/credential-routing.mdx) |
| `setup` reports `Both --key-file and GH_APP_PRIVATE_KEY` | Both were provided | Environment and flags | Provide one. In CI use a file and a differently named variable |
| A configuration is rejected after `setup` with no keyring | Automatic filesystem fallback recorded no key path | `gh app-auth config --show` | Re-run `setup --use-filesystem --key-file <path>` |
| `list --verify-keys` cannot read a key | Missing keyring entry or key file | The `private_key_source` and path | Re-run `setup`, or `migrate` |
| Key file rejected | Group or world readable on POSIX | `ls -l` on the file | `chmod 600` the file |
| `test` fails at the installation step | The App is not installed on that owner, or the repository is outside its selection | The App's installation settings on GitHub | Install the App or add the repository |
| `test` fails with a clock-skew message | The machine clock is off | `date -u` against a trusted source | Fix the clock; see the [ADR](../adr/0001-jwt-clock-skew-margin.md) |
| A long `exec` command fails after about an hour | The installation token expired | Runtime of the child | Split the work, or run `exec` again for each phase |
| A PAT does not work on a headless machine | Filesystem PAT retrieval is not implemented | Whether an OS keyring exists | Use a machine with a keyring |
| `--json` is not recognized | The flag does not exist | `gh app-auth list --help` | Use `list --format json` |
| `scope --repo` is not recognized | `scope` has only `--refresh` | `gh app-auth scope --help` | Use `scope --refresh` |
| `--config` or `--debug` have no effect | The flags are not wired | n/a | Use `GH_APP_AUTH_CONFIG` and `GH_APP_AUTH_DEBUG_LOG` |

## Non-GitHub hosts

`test` is written for GitHub hosts and does not guarantee anything about other providers. For a
Bitbucket-style host, confirm the PAT works with the provider directly, make sure `--username` was set
when the provider needs a real account name, and end the PAT pattern with `/`.

## Git helper checks

- The helper path stored in Git's configuration is the extension binary. After you move or reinstall the
  extension, run `gh app-auth gitconfig --sync` again.
- Git uses the first helper in configuration order that answers. Path-specific entries written by
  `gitconfig --sync` come before generic host helpers.
- There is no cache-removal command for installation tokens, because they live only in process
  memory and disappear when the process exits.

## Diagnostics

gh-app-auth can write a diagnostic log of every credential flow. It is off by default and never
writes to the terminal, so it does not disturb Git's use of standard output.

### Turn it on

Set `GH_APP_AUTH_DEBUG_LOG` to an **explicit absolute file path**, then run the failing command:

```bash
export GH_APP_AUTH_DEBUG_LOG="$HOME/gh-app-auth-debug.log"
git clone https://github.com/example-org/private-repo.git
```

Any non-empty value enables logging, and the value is the file name. A value of `1` creates a file
named `1` in the current directory, not a default log. Unset the variable when you are done.

The log file is created with mode `0600` and appended to. Missing parent directories are created with
mode `0700`. If the directory cannot be created, the log goes to `gh-app-auth-debug.log` in the system
temporary directory, and if the file cannot be opened, logging is silently disabled. `--debug` is not
a substitute.

### Read the log

Each line has a timestamp, an event name and an operation ID, followed by `key=value` fields:

```text
[2026-01-01T10:00:00.000Z] SESSION_START [session_1_42_op1] pid=42 args=[...]
[2026-01-01T10:00:00.010Z] FLOW_START [session_1_42_op2] operation=git_credential flow=START
[2026-01-01T10:00:00.020Z] FLOW_STEP [session_1_42_op3] step=match_app flow=STEP url=...
[2026-01-01T10:00:00.030Z] FLOW_SUCCESS [session_1_42_op9] operation=git_credential flow=SUCCESS
```

| Event | Meaning |
| --- | --- |
| `SESSION_START` / `SESSION_END` | One process. Records the process ID and its arguments |
| `FLOW_START` / `FLOW_SUCCESS` / `FLOW_ERROR` | The begin and end of one credential operation |
| `FLOW_STEP` | One step: reading input, matching, generating credentials |

Useful steps to look for: `no_pattern_match`, `app_matched` and `app_matched_by_pattern` (which App was
chosen and by which mechanism), `no_match_exit` (nothing matched, so the helper stayed silent) and
`host_only_query` (Git provided a host without a path, so the helper returned empty output).

To follow one failure, search for the process ID from `SESSION_START` and read that session
top to bottom.

### Share a log safely

- Tokens are replaced by redacted secret metadata (type and length, not a cryptographic digest or unique
  identifier) rather than printed, and obvious secret-looking values are redacted. Do not rely on that alone.
- The log still contains **command-line arguments, repository URLs and file paths**. Read it before
  attaching it anywhere.
- Do not enable `GIT_TRACE`, `GIT_CURL_VERBOSE` or similar Git tracing together with real credentials
  unless you must. They can print authorization headers.
- Delete the log afterwards: `rm -f "$GH_APP_AUTH_DEBUG_LOG"`.

## Known limitations

These are implementation limits, not misconfigurations. Each entry offers a workaround.

### Automatic filesystem fallback for App keys

When no OS keyring is usable, `setup` may fall back to the filesystem and record
`private_key_source: filesystem` **without** recording a key path. Configuration validation then rejects
the entry. The managed fallback copy is raw bytes written with mode `0400` when the file is created, under
`~/.config/gh/extensions/gh-app-auth/secrets/`. An existing writable file keeps its mode, so check
permissions yourself. The copy is not encrypted.

**Do instead:** pass `--use-filesystem` together with `--key-file <path>`. The configuration keeps your
path, and nothing depends on the fallback.

### PATs need a keyring

Reading a personal access token from the filesystem store returns "not implemented". A machine without
an OS keyring cannot use a PAT.

**Do instead:** keep PATs on machines with a keyring, or use a GitHub App.

### GH_APP_PRIVATE_KEY and files

`GH_APP_PRIVATE_KEY` takes precedence over `--key-file`, and `setup --use-filesystem` rejects the
two together. Supplying a key through the environment does not make storage memory-only.

**Do instead:** in CI, supply the secret under another name and write it to a file you own.

### Long-running exec children

`exec` gives its child a token snapshot. Installation tokens expire after about an hour and nothing
renews them during the run.

**Do instead:** run `exec` per phase, or call it again before a long step.

### --config and --debug flags

The flags are registered on every command but are not wired as their help suggests.

**Do instead:** use `GH_APP_AUTH_CONFIG` for the configuration path and `GH_APP_AUTH_DEBUG_LOG` with an
absolute file path for logging.

### GH_APP_AUTH_CONFIG and secrets

The variable moves the configuration file but not the secrets directory, which stays under `HOME`.

**Do instead:** isolate `HOME` when you need isolated secrets, as the CI guides do.

### Key file permissions

POSIX checks reject group or world **read** access, and are skipped on Windows. They do not detect
symlinks, group write or later changes, and a key read from a stored configuration path is not
re-checked.

**Do instead:** use `chmod 600` on POSIX and set ACLs on Windows yourself.

### Composite action cleanup

`setup-gh-app-auth` registers its cleanup as a shell `trap` inside the setup step, not as a post-job hook.
The trap runs when that step's shell exits, so it can delete the credentials just configured before later
steps run. `cleanup-gh-app-auth` removes all Apps but not PATs.

**Do instead:** use the explicit steps in the [GitHub Actions guide](../guides/ci/index.md#github-actions).
Details are in the [GitHub Actions reference](../reference/github-actions.md).

### Equal priorities

When two credentials tie on priority, the winner depends on implementation details. Do not rely on
which one wins.

**Do instead:** make patterns unambiguous. See [routing](../understand/credential-routing.mdx).

## Recovery

> **Destructive scope.** `gitconfig --clean` removes the helper entries for all credentials, and
> `remove --all` removes all configured Apps and their managed secrets. Prefer removing one credential
> at a time. Outside an isolated, disposable CI job, run `remove --all` interactively without `--force`
> and read the prompt.

```bash
gh app-auth gitconfig --clean
gh app-auth remove --all
```

Then register the credentials again with `setup` and run `gitconfig --sync`. `remove --all` does not
remove PATs; run `remove --all-pats` or `remove --pat-name` separately, because target flags cannot be
combined. See [credential lifecycle](../guides/credentials.md#maintain-credentials).

## Still stuck

Capture a [diagnostic log](#diagnostics), review it for sensitive values and open an issue on
[GitHub](https://github.com/AmadeusITGroup/gh-app-auth/issues). For vulnerabilities, follow the
[security policy](../../.github/SECURITY.md) instead.

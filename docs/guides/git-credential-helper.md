---
title: Git integration
description: What gh app-auth gitconfig writes into your Git configuration, how to scope it, and how to remove it.
sidebar:
  order: 4
---

Git only calls gh-app-auth if its configuration says so. `gh app-auth gitconfig` writes and removes
that configuration for you.

## Sync and clean

```bash
gh app-auth gitconfig --sync
gh app-auth gitconfig --clean
```

- `--sync` adds a helper entry for every pattern of every configured App and PAT.
- `--clean` removes every helper entry that gh-app-auth wrote, for all credentials.
- `--global` (the default) edits your user Git configuration. `--local` edits the current
  repository only.
- `--auto` configures a single App from environment variables; see
  [environment variables](../reference/configuration.md#environment-variables).

Run `--sync` again after you add or change a credential.

## What gets written

For each pattern, `--sync` sets a credential helper in the matching Git credential context, along
these lines:

```text
[credential "https://github.com/example-org"]
    helper =
    helper = !"/path/to/gh-app-auth" git-credential --pattern github.com/example-org/
    useHttpPath = true
```

It clears earlier helpers in that context first, sets `useHttpPath` for hosts with path-specific
patterns so Git sends the repository path, and puts path-specific helpers ahead of generic host
helpers. Patterns that contain spaces or glob characters are quoted. On Windows the helper is a
generated wrapper script.

Inspect the result with:

```bash
git config --show-origin --get-regexp '^credential\.'
```

## Why the pattern is passed to the helper

The helper receives `--pattern` so it can honor the Git context that selected it. That shortcut and
the URL matcher are different mechanisms; see
[how credential routing works](../understand/credential-routing.mdx).

## Other helpers

Git consults helpers in configuration order and uses the first that answers. If another helper
answers first, gh-app-auth is never asked. Check the order if Git still prompts, and see
[troubleshooting](../troubleshoot/index.md).

## Remove credentials

> **Destructive scope.** `gitconfig --clean` removes the helper entries for **all** credentials, and
> `remove --all` removes **all** configured Apps and their managed secrets. Neither can be undone
> without registering the credentials again. Outside an isolated, disposable CI job, prefer removing
> one credential at a time, or run `remove --all` interactively and read the prompt.

```bash
gh app-auth gitconfig --clean
gh app-auth remove --all
```

The first command removes helper entries and the second removes configured Apps and their stored
secrets; it asks for confirmation unless you add `--force`, which belongs only in an isolated CI job.
`remove --all` does not remove PATs. Remove those in a separate invocation, with `remove --all-pats` or
`remove --pat-name`; target flags cannot be combined.

## Multiple organizations and submodules

A GitHub App installation belongs to one account. To reach repositories in several organizations,
register one App (or installation) per organization and give each its own pattern.

### Example: two organizations

```bash
gh app-auth setup --client-id Iv1.orga --key-file "$HOME/keys/org-a.pem" \
  --patterns 'github.com/org-a/' --name 'Org A'
gh app-auth setup --client-id Iv1.orgb --key-file "$HOME/keys/org-b.pem" \
  --patterns 'github.com/org-b/' --name 'Org B'
gh app-auth gitconfig --sync
```

Every value is a placeholder. After `--sync`, a clone of `github.com/org-a/...` uses `Org A` and a
clone of `github.com/org-b/...` uses `Org B`.

### Narrower patterns for exceptions

A longer pattern wins for App-to-App matching. Give one repository its own App without touching the
organization-wide entry:

```text
github.com/org-a/        -> Org A
github.com/org-a/secret  -> Secret App
```

Avoid two Apps with identical patterns, and avoid overlapping PAT patterns. Equal specificity makes
the outcome depend on implementation details. See
[how credential routing works](../understand/credential-routing.mdx).

### Submodules

Git asks for credentials separately for each submodule URL, so each is routed by its own URL:

```bash
git clone --recurse-submodules https://github.com/org-a/parent.git
```

If `parent` has a submodule in `org-b`, the helper selects `Org B` for that request. Nothing else
is needed, provided `--sync` ran after both Apps were registered.

### Verify each route

```bash
gh app-auth test --repo github.com/org-a/parent
gh app-auth test --repo github.com/org-b/child
```

`test` contacts GitHub with your real credentials, so run it deliberately. Use
`gh app-auth list` to review what is configured.

### Remember

Patterns choose which credential is sent. The installation's permissions and repository selection
limit what the token can do, so a route to an App that cannot see a repository still fails.

## GitHub Enterprise Server

For hosts other than `github.com`, gh-app-auth talks to the GitHub Enterprise Server REST API under
`https://<host>/api/v3`. For `github.com` it uses `api.github.com`.

### Register an App for your server

Create and install the GitHub App on the server, as described in
[create and install a GitHub App](credentials.md#github-apps), then include the host in the pattern:

```bash
gh app-auth setup \
  --client-id Iv1.example \
  --key-file "$HOME/keys/ghes-app.pem" \
  --patterns 'ghes.example.com/example-org/' \
  --name 'GHES App'
gh app-auth gitconfig --sync
```

Every value is a placeholder.

### Mix with github.com

Register separate entries for each host. Patterns start with the host, so
`github.com/example-org/` and `ghes.example.com/example-org/` never collide:

```bash
gh app-auth list
```

### Using exec against a server

`gh app-auth exec` injects the token as `GH_ENTERPRISE_TOKEN` for hosts other than github.com and
sets `GH_HOST`, so `gh` talks to the right server:

```bash
gh app-auth exec --repo ghes.example.com/example-org/private-repo -- gh api repos/{owner}/{repo}
```

### Checks

- The server must be reachable over HTTPS from the machine running Git.
- `gh app-auth test --repo ghes.example.com/example-org/private-repo` exercises the full path.
- Installation scope metadata is cached for 24 hours; refresh it with `gh app-auth scope --refresh`.

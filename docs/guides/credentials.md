---
title: Manage credentials
description: Create GitHub Apps, register personal access tokens, and inspect, migrate, rotate and remove credentials safely.
sidebar:
  order: 1
---

## Apps and PATs

### GitHub App credentials

A GitHub App is an identity owned by a user or organization. Four pieces work together:

| Piece | What it is | Secret? |
| --- | --- | --- |
| **App identity** | The Client ID (preferred) or numeric App ID | No |
| **Private key** | A PEM key used to sign JWTs as the App | Yes |
| **Installation** | The App installed on one account, with a repository selection and permissions | No |
| **Installation token** | A short-lived token for one installation | Yes |

gh-app-auth signs a JWT with the private key and exchanges it for an installation token. The token's reach
is set by the installation's permissions and repository selection, which gh-app-auth cannot widen or narrow.
Token lifetimes are in the [security model](../understand/security-model.md#tokens-and-caching).

Ownership and installation are different things. A GitHub App is **owned** by one account (a user or
an organization), and it can be **installed** on that account and on many others. Each installation is
specific to one account, so reaching repositories in several organizations means one installation per
organization, which can be installations of the same App or of different Apps. See
[several organizations](git-credential-helper.md#multiple-organizations-and-submodules).

### PAT identity

A personal access token (PAT) is a static credential issued to a person or service account by a
provider. gh-app-auth stores it and returns it unchanged. Its lifetime is decided by provider and
organization policy, its expiry date and revocation. PATs are the only option for providers that
cannot issue GitHub App tokens, such as Bitbucket, and they carry the permissions of the account that
created them. Registration is in [Personal access tokens](#personal-access-tokens) below.

### Identifiers in commands

`setup`, `exec`, `token` and `remove` accept `--client-id` and, for legacy entries, `--app-id`. `remove` also
accepts `--pat-name`, `--all`, `--all-pats` and `--force`. The `--installation-id` selector exists only on
`setup`, `exec` and `token`, where it disambiguates one of an App's installations. Client IDs appear on the
App's settings page, for example `Iv1.…`.

Selecting an App explicitly with `--app-id`, `--client-id` or `--installation-id` makes `exec` skip PATs. The
selector must match a configured entry, and `--repo`, when given, must match that App's route. For a
repository-independent API call, select the App and installation explicitly:

```bash
gh app-auth exec --client-id Iv1.example --installation-id 12345678 -- gh api /installation/repositories
```

The values are placeholders. Storage and exposure limits are in the
[security model](../understand/security-model.md), and the routing algorithm is in
[how credential routing works](../understand/credential-routing.mdx).

## GitHub Apps

### Who can do what

Creating an App, installing it on an organization and generating its private key can each require
an organization owner or an App manager. You may not be allowed to create Apps at all. If you are
not, ask an administrator to do the steps below and to hand you the Client ID and the key through an
approved secret-sharing or access procedure. Organization creation is not required: an existing
organization or a personal account can own the App.

### Choose the permissions

Grant the least privilege your workflow needs:

| Workflow | Repository permission |
| --- | --- |
| Clone and fetch | **Contents: Read-only** |
| Push | **Contents: Read and write** |
| API calls through `gh app-auth exec` | Only the permissions those calls need |

GitHub always adds **Metadata: Read-only**. Patterns in gh-app-auth never reduce what a token can do;
the installation's permissions and repository selection do. See
[how routing works](../understand/credential-routing.mdx).

### Create the App

1. In GitHub, open the account or organization settings, then **Developer settings**, then **GitHub Apps**,
   and choose **New GitHub App**.
2. Give it a name and a homepage URL. A webhook is not needed for gh-app-auth, so you can leave
   webhooks inactive.
3. Set the repository permissions from the table above.
4. Decide who can install it. An App limited to your own account or organization is the safest
   choice.
5. Create the App, and note the **Client ID** on its settings page. Client IDs identify the App
   publicly; they are not secrets.

### Install it

Install the App on the organization or account that owns the repositories, and choose **Only select
repositories** where you can. A fully scoped installation is easier to reason about than **All
repositories**. gh-app-auth detects the installation ID automatically; you can also pass
`--installation-id` to `setup`.

### Generate and store the private key

Generate a private key from the App's settings. GitHub downloads a `.pem` file once; it cannot be
shown again. Store it according to your organization's secret-handling rules. Then:

- On a workstation with an OS keyring, register the key with `setup --key-file`. See
  [your first authenticated clone](../start-here/first-clone.md).
- On an ephemeral CI runner, follow the [CI guide](ci/index.md), which writes the key to a job-owned file.

Protect the key file: `chmod 600` on Linux and macOS. gh-app-auth rejects a key file that is readable
by group or others on POSIX systems, and skips that check on Windows, where ACLs are your
responsibility.

### Check the result

```bash
gh app-auth setup --client-id Iv1.example --key-file "$HOME/keys/example-app.pem" \
  --patterns 'github.com/example-org/' --name 'Example App'
gh app-auth list --verify-keys
gh app-auth test --repo github.com/example-org/private-repo
```

All values are placeholders. If `test` fails, see [troubleshooting](../troubleshoot/index.md).

## Personal access tokens

### Register a PAT

```bash
read -rs PAT_VALUE
gh app-auth setup \
  --pat "$PAT_VALUE" \
  --patterns 'bitbucket.example.com/' \
  --username example-user \
  --name 'Bitbucket PAT'
unset PAT_VALUE
```

`read -rs` keeps the token out of your shell history and off the screen. It does not hide it from
other users of the machine: `--pat` is a command-line argument, so it can appear in the process list
while `setup` runs. Do this only on a machine you trust, and revoke the token if you suspect exposure.

For github.com or GitHub Enterprise hosts, omit `--username`: the default is `x-access-token`. Hosts
that need a real account name, such as Bitbucket Data Center, need `--username`.

### Storage and its limits

- A PAT is stored in the OS keyring. Reading a PAT from the filesystem store is **not
  implemented**, so a machine without a keyring cannot use a PAT. Do not plan a headless PAT setup.
- A PAT is static, and the provider, organization policy, expiry and revocation control how long it lives
  (see [PAT identity](#pat-identity)). Rotate it on your own schedule too.

### Use it

```bash
gh app-auth gitconfig --sync
git ls-remote https://bitbucket.example.com/project/repo.git HEAD
```

`git ls-remote` makes Git ask the helper for credentials and asks the provider to authenticate, without
cloning. It succeeds only if the token has the permissions the provider requires for that repository, so
check the token's scopes with the provider if it is rejected. Reserve `gh app-auth test` for GitHub
hosts: it calls the GitHub API and says nothing reliable about other providers. Clone with plain
`git clone` afterwards.

### How a PAT competes with an App

A PAT pattern matches by plain string prefix and must end with `/`; priority comparison and tie behavior are described in [how credential routing works](../understand/credential-routing.mdx).

## Maintain credentials

### Inspect

```bash
gh app-auth config --path
gh app-auth list --verify-keys
gh app-auth list --format json
```

`list --verify-keys` checks that each private key can be read. `--format` accepts `table`, `json` or
`yaml`. The default configuration file is `~/.config/gh/extensions/gh-app-auth/config.yml`; set
`GH_APP_AUTH_CONFIG` to use another file.

### Migrate key storage to the keyring

```bash
gh app-auth migrate --dry-run
gh app-auth migrate
```

`migrate` with the default `--storage keyring` copies each App's private key from its file into the OS
keyring. Check first that a keyring is actually available on the machine, because migration to a system
without one warns and can end up outside the keyring. Run `--dry-run` first to preview the move. Original
key files are **kept** unless you pass `--force`; do not add `--force` until `gh app-auth list --verify-keys`
succeeds and you have another approved copy of the key if your policy needs one.

### Moving to the filesystem

`migrate --storage filesystem` does **not** copy or export a key into a managed store. It only marks the
App's key source as `filesystem` and needs an existing configured `private_key_path`; without one it
reports an error and skips the App. The reliable way to use a file is to register it explicitly:

```bash
gh app-auth setup --use-filesystem --client-id Iv1.example \
  --key-file "$HOME/keys/example-app.pem" --patterns 'github.com/example-org/' --name 'Example App'
```

Use an existing, approved key file. See [known limitations](../troubleshoot/index.md#known-limitations).

### Rotate a private key

1. **Inventory every consumer of the old key first:** all registrations and their installation IDs, the
   keyring entry names or key files, every machine, and every CI job or other consumer that uses the old
   App key. `setup` does not migrate consumers globally.
2. In the App's settings on GitHub, generate a new private key. The old key stays valid, so the two overlap.
3. Update **every** consumer, keeping each entry's **identifier style, installation and name**: if you
   registered with `--client-id`, use `--client-id` again, with the same `--installation-id` if you set
   one, and the same `--name`. `setup` matches an existing entry by the App identifier (the Client ID when
   present) **and** the installation ID, merges the patterns you pass with the existing ones and replaces
   that one entry. Secrets are looked up by name, so do not rename an entry: a new name stores the key
   under a new lookup and can leave the old entry's secret orphaned.
4. Verify that **all** consumers use the new key before revoking anything, not just one local test.
   `gh app-auth list --verify-keys` only checks that a key is readable. `gh app-auth test --repo <repository>`
   attempts authentication; check that its reported identity is the expected App, and exercise every
   consumer's real workflow with the new key.
5. Only when the inventory is fully updated and tested, delete the old key in GitHub. Then retire the old
   key files according to your policy.

### Refresh scope metadata

`gh app-auth scope --refresh` re-reads which repositories an installation can see. The cached
metadata lasts 24 hours and is separate from installation tokens, which are cached in memory for 55
minutes.

### Remove

```bash
gh app-auth remove --client-id Iv1.example
gh app-auth remove --pat-name 'Bitbucket PAT'
gh app-auth gitconfig --clean
```

`remove` deletes the configuration entry and the secrets gh-app-auth manages for it. It does **not**
delete a key file you supplied with `--key-file`; delete that yourself if it should not remain.
`gitconfig --clean` removes every helper entry gh-app-auth wrote, for all credentials, so run
`gitconfig --sync` afterwards if some credentials remain.

`remove` has no installation selector. If an App has more than one configured installation, read the
confirmation prompt to check what will be removed, and do not use `--force` for an ambiguous removal.
Apps and PATs are removed with separate invocations; target flags cannot be combined.

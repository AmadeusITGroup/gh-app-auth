---
title: Continuous integration
description: Principles and a shared POSIX recipe for using gh-app-auth in disposable CI jobs, with provider-specific guides.
sidebar:
  order: 1
---

CI is where credential handling goes wrong most often: long-lived runners, shared home directories
and secrets interpolated into scripts. This guide keeps one rule at its center: **run gh-app-auth
in a disposable, isolated job, and clean up explicitly.**

| Provider | Section |
| --- | --- |
| GitHub Actions | [GitHub Actions](#github-actions) (complete, copyable workflow) |
| Jenkins | [Jenkins](#jenkins) (agent requirements and pipeline sketch) |
| GitLab CI | [GitLab CI](#gitlab-ci) (runner requirements and job sketch) |

This project has no automated test against Jenkins or GitLab. Treat those two sections as grounded
requirements and untested sketches, and validate them in your own environment.

## Principles

- **Disposable and isolated.** Use an ephemeral runner, container or agent, with a job-owned `HOME`
  and `GH_CONFIG_DIR`. Never edit a shared home directory, and never run broad credential cleanup on a
  shared machine.
- **Secrets through the environment.** Bind the private key and the Client ID to environment
  variables with your CI system's secret mechanism. Do not interpolate secret values into script
  text, such as a Groovy string or a shell line assembled by the CI system.
- **Explicit key file.** Write the key to a file you own with `umask 077`, then use
  `setup --use-filesystem --key-file`. This is the reliable headless route. Automatic filesystem
  fallback is not: see [known limitations](../../troubleshoot/index.md#known-limitations).
- **Supply the key under a different name.** `GH_APP_PRIVATE_KEY` takes precedence over the key file inside
  `setup`, and `setup --use-filesystem` rejects it together with `--key-file`. Name the CI variable
  `APP_PRIVATE_KEY`.
- **Always clean up, and know what cleanup covers.** `gitconfig --clean` removes every
  gh-app-auth helper entry, and `remove` deletes the managed secrets for the App you name. Neither
  deletes the key file you supplied, so delete it yourself.
- **Pin the extension.** Install a specific release, for example
  `gh extension install AmadeusITGroup/gh-app-auth --pin v0.2.0`, so a pipeline does not change
  behavior on its own. Installing needs outbound access to GitHub and a GitHub CLI token.
- **Cleanup is best effort.** A cleanup step does not run if the machine disappears. An ephemeral
  machine is the real guarantee.
- **Tokens expire.** An installation token lasts up to an hour and the helper caches it in memory for 55
  minutes per process. Git starts a fresh helper for each request. `exec` takes a snapshot for its
  child and does not renew it.

## Shared POSIX job recipe

The same shell logic works under any CI system that gives the job a disposable POSIX shell. It expects
`APP_CLIENT_ID`, `APP_PATTERN` and `GH_TOKEN` in the environment, plus **exactly one** of
`APP_PRIVATE_KEY` (the PEM text) or `APP_PRIVATE_KEY_FILE` (the path of a file that holds the PEM, which is
how GitLab File variables arrive).

```sh
set -eu
umask 077
if [ -n "${APP_PRIVATE_KEY:-}" ] && [ -n "${APP_PRIVATE_KEY_FILE:-}" ]; then
  echo "set only one of APP_PRIVATE_KEY and APP_PRIVATE_KEY_FILE" >&2
  exit 1
fi
if [ -z "${APP_PRIVATE_KEY:-}" ] && [ -z "${APP_PRIVATE_KEY_FILE:-}" ]; then
  echo "set APP_PRIVATE_KEY or APP_PRIVATE_KEY_FILE" >&2
  exit 1
fi

job_dir="$(mktemp -d)"
export HOME="$job_dir/home"
export GH_CONFIG_DIR="$HOME/.config/gh"
mkdir -p "$GH_CONFIG_DIR"
key_file="$job_dir/app-key"

cleanup() {
  status=$?
  gh app-auth gitconfig --clean --global || status=1
  gh app-auth remove --client-id "$APP_CLIENT_ID" --force || status=1
  rm -rf -- "$job_dir" || status=1
  exit "$status"
}
trap cleanup EXIT

if [ -n "${APP_PRIVATE_KEY_FILE:-}" ]; then
  cat -- "$APP_PRIVATE_KEY_FILE" > "$key_file"
else
  printf '%s\n' "$APP_PRIVATE_KEY" > "$key_file"
  unset APP_PRIVATE_KEY
fi

gh extension install AmadeusITGroup/gh-app-auth --pin v0.2.0
gh app-auth setup \
  --use-filesystem \
  --client-id "$APP_CLIENT_ID" \
  --key-file "$key_file" \
  --patterns "$APP_PATTERN" \
  --name 'CI App'
gh app-auth gitconfig --sync --global
```

Run your Git commands after this block in the same shell. Because `HOME` is job-owned, the cleanup also
removes the files `setup` wrote.

## GitHub Actions

This workflow is for GitHub-hosted `ubuntu-latest` runners only. Each job starts on a fresh virtual
machine that is discarded afterwards. It is not a recipe for shared or self-hosted machines.

The job uses its own home directory under `RUNNER_TEMP`, so the extension, its configuration and the
Git configuration it writes stay inside the job. The private key is written to a file owned by the job
and passed with `--use-filesystem --key-file`, which is the reliable path on a machine without an OS
keyring.

```yaml
name: private-clone

on: workflow_dispatch

permissions:
  contents: read

jobs:
  clone:
    runs-on: ubuntu-latest
    env:
      APP_CLIENT_ID: ${{ vars.APP_CLIENT_ID }}
    steps:
      - name: Prepare isolated home
        run: |
          job_home="$(mktemp -d "$RUNNER_TEMP/gh-app-auth-home.XXXXXX")"
          echo "HOME=$job_home" >> "$GITHUB_ENV"
          echo "GH_CONFIG_DIR=$job_home/.config/gh" >> "$GITHUB_ENV"

      - name: Configure gh-app-auth
        env:
          APP_PRIVATE_KEY: ${{ secrets.APP_PRIVATE_KEY }}
          GH_TOKEN: ${{ github.token }}
        run: |
          umask 077
          app_key_file="$(mktemp "$RUNNER_TEMP/app-key.XXXXXX")"
          echo "APP_KEY_FILE=$app_key_file" >> "$GITHUB_ENV"
          printf '%s\n' "$APP_PRIVATE_KEY" > "$app_key_file"
          gh extension install AmadeusITGroup/gh-app-auth --pin v0.2.0
          gh app-auth setup \
            --use-filesystem \
            --client-id "$APP_CLIENT_ID" \
            --key-file "$app_key_file" \
            --patterns 'github.com/example-org/' \
            --name 'CI App'
          gh app-auth gitconfig --sync --global

      - name: Clone a private repository
        run: git clone https://github.com/example-org/private-repo.git

      - name: Remove credentials
        if: always()
        run: |
          status=0
          gh app-auth gitconfig --clean --global || status=1
          gh app-auth remove --client-id "$APP_CLIENT_ID" --force || status=1
          if [ -n "${APP_KEY_FILE:-}" ]; then rm -f -- "$APP_KEY_FILE" || status=1; fi
          exit "$status"
```

The Client ID lives in a repository variable and the private key in a secret. Both reach the shell
through `env`, never through an expression inside the script. The key variable is deliberately not
named `GH_APP_PRIVATE_KEY`: that variable takes precedence over the key file inside `setup`, and
`setup --use-filesystem` rejects it together with `--key-file`.

The final step runs every cleanup command even when an earlier one fails, then reports the combined
status. It deletes the key file explicitly because `remove` deletes the credentials gh-app-auth manages,
not the file you supplied.

### What to know

- **Cleanup is best effort.** An `if: always()` step runs after failures and cancellations, but not if
  the runner itself is lost. On GitHub-hosted runners the discarded virtual machine is the real
  guarantee.
- **Cleanup is not selective.** `gitconfig --clean` removes every helper entry that gh-app-auth
  wrote, for all registered credentials, not only one App. `remove` deletes only the App you name.
  That is why this sample uses a fresh job-owned home.
- **Do not broaden cleanup.** Avoid cleanup that wipes unrelated `credential.helper` settings or deletes
  files you did not create.
- **Patterns select, they do not restrict.** The App's installation permissions limit what a token
  can do. See [credential routing](../../understand/credential-routing.mdx).

### The repository's composite action

The repository ships a composite action, `setup-gh-app-auth`. Its "Register cleanup hook" step sets a
shell `trap` inside the setup step, so it is **not** a post-job hook and can remove the credentials it
just configured before later steps run. Prefer the explicit steps above; if you do use the action, set
`cleanup-on-exit: 'false'` and manage cleanup yourself.
Its inputs, outputs and limitations are in the
[GitHub Actions reference](../../reference/github-actions.md).

## Jenkins

This project has no automated test against Jenkins. The requirements below follow from how gh-app-auth
works; the pipeline is a sketch to adapt and validate in your own environment.

### Agent requirements

- A **disposable** agent: a fresh container or virtual machine per build. Do not use a shared,
  long-lived agent.
- `gh`, `git`, a POSIX `sh` and outbound HTTPS access to GitHub (or your GitHub Enterprise Server).
- The extension installed inside the build, pinned to a release, with a job-owned `HOME` and
  `GH_CONFIG_DIR`, as in the [shared POSIX recipe](#shared-posix-job-recipe).
- Secrets stored as Jenkins credentials: the App's Client ID, its private key and a GitHub CLI
  token.

### Pipeline sketch

Bind the credentials with `withCredentials`, and put the shell logic in a **single-quoted** Groovy
string so Jenkins never interpolates a secret into script text:

```groovy
pipeline {
  agent { label 'disposable-gh-app-auth' }
  stages {
    stage('Clone private repository') {
      steps {
        withCredentials([
          string(credentialsId: 'gh-app-client-id', variable: 'APP_CLIENT_ID'),
          string(credentialsId: 'gh-app-private-key', variable: 'APP_PRIVATE_KEY'),
          string(credentialsId: 'gh-cli-token', variable: 'GH_TOKEN')
        ]) {
          sh '''
            export APP_PATTERN='github.com/example-org/'
            sh ./ci/gh-app-auth-job.sh
          '''
        }
      }
    }
  }
}
```

`ci/gh-app-auth-job.sh` is your checked-in copy of the shared recipe followed by your Git commands.
Credential IDs, label and pattern are placeholders.

### Cautions

- Do not use double-quoted Groovy strings around secrets.
- Do not echo environment variables or enable `set -x` while secrets are bound; Jenkins masking is
  best effort.
- Never mutate the agent's real home directory, and never run a broad credential wipe.
- Cleanup runs in the recipe's `EXIT` trap. It is best effort; disposable agents are the guarantee.

## GitLab CI

This project has no automated test against GitLab CI. The requirements below follow from how
gh-app-auth works; the job is a sketch to adapt and validate in your own environment. No container
image is named because this project does not publish or verify one.

### Runner requirements

- A **disposable** executor, such as a Docker or Kubernetes executor that starts a clean container
  per job. Do not use a shared shell executor.
- The job environment provides `gh`, `git` and a POSIX `sh`, and can reach GitHub over HTTPS.
- The extension is installed during the job, pinned to a release, with a job-owned `HOME` and
  `GH_CONFIG_DIR`, as in the [shared POSIX recipe](#shared-posix-job-recipe).
- `APP_CLIENT_ID` and `GH_TOKEN` are **masked, protected** variables of type *Variable*.
- The private key is a **protected File variable** named `APP_PRIVATE_KEY_FILE`. A multi-line PEM cannot be
  masked, so do not store it as a masked Variable. For a File variable the value your job sees is the
  **path** of a file that GitLab writes for the job. Restrict it to protected branches and tags, and see
  GitLab's documentation on [SSH keys in jobs](https://docs.gitlab.com/ci/jobs/ssh_keys/) and
  [CI/CD variables](https://docs.gitlab.com/ci/variables/).
- Because the key cannot be masked, it is **not hidden from job logs**. Never `cat` it to the log, and
  never enable shell tracing or CI debug tracing in a job that can read it.

### Job sketch

```yaml
clone-private-repository:
  stage: build
  tags:
    - disposable-gh-app-auth
  variables:
    APP_PATTERN: 'github.com/example-org/'
  script:
    - sh ./ci/gh-app-auth-job.sh
```

`ci/gh-app-auth-job.sh` is your checked-in copy of the shared recipe followed by your Git commands.
The recipe copies the contents of the File variable's path, with output redirected into a file
owned by the job (`cat -- "$APP_PRIVATE_KEY_FILE" > "$key_file"`), so nothing is printed. The job exports
no raw key text and interpolates no key into a script line. Tag and pattern are placeholders.

### Cautions

- Do not paste secret values into `script:` lines or `variables:` blocks; reference them as
  environment variables inside the script file.
- Do not print variables or enable `set -x` while secrets are available.
- Protect the variables and limit which branches and tags can read them; mask the ones that can be masked.
- Cleanup runs in the recipe's `EXIT` trap and is best effort; a disposable executor is the
  guarantee.

## Not covered

Shared or long-lived self-hosted machines need stronger isolation than a temporary home directory, and
that setup is out of scope here. Windows agents are not covered.

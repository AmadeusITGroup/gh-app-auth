# GitHub Actions for gh-app-auth

This directory holds two composite actions, `setup-gh-app-auth` and `cleanup-gh-app-auth`. They are
conveniences with real limitations; the documentation recommends explicit, isolated workflow steps instead.

> **Read the limits before you use them.**
>
> - The "cleanup hook" is a shell `trap` inside the setup step, not a post-job hook. It runs when that step
>   ends and can **remove the credentials the step just configured** before later steps run. If you use the
>   action, set `cleanup-on-exit: 'false'` and always run the cleanup action yourself with `if: always()`.
> - Inputs and secrets are interpolated into shell scripts, so a secret containing quotes or shell syntax can
>   break the step or run unintended commands.
> - On a headless runner without a keyring it relies on an automatic filesystem fallback that is known to be
>   unreliable.
> - Cleanup removes all configured Apps and the Git helper entries, but not personal access tokens.
> - Do not use either action on shared or long-lived machines.

For a complete isolated workflow, see [Continuous integration](../../docs/guides/ci/index.md#github-actions).
Inputs, outputs and defaults are in the [GitHub Actions reference](../../docs/reference/github-actions.md).

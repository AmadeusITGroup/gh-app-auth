# Legacy end-to-end scripts

> **Legacy. Not maintained and not the acceptance path.** The maintained live suite is the Go tests in
> [`test/e2e`](..), run as described in the
> [end-to-end testing tutorial](../../../docs/contribute/testing.md#live-end-to-end-tests). Use the Go suite instead.

These shell scripts are an older, manual way to set up and check a test environment. They are kept for
reference only, and they have known problems:

- `03-validate-basic-functionality.sh` uses flags that no longer exist (`list --json`,
  `test --app-id`) and removes files with broad patterns under `/tmp`.
- `04-run-advanced-tests.sh` also removes `/tmp` paths by pattern.
- `02-create-test-repos.sh` creates real repositories, and `99-cleanup.sh` deletes them and removes local
  files with broad `/tmp` patterns.
- `setup-wizard.sh` drives the other scripts.

| Script | Status |
| --- | --- |
| `setup-wizard.sh` | Legacy; drives the other scripts |
| `01-verify-prerequisites.sh` | Legacy |
| `02-create-test-repos.sh` | Legacy; mutates remote resources |
| `03-validate-basic-functionality.sh` | Legacy; uses stale flags and broad `/tmp` cleanup |
| `04-run-advanced-tests.sh` | Legacy; broad `/tmp` cleanup |
| `99-cleanup.sh` | Legacy; deletes repositories and uses broad `/tmp` cleanup |

Do not run them against real organizations or with real credentials. The scripts are intentionally left
unchanged in this documentation branch.

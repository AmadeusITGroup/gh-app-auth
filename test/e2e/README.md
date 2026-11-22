# End-to-end tests

> **These tests use real credentials and are not part of `make test`.** Every Go file here has the
> `//go:build e2e` constraint, so it is compiled only with `-tags=e2e`, and the suite fails without credentials
> instead of skipping. Run it only against the dedicated non-production test fixture.

- How to run it safely, including the required variables and the preflight:
  [Live end-to-end tests](../../docs/contribute/testing.md#live-end-to-end-tests).
- Provisioning the fixture, secrets and workflows (maintainers only):
  [End-to-end test infrastructure](../../docs/contribute/e2e-infrastructure.md).
- The older shell scripts in [`scripts/`](scripts/README.md) are legacy and are not the acceptance path.

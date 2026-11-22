# Contributing to gh-app-auth

Thank you for your interest in contributing to the GitHub App Authentication extension for GitHub CLI!

## Development Setup

Build, test and tooling instructions live in the [development guide](docs/contribute/development.md), and test
levels (including the opt-in live end-to-end suite) in the [testing guide](docs/contribute/testing.md). Go must
match [go.mod](go.mod).

## Development Guidelines

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comprehensive tests for new functionality
- Include documentation for public APIs

### Testing

- Write unit tests for all packages
- Include integration tests for CLI commands
- Test security-critical code paths thoroughly
- Use table-driven tests where appropriate

### Security Considerations

- Never log or expose private keys or tokens
- Validate file permissions for private key files
- Use secure temporary directories for testing
- Follow principle of least privilege

## Making Changes

### Adding New Commands

1. Create command file in `cmd/` directory
2. Implement cobra.Command with appropriate flags
3. Add command to root.go
4. Write comprehensive tests
5. Update documentation

### Adding New Features

1. Design the feature with security in mind
2. Implement in appropriate package
3. Add configuration options if needed
4. Write tests covering all code paths
5. Update documentation and examples

### Bug Fixes

1. Write a test that reproduces the bug
2. Implement the fix
3. Verify the test passes
4. Consider if documentation needs updates

## Pull Request Process

1. **Fork and branch**: Create a feature branch from main
2. **Implement**: Make your changes following the guidelines
3. **Test**: Ensure all tests pass and add new tests
4. **Document**: Update relevant documentation
5. **Submit**: Create a pull request with:
   - Clear description of the change
   - Reference to any related issues
   - Test evidence (screenshots, test output)

### PR Checklist

- [ ] Code follows project style guidelines
- [ ] Tests added for new functionality
- [ ] All tests pass
- [ ] Documentation updated
- [ ] Security considerations addressed
- [ ] Breaking changes clearly documented
- [ ] Commit messages follow conventional commits format

## Commit Message Guidelines

We use [Conventional Commits](https://www.conventionalcommits.org/) for all commit messages. This drives
automated changelog generation and version selection.

```text
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

- **Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`.
- **Scopes (optional):** `auth`, `config`, `cli`, `cache`, `security`, `docs`, `ci`, `deps`.
- Use the imperative mood, keep the subject under 72 characters, and keep one concern per commit.
- Mark a breaking change with `!` after the type or scope, or with a `BREAKING CHANGE:` footer.

Examples: `feat(auth): add JWT token caching`, `fix(config): handle missing config file`,
`docs: update installation instructions`.

## Code Review

All submissions require code review. Please:

- Be responsive to feedback
- Keep changes focused and atomic
- Write clear commit messages
- Rebase rather than merge when updating PRs

## Release Process

Releases are cut by merging the release-please pull request on `main`. The release workflow then stages
a draft release, builds and attaches every asset, gates on the end-to-end suite and provenance, and
publishes the draft as latest. Do not create tags or releases by hand. The single authoritative guide
is the [release runbook](docs/contribute/releasing.md).

## Architecture Decision Records

Breaking changes, new features, security decisions, and technology choices are recorded as ADRs in
[`docs/adr/`](docs/adr/README.md). That README holds the template, naming convention, and guidance on
when an ADR is warranted. Include the ADR in the same PR as the change it documents.

## Documentation

The documentation lives in [`docs/`](docs/README.md) and is built into a site from [`website/`](website/package.json).
Update the relevant page in the same pull request as a behavior change, and run `npm run check` in
`website/` when you touch it. The [documentation guide](docs/contribute/documentation.md) explains the
layout, link conventions and checks.

## Getting Help

- **Issues**: Open GitHub issues for bugs and feature requests
- **Discussions**: Use GitHub Discussions for questions
- **Security**: Report security issues via GitHub Security Advisories

## Resources

- [GitHub CLI Extension Development](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)
- [go-gh Library Documentation](https://pkg.go.dev/github.com/cli/go-gh/v2)
- [GitHub App Authentication](https://docs.github.com/en/developers/apps/building-github-apps/authenticating-with-github-apps)

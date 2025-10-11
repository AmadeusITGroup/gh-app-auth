# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of gh-app-auth GitHub CLI extension
- GitHub App JWT token generation with RSA private key support
- Installation access token exchange for repository-specific authentication
- Git credential helper protocol implementation
- Repository pattern matching for multi-organization support
- Secure token caching with automatic expiration (55 minutes)
- CLI commands: setup, list, remove, test, git-credential
- Comprehensive configuration system supporting YAML and JSON formats
- File permission validation for private keys (600/400 only)
- Path traversal protection for configuration files
- Input validation throughout the application
- Memory security with automatic cleanup of sensitive data
- Cross-platform support (Linux, macOS, Windows)
- Integration with GitHub CLI ecosystem using go-gh library

### Security
- Private key file permission validation prevents world-readable keys
- Secure token caching with automatic expiration
- Input sanitization prevents injection attacks
- Path validation prevents directory traversal
- Memory cleanup of sensitive data after use
- No logging of sensitive information (keys, tokens)

## [1.0.0] - 2024-10-11

### Added
- First stable release
- Complete GitHub App authentication workflow
- Production-ready security implementation
- Professional documentation and contribution guidelines
- CI/CD workflows with security scanning
- Development container configuration

[Unreleased]: https://github.com/wherka-ama/gh-app-auth/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/wherka-ama/gh-app-auth/releases/tag/v1.0.0

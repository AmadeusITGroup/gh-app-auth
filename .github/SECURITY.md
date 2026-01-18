# Security Policy

## Reporting Security Vulnerabilities

The gh-app-auth extension handles sensitive authentication data including private keys and access tokens. We take security seriously and appreciate responsible disclosure of security vulnerabilities.

**Please do not report security vulnerabilities through public GitHub issues, discussions, or pull requests.**

### How to Report

If you believe you have found a security vulnerability, please report it through one of these channels:

#### 1. Private Vulnerability Reporting (Preferred)

Use GitHub's private vulnerability reporting feature:

- Go to the [Security tab](https://github.com/AmadeusITGroup/gh-app-auth/security) of this repository
- Click "Report a vulnerability"
- Fill out the form with detailed information

#### 2. Email

Send an email to the maintainers with:

- **Subject**: `[SECURITY] Vulnerability Report - gh-app-auth`
- **Description**: Detailed description of the vulnerability
- **Impact**: Potential impact and affected components
- **Steps to Reproduce**: Clear steps to reproduce the issue
- **Proof of Concept**: Code or screenshots demonstrating the vulnerability

### What to Include

Please include as much information as possible:

- **Type of vulnerability** (e.g., authentication bypass, information disclosure, code injection)
- **Affected components** (e.g., JWT generation, private key handling, token caching)
- **Attack scenarios** (e.g., local file access, credential theft, privilege escalation)
- **Potential impact** (e.g., private key exposure, unauthorized repository access)
- **Affected versions** (if known)
- **Suggested mitigation** (if you have ideas)

### Security Scope

This security policy covers:

#### In Scope

- **Private Key Security**: RSA private key file handling and validation
- **Token Security**: JWT generation and installation token caching
- **Input Validation**: Command-line arguments and configuration parsing
- **File System Security**: Configuration file permissions and path traversal
- **Memory Security**: Sensitive data cleanup and memory management
- **Credential Helper Security**: Git credential protocol implementation

#### Out of Scope

- **GitHub.com Infrastructure**: Issues with GitHub's servers or services
- **GitHub CLI Core**: Issues in the base GitHub CLI (report to [cli/cli](https://github.com/cli/cli))
- **Operating System**: OS-level security issues
- **Network Security**: TLS/HTTPS implementation (handled by Go standard library)

### Response Timeline

- **Acknowledgment**: Within 48 hours of report
- **Initial Assessment**: Within 5 business days
- **Status Updates**: Weekly updates during investigation
- **Resolution Timeline**: Varies by severity and complexity

### Security Updates

Security updates will be:

- Released as soon as possible after verification
- Announced in release notes with appropriate detail
- Tagged with security advisory when applicable
- Communicated through GitHub Security Advisories

### Security Best Practices for Users

To use gh-app-auth securely:

#### Private Key Security

- **File Permissions**: Ensure private key files have restrictive permissions (600 or 400)
- **Key Storage**: Store private keys in secure locations (e.g., `~/.ssh/`)
- **Key Rotation**: Rotate GitHub App private keys regularly
- **No Sharing**: Never share or commit private keys to version control

#### Configuration Security

- **File Permissions**: Keep configuration files readable only by your user
- **Environment Variables**: Use environment variables in CI/CD instead of hardcoded values
- **Repository Patterns**: Use specific patterns to limit access scope

#### System Security

- **Regular Updates**: Keep gh-app-auth and dependencies updated
- **Minimal Permissions**: Run with least necessary privileges
- **Audit Logs**: Monitor authentication activity in GitHub App settings

### Known Security Considerations

#### By Design

- **Local Storage**: Tokens are cached locally for performance (55-minute expiration)
- **Memory Usage**: Sensitive data exists in memory during processing
- **File System Access**: Extension requires file system access for keys and config

#### Mitigations

- **Automatic Cleanup**: Tokens are cleared on expiration
- **Permission Validation**: File permissions are checked before key loading
- **Input Validation**: All inputs are validated to prevent injection attacks
- **Secure Defaults**: Conservative security settings by default
- **Vulnerability Monitoring**: Weekly security scans via CI/CD

### Security Features

Current security implementations:

- ✅ Private key file permission validation (600/400 only)
- ✅ Secure token caching with automatic expiration
- ✅ Path traversal protection for configuration files
- ✅ Input validation for all command-line arguments
- ✅ Memory cleanup of sensitive data
- ✅ No logging of sensitive information

---

**Thank you for helping keep gh-app-auth and the community safe!**

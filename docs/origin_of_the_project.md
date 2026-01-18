# Why This Project Exists

## Executive Summary

Enterprise CI/CD pipelines face significant challenges with GitHub authentication, particularly when working across multiple organizations. This project was created to address four key pain points:

1. **Robot accounts vs. GitHub Apps**: Governance vs. simplicity trade-offs
2. **Cross-org GitHub Apps with git submodules**: Multiple tokens complicate automation
3. **Jenkins/CI complexity with multi-org submodules**: Advanced configuration required
4. **Token expiry for long-running jobs**: 1-hour GitHub App token limit breaks builds

## The Problems

### 1) Robot Accounts vs. GitHub Apps

Organizations face a difficult choice:

| Approach | Pros | Cons |
|----------|------|------|
| **Robot users** | Simple to configure, behave like humans | Licensing costs, recertification overhead, security risks |
| **GitHub Apps** | Better governance, no license costs, audit trail | Token scope complexity, expiry management |

Many enterprises need a documented approach to support cross-org repository access without robot accounts.

### 2) GitHub Apps with Git Submodules Across Multiple Organizations

- GitHub App tokens are tied to installation scope
- Multi-org repos and submodules require multiple installations and tokens
- Even enterprise-level installs do not grant automatic repo access across organizations
- **Pain point**: A human can clone recursively with one token; automation needs N+1 secrets for N organizations

### 3) CI/CD Complexity with Multi-Org Git Submodules

CI pipelines (Jenkins, GitHub Actions, etc.) cloning across organizations require:

- Advanced submodule credential configuration
- Parent credentials propagation
- Multiple GitHub App installations with token refresh logic

### 4) Long-Running Jobs and 1-Hour Token Expiry

- GitHub App installation tokens expire after 1 hour
- Long builds or organization scans fail mid-run unless tokens are refreshed
- Most CI plugins warn about token validity but don't handle refresh automatically
- Community confirms this limit is non-extendable by GitHub

## How gh-app-auth Solves These Problems

This extension provides:

1. **Unified credential management** - Single tool for GitHub Apps AND PATs
2. **Automatic token refresh** - Transparent handling of 1-hour token expiry
3. **Multi-org support** - Pattern-based routing to different credentials
4. **Git integration** - Native credential helper for seamless git operations
5. **Secure storage** - OS-native keyring integration for private keys and tokens
6. **CI/CD ready** - Designed for both local development and automated pipelines

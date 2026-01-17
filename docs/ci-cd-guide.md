### Adding Personal Access Tokens (e.g., Bitbucket PAT)

Some workflows require PATs (either for user-scoped GitHub access or because the repository lives outside GitHub). Add PAT configuration alongside Apps:

```yaml
    - name: Configure Bitbucket PAT
      env:
        BITBUCKET_PAT: ${{ secrets.BB_HTTP_ACCESS_TOKEN }}
        BITBUCKET_USERNAME: ${{ secrets.BB_USERNAME }}
      run: |
        gh app-auth setup \
          --pat "$BITBUCKET_PAT" \
          --patterns "bitbucket.example.com/" \
          --username "$BITBUCKET_USERNAME" \
          --name "Bitbucket PAT" \
          --priority 40

        # gh app-auth gitconfig --sync updates git credential helpers for both Apps and PATs
        gh app-auth gitconfig --sync --global
```

PAT entries share the same priority rules as Apps. For example, set a higher priority PAT to override Automation for specific repos, or keep PAT priority lower so CI prefers GitHub App tokens.
# CI/CD Integration Guide

This guide provides comprehensive examples and best practices for using gh-app-auth in CI/CD environments.

> **New! Dual Auth:** gh-app-auth now routes both GitHub App credentials **and** Personal Access Tokens (PATs). PATs live beside Apps in the encrypted keyring, obey the same pattern/priority rules, and support non-GitHub providers (Bitbucket Server/Data Center) via the optional `--username` flag.

## Table of Contents

- [Overview](#overview)
- [GitHub Actions](#github-actions)
- [Jenkins](#jenkins)
- [GitLab CI](#gitlab-ci)
- [Common Patterns](#common-patterns)
- [Troubleshooting](#troubleshooting)

## Overview

The gh-app-auth extension solves several critical CI/CD challenges:

### Problem 1: Robot Accounts vs. GitHub Apps
**Challenge**: Organizations debate between robotic user accounts (which behave like humans) and GitHub Apps (preferred for governance).

**Solution**: gh-app-auth enables GitHub Apps to work seamlessly in CI/CD, eliminating the need for robot accounts while maintaining ease of use. When automation must impersonate a human (e.g., release manager approvals) or access external Git hosting, PAT support bridges the gap without sacrificing secure storage.

### Problem 2: Cross-Organization Repository Access
**Challenge**: GitHub App tokens are scoped to specific installations. Multi-org repositories and submodules require multiple installations and tokens.

**Solution**: Configure multiple GitHub Apps **and/or PATs** with pattern matching. The extension automatically selects the correct credential based on repository URL and priority (e.g., App for CI, PAT for personal repos, Bitbucket PAT for legacy code).

### Problem 3: Git Submodules Across Organizations
**Challenge**: Cloning repositories with submodules across multiple organizations requires complex credential management.

**Solution**: Configure git credential helper once. All git operations (including submodules) automatically use the correct GitHub App credentials.

### Problem 4: Long-Running Jobs and Token Expiry
**Challenge**: GitHub App installation tokens expire after 1 hour, causing long-running jobs to fail mid-execution.

**Solution**: The extension automatically refreshes tokens on-demand. Jobs can run indefinitely without manual token management. PATs are retrieved on each request directly from the keyring (ideal when a PAT is required for third-party Git or Bitbucket pipelines).

### Problem 5: Mixed Git Providers (GitHub + Bitbucket)
**Challenge**: Enterprise portfolios often include GitHub and Bitbucket Server/Data Center. Credential helpers must send a real username to Bitbucket but `x-access-token` to GitHub.

**Solution**: Configure PAT entries with `--username <bitbucket_user>` for Bitbucket hosts while leaving GitHub entries untouched. gh-app-auth automatically outputs the correct username/password pair per host.

## GitHub Actions

### Single Organization Setup

```yaml
name: CI with GitHub App

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    
    steps:
      - name: Setup GitHub App Authentication
        uses: actions/setup-gh-app-auth@v1  # Custom action (see below)
        with:
          app-id: ${{ secrets.GITHUB_APP_ID }}
          private-key: ${{ secrets.GITHUB_APP_PRIVATE_KEY }}
          organization: ${{ github.repository_owner }}
      
      - name: Checkout with submodules
        run: |
          git clone --recurse-submodules \
            https://github.com/${{ github.repository }}.git repo
          cd repo
      
      - name: Build
        run: |
          cd repo
          make build
```

### Auto Mode (Simplified Setup)

For CI/CD environments where a single GitHub App has access to all needed repositories, use `--auto` mode for simplified configuration:

```yaml
name: CI with Auto Mode

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    
    steps:
      - name: Install gh-app-auth
        run: gh extension install AmadeusITGroup/gh-app-auth
      
      - name: Configure Auto Mode
        env:
          GH_APP_ID: ${{ secrets.GITHUB_APP_ID }}
          GH_APP_PRIVATE_KEY_PATH: /tmp/app-key.pem
        run: |
          echo "${{ secrets.GITHUB_APP_PRIVATE_KEY }}" > /tmp/app-key.pem
          chmod 600 /tmp/app-key.pem
          gh app-auth gitconfig --sync --auto
      
      - name: Checkout (works for any org the app can access)
        run: |
          git clone --recurse-submodules \
            https://github.com/any-org/any-repo.git repo
      
      - name: Cleanup
        if: always()
        run: rm -f /tmp/app-key.pem
```

**When to use auto mode:**
- Single GitHub App with broad access (e.g., organization-wide installation)
- Dynamic environments where repository patterns aren't known in advance
- Simplified setup without per-organization pattern configuration

### Custom Composite Action

Create `.github/actions/setup-gh-app-auth/action.yml`:

```yaml
name: 'Setup GitHub App Authentication'
description: 'Configure gh-app-auth extension for CI/CD'

inputs:
  app-id:
    description: 'GitHub App ID'
    required: true
  private-key:
    description: 'GitHub App Private Key'
    required: true
  organization:
    description: 'GitHub Organization'
    required: true

runs:
  using: 'composite'
  steps:
    - name: Install GitHub CLI
      uses: cli/gh@v2
      
    - name: Install gh-app-auth extension
      shell: bash
      run: gh extension install AmadeusITGroup/gh-app-auth
    
    - name: Configure GitHub App
      shell: bash
      env:
        APP_ID: ${{ inputs.app-id }}
        APP_KEY: ${{ inputs.private-key }}
        ORG: ${{ inputs.organization }}
      run: |
        echo "$APP_KEY" > /tmp/app-key.pem
        chmod 600 /tmp/app-key.pem
        gh app-auth setup \
          --app-id "$APP_ID" \
          --key-file /tmp/app-key.pem \
          --patterns "github.com/$ORG/*"
        git config --global credential."https://github.com/$ORG".helper \
          "!gh app-auth git-credential"
        rm -f /tmp/app-key.pem
```

### Matrix Build with Multiple Organizations

```yaml
name: Multi-Org Matrix Build

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - org: org1
            app_id_secret: ORG1_APP_ID
            key_secret: ORG1_APP_KEY
          - org: org2
            app_id_secret: ORG2_APP_ID
            key_secret: ORG2_APP_KEY
    
    steps:
      - name: Configure GitHub App for ${{ matrix.org }}
        env:
          APP_ID: ${{ secrets[matrix.app_id_secret] }}
          APP_KEY: ${{ secrets[matrix.key_secret] }}
        run: |
          gh extension install AmadeusITGroup/gh-app-auth
          echo "$APP_KEY" > /tmp/key.pem
          chmod 600 /tmp/key.pem
          gh app-auth setup \
            --app-id "$APP_ID" \
            --key-file /tmp/key.pem \
            --patterns "github.com/${{ matrix.org }}/*"
          git config --global credential."https://github.com/${{ matrix.org }}".helper \
            "!gh app-auth git-credential"
          rm -f /tmp/key.pem
      
      - name: Test access to ${{ matrix.org }}
        run: |
          gh app-auth test --repo github.com/${{ matrix.org }}/test-repo
          gh app-auth test --repo bitbucket.org/${{ matrix.org }}/test-repo --username ${{ secrets.BITBUCKET_USERNAME }}
```

## Jenkins

### Declarative Pipeline with Credentials

```groovy
pipeline {
    agent any
    
    parameters {
        string(name: 'REPO_URL', defaultValue: 'https://github.com/myorg/my-repo.git', 
               description: 'Repository URL to clone')
        booleanParam(name: 'INCLUDE_SUBMODULES', defaultValue: true, 
                    description: 'Clone submodules recursively')
    }
    
    environment {
        GITHUB_APP_ID = credentials('github-app-id')
        GITHUB_APP_PRIVATE_KEY = credentials('github-app-private-key')
        GH_TOKEN = '' // Prevent gh CLI from using other auth
    }
    
    stages {
        stage('Setup') {
            steps {
                script {
                    sh '''
                        # Ensure GitHub CLI is installed
                        if ! command -v gh &> /dev/null; then
                            echo "Installing GitHub CLI..."
                            type -p curl >/dev/null || (sudo apt update && sudo apt install curl -y)
                            curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | \
                                sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
                            sudo chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg
                            echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | \
                                sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
                            sudo apt update
                            sudo apt install gh -y
                        fi
                        
                        # Install extension
                        gh extension install AmadeusITGroup/gh-app-auth || \
                            gh extension upgrade app-auth || true
                        
                        # Configure authentication
                        echo "${GITHUB_APP_PRIVATE_KEY}" > "${WORKSPACE}/app-key.pem"
                        chmod 600 "${WORKSPACE}/app-key.pem"
                        
                        gh app-auth setup \
                            --app-id "${GITHUB_APP_ID}" \
                            --key-file "${WORKSPACE}/app-key.pem" \
                            --patterns "github.com/*/*"

                        # Optional: configure Bitbucket PAT for mirrored repos
                        if [ -n "${BITBUCKET_PAT:-}" ]; then
                            gh app-auth setup \
                                --pat "${BITBUCKET_PAT}" \
                                --patterns "bitbucket.example.com/" \
                                --username "${BITBUCKET_USERNAME}" \
                                --name "Bitbucket PAT" \
                                --priority 40
                        fi
                        
                        # Configure git
                        git config --global credential.helper "!gh app-auth git-credential"
                        
                        # Verify setup
                        gh app-auth list
                    '''
                }
            }
        }
        
        stage('Checkout') {
            steps {
                script {
                    def cloneCommand = "git clone"
                    if (params.INCLUDE_SUBMODULES) {
                        cloneCommand += " --recurse-submodules"
                    }
                    cloneCommand += " ${params.REPO_URL} source"
                    
                    sh cloneCommand
                }
            }
        }
        
        stage('Build') {
            steps {
                dir('source') {
                    sh '''
                        # Your build commands
                        echo "Building project..."
                        # make build
                    '''
                }
            }
        }
        
        stage('Test') {
            steps {
                dir('source') {
                    sh '''
                        # Your test commands
                        echo "Running tests..."
                        # make test
                    '''
                }
            }
        }
    }
    
    post {
        always {
            sh '''
                # Cleanup
                rm -f "${WORKSPACE}/app-key.pem"
                gh app-auth remove --all || true
            '''
        }
        success {
            echo 'Build succeeded!'
        }
        failure {
            echo 'Build failed!'
            sh '''
                # Debug information
                gh app-auth list || true
                git config --list | grep credential || true
            '''
        }
    }
}
```

### Scripted Pipeline with Multiple Organizations

```groovy
node {
    def organizations = [
        [name: 'org1', appId: 'org1-github-app-id', keyFile: 'org1-github-app-key'],
        [name: 'org2', appId: 'org2-github-app-id', keyFile: 'org2-github-app-key']
    ]
    
    try {
        stage('Setup Multi-Org Authentication') {
            // Install extension once
            sh 'gh extension install AmadeusITGroup/gh-app-auth || true'
            
            // Configure each organization
            organizations.each { org ->
                withCredentials([
                    string(credentialsId: org.appId, variable: 'APP_ID'),
                    file(credentialsId: org.keyFile, variable: 'KEY_FILE')
                ]) {
                    sh """
                        gh app-auth setup \\
                            --app-id "\${APP_ID}" \\
                            --key-file "\${KEY_FILE}" \\
                            --patterns "github.com/${org.name}/*"
                        
                        git config --global credential."https://github.com/${org.name}".helper \\
                            "!gh app-auth git-credential"
                    """
                }
            }
            
            // Verify configuration
            sh 'gh app-auth list'
        }
        
        stage('Clone Cross-Org Repository') {
            // Clone main repo from org1 with submodules from org2
            sh '''
                git clone --recurse-submodules \\
                    https://github.com/org1/main-repo.git
                cd main-repo
                git submodule status
            '''
        }
        
        stage('Long Running Build') {
            // Simulate long-running job
            // Tokens will auto-refresh after 55 minutes
            sh '''
                cd main-repo
                echo "Starting long build process..."
                # sleep 3700  # Simulate >1 hour build
                # git operations still work after token refresh
                git submodule update --recursive
                echo "Build completed"
            '''
        }
        
    } finally {
        // Cleanup
        sh '''
            gh app-auth remove --all || true
        '''
    }
}
```

## GitLab CI

```yaml
variables:
  GIT_SUBMODULE_STRATEGY: none  # We'll handle submodules manually

before_script:
  - |
    # Install GitHub CLI
    curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | \
      dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | \
      tee /etc/apt/sources.list.d/github-cli.list > /dev/null
    apt update && apt install gh -y
    
    # Install gh-app-auth
    gh extension install AmadeusITGroup/gh-app-auth
    
    # Configure GitHub App
    echo "$GITHUB_APP_PRIVATE_KEY" > /tmp/app-key.pem
    chmod 600 /tmp/app-key.pem
    gh app-auth setup \
      --app-id "$GITHUB_APP_ID" \
      --key-file /tmp/app-key.pem \
      --patterns "github.com/$GITHUB_ORG/*"
    
    # Configure git
    git config --global credential."https://github.com/$GITHUB_ORG".helper \
      "!gh app-auth git-credential"

build:
  stage: build
  script:
    - git clone --recurse-submodules https://github.com/$GITHUB_ORG/my-repo.git
    - cd my-repo
    - make build
  
  after_script:
    - rm -f /tmp/app-key.pem
```

## Common Patterns

### Pattern 1: Conditional Submodule Checkout

```bash
# Only checkout submodules if they haven't been cloned yet
if [ ! -d ".git/modules" ]; then
    echo "Initializing submodules..."
    git submodule update --init --recursive
else
    echo "Updating existing submodules..."
    git submodule update --recursive --remote
fi
```

### Pattern 2: Selective Submodule Checkout

```bash
# Only checkout specific submodules
git submodule init
git submodule update --recursive libs/common
git submodule update --recursive libs/utils
```

### Pattern 3: Parallel Organization Setup

```bash
# Configure multiple orgs in parallel (bash)
declare -A orgs=(
    ["org1"]="APP_ID_1:KEY_FILE_1"
    ["org2"]="APP_ID_2:KEY_FILE_2"
)

for org in "${!orgs[@]}"; do
    IFS=':' read -r app_id key_file <<< "${orgs[$org]}"
    (
        gh app-auth setup \
            --app-id "$app_id" \
            --key-file "$key_file" \
            --patterns "github.com/$org/*"
        git config --global credential."https://github.com/$org".helper \
            "!gh app-auth git-credential"
    ) &
done
wait
```

### Pattern 4: Dynamic Organization Discovery

```groovy
// Jenkinsfile: Discover organizations from repository
stage('Discover Dependencies') {
    script {
        def orgs = sh(
            script: '''
                git submodule status | \
                awk '{print $2}' | \
                xargs -I {} git config -f .gitmodules submodule.{}.url | \
                sed 's|https://github.com/||' | \
                cut -d'/' -f1 | \
                sort -u
            ''',
            returnStdout: true
        ).trim().split('\n')
        
        orgs.each { org ->
            echo "Configuring access for organization: ${org}"
            // Configure GitHub App for this org
        }
    }
}
```

## Troubleshooting

### Issue: "Permission denied" when cloning

**Symptoms**:
```
fatal: could not read Username for 'https://github.com': No such device or address
```

**Solution**:
```bash
# Verify GitHub App is configured
gh app-auth list

# Test authentication
gh app-auth test --repo github.com/myorg/repo

# Verify git credential helper is configured
git config --get-all credential.helper

# Should output: !gh app-auth git-credential
```

### Issue: Submodules fail to clone

**Symptoms**:
```
fatal: clone of 'https://github.com/org2/submodule' into submodule path failed
```

**Solution**:
```bash
# Ensure GitHub App is configured for all organizations
gh app-auth list

# Configure additional organizations
gh app-auth setup \
  --app-id $ORG2_APP_ID \
  --key-file org2-key.pem \
  --patterns "github.com/org2/*"

# Configure git credential helper for the organization
git config --global credential."https://github.com/org2".helper \
  "!gh app-auth git-credential"
```

### Issue: Tokens expired during long-running job

**Symptoms**:
```
remote: Invalid username or password.
fatal: Authentication failed
```

**Solution**:
This should not happen with gh-app-auth (tokens auto-refresh), but if it does:

```bash
# Check token cache status
gh app-auth list

# Force token refresh by removing cache
rm -rf ~/.config/gh-app-auth/cache/

# Retry git operation (will generate fresh token)
git pull
```

### Issue: Multiple GitHub Apps causing conflicts

**Symptoms**:
```
Error: Multiple GitHub Apps match repository pattern
```

**Solution**:
```bash
# List configured apps and their patterns
gh app-auth list

# Remove conflicting configuration
gh app-auth remove --app-id <conflicting-app-id>

# Use more specific patterns
gh app-auth setup \
  --app-id 111111 \
  --patterns "github.com/org1/specific-repo" \
  --key-file app1.pem
```

### Debug Mode

```bash
# Enable verbose output
export GH_DEBUG=api

# Test with debugging
gh app-auth test --repo github.com/myorg/repo

# Check git credential helper execution
GIT_CURL_VERBOSE=1 GIT_TRACE=1 git clone https://github.com/myorg/repo.git
```

## Performance Optimization

### Cache Token Between Jobs

```yaml
# GitHub Actions
- name: Cache GitHub App tokens
  uses: actions/cache@v3
  with:
    path: ~/.config/gh-app-auth/cache
    key: gh-app-auth-${{ github.run_id }}
    restore-keys: gh-app-auth-
```

### Reduce API Calls

```bash
# Configure all apps before any git operations
gh app-auth setup --app-id 111 --key-file key1.pem --patterns "github.com/org1/*"
gh app-auth setup --app-id 222 --key-file key2.pem --patterns "github.com/org2/*"

# Then perform all git operations
git clone --recurse-submodules https://github.com/org1/repo.git
```

## Security Best Practices

1. **Never log private keys**: Ensure keys are not printed in CI logs
2. **Use secure credentials storage**: GitHub Secrets, Jenkins Credentials, etc.
3. **Clean up after builds**: Remove private key files in post-build steps
4. **Limit App permissions**: Only grant necessary repository access
5. **Use read-only tokens when possible**: Configure GitHub App with minimal permissions
6. **Rotate keys regularly**: Establish key rotation policy
7. **Audit App usage**: Review GitHub App activity logs regularly

## Support

For issues specific to CI/CD integration:
- Check [Troubleshooting Guide](troubleshooting.md)
- Review [Security Considerations](security.md)
- Report issues at [GitHub Issues](https://github.com/AmadeusITGroup/gh-app-auth/issues)

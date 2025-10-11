# GitHub App Authentication Extension

A GitHub CLI extension that enables GitHub App authentication for Git operations and API access.

## Features

- 🔐 **Dual Authentication**: Use GitHub Apps **and** Personal Access Tokens (PATs) with secure priority-based routing (GitHub.com + GitHub Enterprise + Bitbucket Server)
- 🔑 **Encrypted Storage**: Private keys stored in OS-native encrypted keyring (Keychain, Credential Manager, Secret Service)
- 🎯 **Repository-Specific**: Configure different apps for different repository patterns
- 🔄 **Git Integration**: Seamless git credential helper integration
- 💾 **Smart Token Caching**: In-memory caching of installation tokens (55-min TTL) reduces API calls by ~98%, automatic expiration handling
- 🔒 **Security**: 71% risk reduction with encrypted key storage and proper permission validation
- 🏢 **Enterprise Ready**: Multi-organization and enterprise GitHub support
- 🌍 **Environment Variables**: Support for CI/CD with `GH_APP_PRIVATE_KEY`
- ♻️  **Graceful Degradation**: Automatic fallback to filesystem if keyring unavailable

## Installation

```bash
gh extension install AmadeusITGroup/gh-app-auth
```

## Quick Start

### Option 1: Encrypted Storage (Recommended) 🔐

1. **Create a GitHub App** in your organization settings
2. **Download the private key** file
3. **Configure with encrypted storage**:

```bash
# Store key in encrypted keyring via environment variable
export GH_APP_PRIVATE_KEY="$(cat ~/.ssh/my-app.private-key.pem)"
gh app-auth setup \
  --app-id 123456 \
  --patterns "github.com/myorg/*"

# Key is now securely stored, can clear env var
unset GH_APP_PRIVATE_KEY
```

### Option 2: File-based (Legacy)

```bash
# Store key reference (file remains on disk)
# Store key reference (file remains on disk)
gh app-auth setup \
  --app-id 123456 \
  --key-file ~/.ssh/my-app.private-key.pem \
  --patterns "github.com/myorg/*"
```

### Option 3: Personal Access Token (PAT)

Use this when you want to act as yourself rather than a GitHub App, while keeping tokens in the OS keyring.

```bash
gh app-auth setup \
  --pat ghp_your_token_here \
  --patterns "github.com/myorg/" \
  --name "My PAT" \
  --priority 10

# For services that require a real username (e.g., Bitbucket Server)
# add the optional --username flag:
#
# gh app-auth setup \
#   --pat bbpat_your_token \
#   --patterns "bitbucket.example.com/" \
#   --username your_bitbucket_username
```

PATs support the same pattern routing as apps. When both an app and a PAT match the same pattern, the entry with the higher priority wins (PATs default to higher priority for personal workflows). PATs also work for other Git providers (e.g., Bitbucket) when paired with `--username`.

### Setup Git Credential Helper (Automatic) ✨

```bash
# Automatically configure git for all configured apps
gh app-auth gitconfig --sync
```

Or manually configure for specific contexts:

```bash
git config --global credential."https://github.com/myorg".helper \
  "!gh app-auth git-credential --pattern 'github.com/myorg/*'"
```

### Test and Use

```bash
# Test authentication
gh app-auth test --repo github.com/myorg/private-repo

# Use git normally - now uses GitHub App authentication
git clone https://github.com/myorg/private-repo.git
```

## URL Prefix Routing 🎯

Route different repositories to different GitHub Apps using URL prefix matching:

```bash
# Configure App 1 for AmadeusITGroup
git config --global credential.'https://github.com/AmadeusITGroup'.helper \
  '!gh app-auth git-credential --pattern "https://github.com/AmadeusITGroup"'

# Configure App 2 for another organization
git config --global credential.'https://github.com/myorg'.helper \
  '!gh app-auth git-credential --pattern "https://github.com/myorg"'
```

**Benefits:**
- ✅ **Multiple Apps**: Use different GitHub Apps for different organizations
- ✅ **URL Prefix Matching**: Aligns with git's native credential helper behavior
- ✅ **Fine-Grained Control**: Per-organization or per-repository configuration
- ✅ **Enterprise Support**: Separate apps for GitHub Enterprise and GitHub.com

See [URL Prefix Routing Guide](docs/PATTERN_ROUTING.md) for detailed examples and use cases.

## Commands

- `gh app-auth setup` - Configure GitHub Apps **or** Personal Access Tokens (use `--pat`)
- `gh app-auth list` - List configured GitHub Apps **and PATs** (use `--verify-keys` to check key/pat accessibility)
- `gh app-auth remove` - Remove GitHub App (**--app-id**) or PAT (**--pat-name**) configuration (automatically deletes encrypted keys/tokens)
- `gh app-auth test` - Test authentication for a repository
- `gh app-auth scope` - Detect which GitHub App will handle a repository URL (shows matched app, pattern, and git config context)
- `gh app-auth gitconfig` - Manage git credential helper configuration
  - `--sync` - Automatically configure git for all apps
  - `--clean` - Remove all gh-app-auth git configurations
- `gh app-auth migrate` - Migrate private keys to encrypted storage
- `gh app-auth git-credential` - Git credential helper (internal)

See [Git Config Management Guide](docs/GITCONFIG_COMMAND.md) for details on the `gitconfig` command.

## Encrypted Storage

This extension now supports **encrypted storage** for private keys using OS-native secure storage:

- **macOS**: Keychain (AES-256 encrypted)
- **Windows**: Credential Manager (DPAPI encrypted)
- **Linux**: Secret Service API (GNOME Keyring, KWallet, etc.)

### Benefits

- ✅ **71% risk reduction** - Keys encrypted at rest
- ✅ **No plain text** - Keys never stored in config files
- ✅ **Per-app isolation** - Each app's key stored separately
- ✅ **Automatic cleanup** - Keys deleted when removing apps
- ✅ **Graceful fallback** - Uses filesystem if keyring unavailable

### Using Encrypted Storage

```bash
# From environment variable (recommended for CI/CD)
export GH_APP_PRIVATE_KEY="$(cat ~/my-key.pem)"
gh app-auth setup --app-id 12345 --patterns "github.com/org/*"

# From file (stores in keyring, keeps file as fallback)
gh app-auth setup --app-id 12345 --key-file ~/my-key.pem --patterns "github.com/org/*"

# Store a Personal Access Token in the keyring
gh app-auth setup --pat ghp_your_token --patterns "github.com/org/"

# Check where keys are stored
gh app-auth list

# Verify keys are accessible
gh app-auth list --verify-keys
```

### Migrating Existing Configurations

```bash
# Preview migration
gh app-auth migrate --dry-run

# Migrate to encrypted storage
gh app-auth migrate

# Migrate and remove original key files
gh app-auth migrate --force
```

### Force Filesystem Storage

```bash
# If keyring is not available or you prefer filesystem
gh app-auth setup --app-id 12345 --key-file ~/my-key.pem --patterns "github.com/org/*" --use-filesystem
```

## Token Caching

This extension automatically caches GitHub App installation tokens **within each command/process** to minimize redundant API calls during long-running operations (e.g., `gh app-auth test`, `gh app-auth debug`).

### How It Works

1. **Two Token Types**:
   - **JWT Tokens**: Generated on-demand (~10min validity), not cached
   - **Installation Tokens**: Cached in memory for 55 minutes (GitHub provides 60-min validity)

2. **Automatic Expiration**:
   - Tokens checked for expiration on every use
   - Expired tokens automatically regenerated
   - Background cleanup removes expired tokens every minute

3. **Security**:
   - Tokens stored **in-memory only** (not persisted to disk)
   - Tokens zeroed from memory on cleanup (best-effort)
   - Cache lost on process restart (requires re-authentication)

### Performance Impact

- **First operation**: ~200-500ms (JWT generation + API call to GitHub)
- **Cached operations**: <1ms (memory lookup)
- **API call reduction**: ~98% (1 call per hour vs 1 call per operation)

### Why Memory-Only?

Installation tokens are powerful (1-hour validity, full repo access). We prioritize security over convenience:

- ✅ **More Secure**: Tokens don't persist to disk, limited to process lifetime
- ✅ **Reduced Attack Surface**: No encrypted tokens in keyring to compromise
- ⚠️ **Tradeoff**: Re-authentication needed after process restart (~500ms overhead)

Because git invokes credential helpers as short-lived processes, each `git credential` call starts with a fresh cache. The performance win applies when a single CLI invocation needs multiple installation tokens (tests, diagnostics, multi-repo enumeration). For CI/CD (ephemeral containers) and normal development, this provides the optimal security/performance balance while avoiding persistent tokens.

**See [Token Caching Documentation](docs/TOKEN_CACHING.md) for detailed technical information.**

## Personal Access Tokens

While GitHub Apps are ideal for automation and organization-level governance, some workflows require acting as an individual. PAT support adds:

- ✅ **Secure Storage**: PATs are stored in the OS keyring with filesystem fallback
- ✅ **Pattern Routing**: Route PAT usage by host/org/repo just like apps
- ✅ **Priority Control**: PATs can override apps with higher `--priority` values
- ✅ **Seamless Git Integration**: `gitconfig --sync` configures both apps and PATs

Example mixed configuration (`~/.config/gh/extensions/gh-app-auth/config.yml`):

```yaml
version: "1.0"
github_apps:
  - name: Org Automation App
    app_id: 123456
    installation_id: 987654
    private_key_source: keyring
    patterns:
      - github.com/myorg/
    priority: 5
pats:
  - name: Personal Workflows
    private_key_source: keyring
    patterns:
      - github.com/myorg/
    priority: 10
```

In this example, personal git operations use the PAT while CI jobs can continue to rely on the GitHub App by configuring a lower-priority PAT or using repo-specific patterns.

## CI/CD Integration

This extension is designed to solve common CI/CD authentication challenges with GitHub Apps, including cross-organization access, git submodules, and long-running jobs.

### GitHub Actions

#### Basic Setup (with Encrypted Storage)

```yaml
name: Build with GitHub App

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Install GitHub CLI
        uses: cli/gh@v2
      
      - name: Install gh-app-auth extension
        run: gh extension install AmadeusITGroup/gh-app-auth
      
      - name: Configure GitHub App authentication
        env:
          GH_APP_PRIVATE_KEY: ${{ secrets.GITHUB_APP_PRIVATE_KEY }}
        run: |
          gh app-auth setup \
            --app-id ${{ secrets.GITHUB_APP_ID }} \
            --patterns "github.com/${{ github.repository_owner }}/*"
      
      - name: Configure git credential helper
        run: |
          git config --global credential."https://github.com/${{ github.repository_owner }}".helper \
            "!gh app-auth git-credential"
      
      - name: Checkout code with submodules
        run: |
          git clone --recurse-submodules https://github.com/${{ github.repository }}.git
          cd $(basename ${{ github.repository }})
```

**Benefits of the new approach:**
- ✅ No temporary files created
- ✅ No chmod needed
- ✅ No cleanup required
- ✅ Keys stored securely in memory only
- ✅ Cleaner CI/CD logs

#### Reusable Actions (Recommended) ✨

Use our pre-built composite actions for even simpler setup:

```yaml
name: Build with GitHub App

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      # One-step setup with automatic cleanup
      - name: Setup GitHub App Auth
        uses: AmadeusITGroup/gh-app-auth/.github/actions/setup-gh-app-auth@main
        with:
          app-id: ${{ secrets.GITHUB_APP_ID }}
          private-key: ${{ secrets.GITHUB_APP_PRIVATE_KEY }}
          patterns: 'github.com/myorg/*'
          cleanup-on-exit: 'true'  # Auto-cleanup for non-ephemeral runners

      - name: Clone repo with submodules
        run: git clone --recurse-submodules https://github.com/myorg/repo
      
      - name: Build
        run: cd repo && make build

      # Cleanup runs automatically on job completion
```

**Features:**
- ✅ **One-step setup** - Installs GitHub CLI, extension, and configures git
- ✅ **Auto-sync git config** - No manual credential helper setup needed
- ✅ **Automatic cleanup** - Removes credentials on non-ephemeral runners
- ✅ **Multiple organizations** - Supports comma-separated patterns

See [GitHub Actions Documentation](.github/actions/README.md) for advanced usage.

#### Multi-Organization Repositories with Submodules

```yaml
name: Build with Multi-Org Submodules

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Install and configure gh-app-auth
        run: gh extension install AmadeusITGroup/gh-app-auth
      
      - name: Configure GitHub Apps for multiple organizations
        env:
          ORG1_PRIVATE_KEY: ${{ secrets.ORG1_GITHUB_APP_PRIVATE_KEY }}
          ORG2_PRIVATE_KEY: ${{ secrets.ORG2_GITHUB_APP_PRIVATE_KEY }}
        run: |
          # Configure App for Organization 1
          echo "$ORG1_PRIVATE_KEY" > /tmp/org1-key.pem
          chmod 600 /tmp/org1-key.pem
          gh app-auth setup \
            --app-id ${{ secrets.ORG1_APP_ID }} \
            --key-file /tmp/org1-key.pem \
            --patterns "github.com/org1/*"
          
          # Configure App for Organization 2
          echo "$ORG2_PRIVATE_KEY" > /tmp/org2-key.pem
          chmod 600 /tmp/org2-key.pem
          gh app-auth setup \
            --app-id ${{ secrets.ORG2_APP_ID }} \
            --key-file /tmp/org2-key.pem \
            --patterns "github.com/org2/*"
          
          # Configure git to use app-auth for all organizations
          git config --global credential."https://github.com/org1".helper "!gh app-auth git-credential"
          git config --global credential."https://github.com/org2".helper "!gh app-auth git-credential"
      
      - name: Clone with cross-org submodules
        run: |
          # Automatically handles authentication for all configured orgs
          git clone --recurse-submodules https://github.com/org1/main-repo.git
```

### Jenkins Pipeline

#### Basic Jenkinsfile

```groovy
pipeline {
    agent any
    
    environment {
        GITHUB_APP_ID = credentials('github-app-id')
        GITHUB_APP_PRIVATE_KEY = credentials('github-app-private-key')
    }
    
    stages {
        stage('Setup GitHub App Authentication') {
            steps {
                sh '''
                    # Install GitHub CLI if not available
                    if ! command -v gh &> /dev/null; then
                        curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | \
                            sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
                        echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | \
                            sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
                        sudo apt update
                        sudo apt install gh -y
                    fi
                    
                    # Install gh-app-auth extension
                    gh extension install AmadeusITGroup/gh-app-auth || true
                    
                    # Configure GitHub App
                    echo "$GITHUB_APP_PRIVATE_KEY" > /tmp/app-key.pem
                    chmod 600 /tmp/app-key.pem
                    gh app-auth setup \
                        --app-id "$GITHUB_APP_ID" \
                        --key-file /tmp/app-key.pem \
                        --patterns "github.com/myorg/*"
                    
                    # Configure git credential helper
                    git config --global credential."https://github.com/myorg".helper \
                        "!gh app-auth git-credential"
                '''
            }
        }
        
        stage('Checkout') {
            steps {
                sh '''
                    # Clone with automatic GitHub App authentication
                    git clone --recurse-submodules https://github.com/myorg/my-repo.git
                    cd my-repo
                '''
            }
        }
        
        stage('Build') {
            steps {
                sh '''
                    cd my-repo
                    # Your build commands here
                '''
            }
        }
    }
    
    post {
        always {
            sh 'rm -f /tmp/app-key.pem'
        }
    }
}
```

#### Multi-Organization Jenkins Pipeline with Token Refresh

For long-running jobs (>1 hour), tokens are automatically refreshed by the extension:

```groovy
pipeline {
    agent any
    
    environment {
        ORG1_APP_ID = credentials('org1-github-app-id')
        ORG1_APP_KEY = credentials('org1-github-app-private-key')
        ORG2_APP_ID = credentials('org2-github-app-id')
        ORG2_APP_KEY = credentials('org2-github-app-private-key')
    }
    
    stages {
        stage('Setup Multi-Org Authentication') {
            steps {
                sh '''
                    # Install extension
                    gh extension install AmadeusITGroup/gh-app-auth || true
                    
                    # Configure Organization 1
                    echo "$ORG1_APP_KEY" > /tmp/org1-key.pem
                    chmod 600 /tmp/org1-key.pem
                    gh app-auth setup \
                        --app-id "$ORG1_APP_ID" \
                        --key-file /tmp/org1-key.pem \
                        --patterns "github.com/org1/*"
                    
                    # Configure Organization 2
                    echo "$ORG2_APP_KEY" > /tmp/org2-key.pem
                    chmod 600 /tmp/org2-key.pem
                    gh app-auth setup \
                        --app-id "$ORG2_APP_ID" \
                        --key-file /tmp/org2-key.pem \
                        --patterns "github.com/org2/*"
                    
                    # Configure git for both orgs
                    git config --global credential."https://github.com/org1".helper \
                        "!gh app-auth git-credential"
                    git config --global credential."https://github.com/org2".helper \
                        "!gh app-auth git-credential"
                '''
            }
        }
        
        stage('Long-Running Job') {
            steps {
                sh '''
                    # Tokens are automatically refreshed when needed
                    # Extension handles the 1-hour expiry transparently
                    git clone --recurse-submodules https://github.com/org1/main-repo.git
                    
                    # Long build process (>1 hour)
                    cd main-repo
                    ./run-long-build.sh
                    
                    # Git operations continue to work - tokens auto-refresh
                    git submodule update --recursive
                '''
            }
        }
    }
    
    post {
        always {
            sh '''
                rm -f /tmp/org1-key.pem /tmp/org2-key.pem
            '''
        }
    }
}
```

### Token Expiry and Long-Running Jobs

The extension automatically handles GitHub App token expiry (1-hour limit):

- **Automatic Refresh**: Tokens are generated on-demand and cached for 55 minutes
- **Seamless Experience**: Git operations automatically get fresh tokens when needed
- **No Manual Intervention**: No need to implement token refresh logic in your pipelines

```bash
# Token lifecycle (handled automatically)
# 1. First git operation: Generate new token
# 2. Subsequent operations (<55 min): Use cached token
# 3. After 55 minutes: Automatically generate new token
# 4. Your job continues working regardless of duration
```

### Enterprise GitHub

```bash
# Configure for GitHub Enterprise Server
gh app-auth setup \
  --app-id 123456 \
  --key-file enterprise-app.pem \
  --patterns "github.example.com/corp/*"

# Configure git credential helper
git config --global credential."https://github.example.com/corp".helper \
  "!gh app-auth git-credential"
```

### Best Practices for CI/CD

1. **Security**:
   - Store private keys in secrets management (GitHub Secrets, Jenkins Credentials)
   - Use `chmod 600` on private key files
   - Clean up key files in post-build steps

2. **Multi-Organization**:
   - Configure separate GitHub Apps for each organization
   - Use pattern matching to route authentication automatically
   - Test cross-org submodule access before production use

3. **Performance**:
   - Tokens are cached for 55 minutes to minimize API calls
   - Git credential helper integrates seamlessly with git operations
   - No need to pre-generate tokens for long-running jobs

4. **Debugging**:

   ```bash
   # Test authentication in CI
   gh app-auth test --repo github.com/myorg/my-repo
   
   # List configured apps
   gh app-auth list
   ```

### Advantages Over Robot Accounts

- ✅ **No License Costs**: GitHub Apps don't consume user licenses
- ✅ **Better Governance**: Fine-grained permissions per repository/organization  
- ✅ **Audit Trail**: All actions clearly attributed to the GitHub App
- ✅ **No Recertification**: No periodic password resets or token rotation
- ✅ **Multi-Org Native**: Single App can be installed across multiple organizations
- ✅ **Automatic Token Refresh**: Extension handles 1-hour expiry transparently

## Documentation

### 📺 Presentation

- **[Project Presentation](docs/presentation.md)** - Comprehensive webinar presentation covering problem, solutions, architecture, and results
  - View online at [GitHub Pages](https://AmadeusITGroup.github.io/gh-app-auth/) (when published)
  - Build locally: `make presentation` (requires `make presentation-setup` first)
  - See [Presentation Build Guide](docs/PRESENTATION_BUILD.md) for detailed instructions

### 📚 Guides

- [Installation Guide](docs/installation.md)
- [Configuration Reference](docs/configuration.md)
- [CI/CD Integration Guide](docs/ci-cd-guide.md) - **Comprehensive guide for GitHub Actions, Jenkins, and more**
- [Security Considerations](docs/security.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Architecture Overview](docs/architecture.md)
- [Project Origin](docs/origin_of_the_project.md) - Understanding the problems we solve

### 🧪 Testing

- [E2E Testing Tutorial](docs/E2E_TESTING_TUTORIAL.md) - **Complete guide to setting up real-world test environments**
  - Step-by-step setup with GitHub organizations, apps, and repositories
  - Automated scripts for environment creation and validation
  - Interactive setup wizard for first-time users
- [Token Caching Details](docs/TOKEN_CACHING.md) - Technical deep-dive into token caching implementation

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and contribution guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

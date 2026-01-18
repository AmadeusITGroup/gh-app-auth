# Encrypted Storage Architecture Analysis & Proposal

## Part 1: GitHub CLI (gh-cli) Encrypted Storage Architecture

### Overview

The GitHub CLI implements a **layered credential storage system** that prioritizes security while maintaining backward compatibility and graceful degradation. The architecture uses OS-native encrypted credential stores (keyring/keychain) as the primary storage mechanism with automatic fallback to filesystem-based storage.

### Core Components

#### 1. **Keyring Abstraction Layer** (`internal/keyring/keyring.go`)

```mermaid
graph TB
    App["gh-cli Application Layer"]
    Wrapper["Keyring Wrapper (timeout logic)<br/>- Set(service, user, secret)<br/>- Get(service, user)<br/>- Delete(service, user)"]
    GoKeyring["zalando/go-keyring (v0.2.6)<br/>Cross-platform keyring interface"]
    
    macOS["macOS<br/>Keychain"]
    Linux["Linux<br/>Secret Service"]
    Windows["Windows<br/>Credential Manager"]
    FreeBSD["FreeBSD<br/>keyring"]
    
    App --> Wrapper
    Wrapper --> GoKeyring
    GoKeyring --> macOS
    GoKeyring --> Linux
    GoKeyring --> Windows
    GoKeyring --> FreeBSD
    
    style App fill:#e3f2fd,stroke:#1976d2,stroke-width:2px
    style Wrapper fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    style GoKeyring fill:#fff3e0,stroke:#f57c00,stroke-width:2px
```

**Key Features:**

- **Timeout Protection**: 3-second timeout on all keyring operations to prevent hangs
- **Error Handling**: Wraps `zalando/go-keyring` errors with timeout-specific error types
- **Service Naming**: Uses format `gh:{hostname}` for service identification
- **User-based Keys**: Stores credentials per-user per-host for multi-account support

#### 2. **Authentication Configuration** (`internal/config/config.go`)

```go
type AuthConfig struct {
    cfg                 *ghConfig.Config
    defaultHostOverride func() (string, string)
    hostsOverride       func() []string
    tokenOverride       func(string) (string, string)
}
```

**Token Resolution Strategy (Priority Order):**

```mermaid
graph TB
    Start([Token Resolution])
    Env["1. Environment Variables<br/>(highest priority)<br/>• GH_TOKEN<br/>• GITHUB_TOKEN"]
    Override["2. Token Override<br/>(testing)"]
    Keyring["3. Encrypted Keyring Storage"]
    KeyringUser["Try: keyring.Get(service, username)<br/>(Per-user token)"]
    KeyringLegacy["Fallback: keyring.Get(service, '')<br/>(Legacy unkeyed token)"]
    Config["4. Plain Text Config File<br/>(lowest priority)<br/>~/.config/gh/hosts.yml"]
    
    Start --> Env
    Start --> Override
    Start --> Keyring
    Keyring --> KeyringUser
    Keyring --> KeyringLegacy
    Start --> Config
    
    style Start fill:#e3f2fd,stroke:#1976d2,stroke-width:2px
    style Env fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px
    style Override fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    style Keyring fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    style Config fill:#ffebee,stroke:#c62828,stroke-width:2px
```

**Authentication Flow:**

```mermaid
flowchart TD
    Start([ActiveToken hostname])
    EnvCheck{Env Var Set?}
    ReturnEnv[Return env token]
    GetUser[Get Active User]
    KeyringUser{TokenFromKeyringForUser<br/>hostname, user}
    ReturnKeyring[Return keyring token]
    KeyringLegacy{TokenFromKeyring<br/>hostname, ''}
    ReturnLegacy[Return legacy keyring token]
    ReturnEmpty[Return '']
    
    Start --> EnvCheck
    EnvCheck -->|Yes| ReturnEnv
    EnvCheck -->|No| GetUser
    GetUser --> KeyringUser
    KeyringUser -->|Found| ReturnKeyring
    KeyringUser -->|Not Found| KeyringLegacy
    KeyringLegacy -->|Found| ReturnLegacy
    KeyringLegacy -->|Not Found| ReturnEmpty
    
    style Start fill:#e3f2fd,stroke:#1976d2,stroke-width:2px
    style ReturnEnv fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px
    style ReturnKeyring fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px
    style ReturnLegacy fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    style ReturnEmpty fill:#ffebee,stroke:#c62828,stroke-width:2px
```

#### 3. **Login Flow with Graceful Degradation**

```go
func (c *AuthConfig) Login(hostname, username, token, gitProtocol string, 
                           secureStorage bool) (insecureStorageUsed bool, error) {
    
    var setErr error
    if secureStorage {
        // Attempt encrypted storage
        setErr = keyring.Set(keyringServiceName(hostname), username, token)
        if setErr == nil {
            // Success! Clean up any plain text tokens
            _ = c.cfg.Remove([]string{hostsKey, hostname, usersKey, username, oauthTokenKey})
        }
    }
    
    insecureStorageUsed := false
    if !secureStorage || setErr != nil {
        // Fallback to plain text storage
        c.cfg.Set([]string{hostsKey, hostname, usersKey, username, oauthTokenKey}, token)
        insecureStorageUsed = true
    }
    
    // Set user as active
    return insecureStorageUsed, c.activateUser(hostname, username)
}
```

**Degradation Strategy:**

1. **Attempt**: Store in OS keyring/keychain
2. **On Success**: Remove plain text tokens, return `insecureStorageUsed=false`
3. **On Failure**: Silently fall back to config file, return `insecureStorageUsed=true`
4. **User Notification**: Caller can inform user about storage method used

#### 4. **Multi-Account Support**

The system supports multiple users per hostname:

```yaml
# ~/.config/gh/hosts.yml
hosts:
  github.com:
    user: williammartin          # Active user
    git_protocol: https
    users:
      williammartin:              # User 1
        # Token in keyring: gh:github.com[williammartin]
      monalisa:                   # User 2
        oauth_token: xyz          # Or in plain text if keyring failed
```

**User Switching:**

- Active token stored at: `gh:github.com[""]` (empty user key)
- User-specific tokens: `gh:github.com["username"]`
- Switching copies user token to active slot

#### 5. **Migration Strategy** (`internal/config/migration/multi_account.go`)

The CLI includes a migration system to upgrade old configurations:

```
Pre-Migration (v0):
  github.com:
    user: williammartin
    oauth_token: abc123        # Plain text or keyring[""]

Post-Migration (v1):
  github.com:
    user: williammartin
    users:
      williammartin:
        oauth_token: abc123    # Or keyring["williammartin"]
```

**Migration Process:**

1. Detect config version (or absence thereof)
2. For each host, fetch token (keyring or config)
3. Fetch username (config or API call)
4. Migrate token to user-keyed storage
5. Update config structure
6. Set version to "1"

### Security Features

#### 1. **Platform-Specific Encryption**

| Platform | Storage Backend | Encryption |
|----------|-----------------|------------|
| macOS | Keychain | AES-256, protected by user's login password |
| Windows | Credential Manager | DPAPI (Data Protection API) |
| Linux | Secret Service API | Various (GNOME Keyring, KWallet, etc.) |
| FreeBSD | keyring | Platform-dependent |

#### 2. **Timeout Protection**

All keyring operations have a 3-second timeout to prevent:

- Hung processes waiting for keyring unlock
- UI freezes on systems with broken keyring
- Indefinite waits on remote/SSH sessions

#### 3. **Error Recovery**

```go
type TimeoutError struct {
    message string
}

// Specific error types allow graceful handling:
// - keyring.ErrNotFound → Try fallback locations
// - TimeoutError → Fall back to plain text
// - Other errors → Fall back to plain text
```

### Design Principles

1. **Security by Default**: Encrypted storage attempted first
2. **Graceful Degradation**: Always works, even if keyring unavailable
3. **Backward Compatibility**: Old configs continue to work
4. **Forward Compatibility**: Old CLI versions can read new configs
5. **Multi-Account Ready**: Per-user credential isolation
6. **Timeout Protection**: Never hang on keyring operations
7. **Zero Data Loss**: Migration preserves all credentials

---

## Part 2: Proposal for gh-app-auth Encrypted Storage

### Current Architecture Limitations

**Current State:**

```mermaid
graph TB
    subgraph Current["Current State"]
        Config["Config: ~/.config/gh/extensions/gh-app-auth/config.yml"]
        Metadata["GitHub App metadata<br/>(app_id, installation_id, patterns)"]
        KeyPath["Private key PATH<br/>(unencrypted reference)"]
        PrivateKey["Private Key: User-specified filesystem location"]
        PEM["PEM file with RSA private key<br/>(600 permissions)"]
        
        Config --> Metadata
        Config --> KeyPath
        Config -.-> PrivateKey
        PrivateKey --> PEM
    end
    
    style Config fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    style PrivateKey fill:#ffebee,stroke:#c62828,stroke-width:2px
    style PEM fill:#ffebee,stroke:#c62828,stroke-width:2px
```

**Problems:**

1. **Private keys stored unencrypted** on filesystem
2. **No protection** if filesystem permissions compromised
3. **Keys in plain text** in config file paths
4. **No multi-environment support** (dev/staging/prod keys)
5. **Manual key rotation** requires config file updates

### Proposed Architecture

#### Layer 1: Encrypted Storage Backend

```mermaid
graph TB
    Extension["gh-app-auth Extension"]
    Manager["Secure Storage Manager"]
    API["Store(appName, secretType, value) → error<br/>Get(appName, secretType) → (value, error)<br/>Delete(appName, secretType) → error<br/>IsAvailable() → bool"]
    Keyring["Keyring Store<br/>(Encrypted)"]
    Filesystem["Filesystem Store<br/>(Fallback)"]
    
    Extension --> Manager
    Manager --> API
    Manager --> Keyring
    Manager --> Filesystem
    
    style Extension fill:#e3f2fd,stroke:#1976d2,stroke-width:2px
    style Manager fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    style API fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    style Keyring fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px
    style Filesystem fill:#ffebee,stroke:#c62828,stroke-width:2px
```

#### Layer 2: Secret Types

```go
type SecretType string

const (
    SecretTypePrivateKey     SecretType = "private_key"
    SecretTypeAccessToken    SecretType = "access_token"
    SecretTypeInstallToken   SecretType = "installation_token"
)

type SecretStore interface {
    // Store a secret with app-specific namespacing
    Store(appName string, secretType SecretType, value string) error
    
    // Retrieve a secret
    Get(appName string, secretType SecretType) (string, error)
    
    // Delete a secret
    Delete(appName string, secretType SecretType) error
    
    // Check if encrypted storage is available
    IsAvailable() bool
    
    // Get storage backend type
    Backend() StorageBackend
}

type StorageBackend string

const (
    StorageBackendKeyring    StorageBackend = "keyring"
    StorageBackendFilesystem StorageBackend = "filesystem"
)
```

#### Layer 3: Configuration Changes

**New Config Structure:**

```yaml
version: "1"  # Enable migrations
github_apps:
  - name: my-org-app
    app_id: 12345
    installation_id: 67890
    
    # NEW: Private key can be inline (from env var) or reference
    private_key_source: "keyring"  # or "filesystem" or "inline"
    
    # If filesystem: path to key file
    private_key_path: ~/.ssh/github-app.pem
    
    # If inline: key stored in keyring (this field removed after storage)
    # private_key: "-----BEGIN RSA PRIVATE KEY-----\n..."
    
    patterns:
      - "github.com/myorg/*"
    priority: 100
```

**Keyring Service Names:**

```
Service: gh-app-auth:{app_name}
Users:
  - "private_key"       → RSA private key PEM
  - "access_token"      → Cached access token
  - "install_token"     → Cached installation token
```

### Implementation Plan

#### Phase 1: Core Infrastructure

**File: `pkg/secrets/secrets.go`**

```go
package secrets

import (
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "time"
    
    "github.com/zalando/go-keyring"
)

var (
    ErrNotFound          = errors.New("secret not found")
    ErrTimeout           = errors.New("keyring operation timeout")
    ErrStorageUnavailable = errors.New("encrypted storage unavailable")
)

type Manager struct {
    keyringTimeout time.Duration
    fallbackDir    string
}

func NewManager(fallbackDir string) *Manager {
    return &Manager{
        keyringTimeout: 3 * time.Second,
        fallbackDir:    fallbackDir,
    }
}

// Store attempts encrypted storage first, falls back to filesystem
func (m *Manager) Store(appName string, secretType SecretType, value string) (StorageBackend, error) {
    // Try encrypted storage first
    if m.IsAvailable() {
        if err := m.storeInKeyring(appName, secretType, value); err == nil {
            // Success! Clean up any filesystem version
            _ = m.deleteFromFilesystem(appName, secretType)
            return StorageBackendKeyring, nil
        }
    }
    
    // Fallback to filesystem
    if err := m.storeInFilesystem(appName, secretType, value); err != nil {
        return "", fmt.Errorf("failed to store in both keyring and filesystem: %w", err)
    }
    
    return StorageBackendFilesystem, nil
}

// Get tries keyring first, then filesystem
func (m *Manager) Get(appName string, secretType SecretType) (string, StorageBackend, error) {
    // Try keyring first
    if value, err := m.getFromKeyring(appName, secretType); err == nil {
        return value, StorageBackendKeyring, nil
    }
    
    // Try filesystem
    if value, err := m.getFromFilesystem(appName, secretType); err == nil {
        return value, StorageBackendFilesystem, nil
    }
    
    return "", "", ErrNotFound
}

func (m *Manager) storeInKeyring(appName string, secretType SecretType, value string) error {
    service := m.keyringService(appName)
    user := string(secretType)
    
    ch := make(chan error, 1)
    go func() {
        defer close(ch)
        ch <- keyring.Set(service, user, value)
    }()
    
    select {
    case err := <-ch:
        return err
    case <-time.After(m.keyringTimeout):
        return ErrTimeout
    }
}

func (m *Manager) getFromKeyring(appName string, secretType SecretType) (string, error) {
    service := m.keyringService(appName)
    user := string(secretType)
    
    ch := make(chan struct {
        val string
        err error
    }, 1)
    
    go func() {
        defer close(ch)
        val, err := keyring.Get(service, user)
        ch <- struct {
            val string
            err error
        }{val, err}
    }()
    
    select {
    case res := <-ch:
        if errors.Is(res.err, keyring.ErrNotFound) {
            return "", ErrNotFound
        }
        return res.val, res.err
    case <-time.After(m.keyringTimeout):
        return "", ErrTimeout
    }
}

func (m *Manager) storeInFilesystem(appName string, secretType SecretType, value string) error {
    path := m.filesystemPath(appName, secretType)
    
    // Ensure directory exists with secure permissions
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0700); err != nil {
        return fmt.Errorf("failed to create secrets directory: %w", err)
    }
    
    // Write with secure permissions (owner read-only)
    if err := os.WriteFile(path, []byte(value), 0400); err != nil {
        return fmt.Errorf("failed to write secret file: %w", err)
    }
    
    return nil
}

func (m *Manager) getFromFilesystem(appName string, secretType SecretType) (string, error) {
    path := m.filesystemPath(appName, secretType)
    
    data, err := os.ReadFile(path)
    if os.IsNotExist(err) {
        return "", ErrNotFound
    }
    if err != nil {
        return "", fmt.Errorf("failed to read secret file: %w", err)
    }
    
    return string(data), nil
}

func (m *Manager) deleteFromFilesystem(appName string, secretType SecretType) error {
    path := m.filesystemPath(appName, secretType)
    return os.Remove(path)
}

func (m *Manager) IsAvailable() bool {
    // Test keyring availability with a quick operation
    testService := "gh-app-auth:availability-test"
    testUser := "test"
    testValue := "test"
    
    ch := make(chan bool, 1)
    go func() {
        defer close(ch)
        if err := keyring.Set(testService, testUser, testValue); err != nil {
            ch <- false
            return
        }
        _ = keyring.Delete(testService, testUser)
        ch <- true
    }()
    
    select {
    case available := <-ch:
        return available
    case <-time.After(m.keyringTimeout):
        return false
    }
}

func (m *Manager) keyringService(appName string) string {
    return fmt.Sprintf("gh-app-auth:%s", appName)
}

func (m *Manager) filesystemPath(appName string, secretType SecretType) string {
    filename := fmt.Sprintf("%s.%s", appName, secretType)
    return filepath.Join(m.fallbackDir, "secrets", filename)
}
```

#### Phase 2: Config Integration

**File: `pkg/config/secrets.go`**

```go
package config

import (
    "fmt"
    "os"
    
    "github.com/wherka-ama/gh-app-auth/pkg/secrets"
)

type PrivateKeySource string

const (
    PrivateKeySourceKeyring    PrivateKeySource = "keyring"
    PrivateKeySourceFilesystem PrivateKeySource = "filesystem"
    PrivateKeySourceInline     PrivateKeySource = "inline"
)

// Enhanced GitHubApp with secret management
type GitHubApp struct {
    Name                string              `yaml:"name"`
    AppID               int64               `yaml:"app_id"`
    InstallationID      int64               `yaml:"installation_id"`
    PrivateKeySource    PrivateKeySource    `yaml:"private_key_source"`
    PrivateKeyPath      string              `yaml:"private_key_path,omitempty"`
    Patterns            []string            `yaml:"patterns"`
    Priority            int                 `yaml:"priority"`
}

// GetPrivateKey retrieves the private key from the appropriate source
func (app *GitHubApp) GetPrivateKey(secretMgr *secrets.Manager) (string, error) {
    switch app.PrivateKeySource {
    case PrivateKeySourceKeyring:
        key, _, err := secretMgr.Get(app.Name, secrets.SecretTypePrivateKey)
        if err == nil {
            return key, nil
        }
        // If keyring fails, try filesystem as fallback
        fallthrough
        
    case PrivateKeySourceFilesystem:
        if app.PrivateKeyPath == "" {
            return "", fmt.Errorf("private_key_path not set for filesystem source")
        }
        
        expandedPath, err := expandPath(app.PrivateKeyPath)
        if err != nil {
            return "", fmt.Errorf("failed to expand path: %w", err)
        }
        
        keyData, err := os.ReadFile(expandedPath)
        if err != nil {
            return "", fmt.Errorf("failed to read private key: %w", err)
        }
        
        return string(keyData), nil
        
    case PrivateKeySourceInline:
        // Inline keys should have been migrated to keyring during setup
        return "", fmt.Errorf("inline private keys should be migrated to secure storage")
        
    default:
        return "", fmt.Errorf("unknown private key source: %s", app.PrivateKeySource)
    }
}

// SetPrivateKey stores the private key securely
func (app *GitHubApp) SetPrivateKey(secretMgr *secrets.Manager, privateKey string) error {
    backend, err := secretMgr.Store(app.Name, secrets.SecretTypePrivateKey, privateKey)
    if err != nil {
        return fmt.Errorf("failed to store private key: %w", err)
    }
    
    if backend == secrets.StorageBackendKeyring {
        app.PrivateKeySource = PrivateKeySourceKeyring
        app.PrivateKeyPath = "" // Clear path since key is in keyring
    } else {
        app.PrivateKeySource = PrivateKeySourceFilesystem
        // Path already set by filesystem backend
    }
    
    return nil
}
```

#### Phase 3: CLI Integration

**File: `cmd/setup.go` (enhanced)**

```go
func runSetup(cmd *cobra.Command, args []string) error {
    // ... existing code ...
    
    // Get private key from environment variable or prompt
    var privateKey string
    if envKey := os.Getenv("GH_APP_PRIVATE_KEY"); envKey != "" {
        privateKey = envKey
        fmt.Fprintf(iostreams.Out, "Using private key from GH_APP_PRIVATE_KEY\n")
    } else if privateKeyPath != "" {
        keyData, err := os.ReadFile(privateKeyPath)
        if err != nil {
            return fmt.Errorf("failed to read private key: %w", err)
        }
        privateKey = string(keyData)
    } else {
        return fmt.Errorf("private key required (use --private-key-path or GH_APP_PRIVATE_KEY)")
    }
    
    // Create secret manager
    configDir := filepath.Dir(getDefaultConfigPath())
    secretMgr := secrets.NewManager(configDir)
    
    // Store private key
    backend, err := app.SetPrivateKey(secretMgr, privateKey)
    if err != nil {
        return fmt.Errorf("failed to store private key: %w", err)
    }
    
    // Inform user about storage method
    if backend == secrets.StorageBackendKeyring {
        fmt.Fprintf(iostreams.Out, "✓ Private key stored securely in system keyring\n")
    } else {
        fmt.Fprintf(iostreams.Out, "⚠ Private key stored in filesystem (keyring unavailable)\n")
        fmt.Fprintf(iostreams.Out, "  Location: %s\n", configDir+"/secrets")
    }
    
    // Save config (without private key in YAML)
    cfg.AddOrUpdateApp(&app)
    if err := cfg.Save(); err != nil {
        return fmt.Errorf("failed to save config: %w", err)
    }
    
    return nil
}
```

### Migration Strategy

**File: `pkg/config/migration.go`**

```go
package config

import (
    "fmt"
    
    "github.com/wherka-ama/gh-app-auth/pkg/secrets"
)

const CurrentConfigVersion = "1"

// Migrate upgrades old configs to new format
func Migrate(cfg *Config, secretMgr *secrets.Manager) error {
    // Check if migration needed
    if cfg.Version == CurrentConfigVersion {
        return nil
    }
    
    // Migrate from v0 (no version) to v1
    if cfg.Version == "" {
        if err := migrateV0ToV1(cfg, secretMgr); err != nil {
            return fmt.Errorf("failed to migrate v0 to v1: %w", err)
        }
        cfg.Version = "1"
    }
    
    return cfg.Save()
}

func migrateV0ToV1(cfg *Config, secretMgr *secrets.Manager) error {
    for i := range cfg.GitHubApps {
        app := &cfg.GitHubApps[i]
        
        // If app has a private_key_path, keep it as filesystem source
        if app.PrivateKeyPath != "" {
            app.PrivateKeySource = PrivateKeySourceFilesystem
            continue
        }
        
        // If app has no private key info, it's invalid
        if app.PrivateKeyPath == "" {
            return fmt.Errorf("app %s has no private key configured", app.Name)
        }
    }
    
    return nil
}
```

### Usage Examples

#### Example 1: Setup with Environment Variable

```bash
# Export private key (one-time)
export GH_APP_PRIVATE_KEY="$(cat ~/.ssh/github-app.pem)"

# Setup app (key stored in keyring)
gh app-auth setup \
  --name my-org-app \
  --app-id 12345 \
  --installation-id 67890 \
  --pattern "github.com/myorg/*"

# Private key now in keyring, can unset env var
unset GH_APP_PRIVATE_KEY
```

#### Example 2: Setup with File Path (Fallback)

```bash
# Setup with file path (keyring unavailable or disabled)
gh app-auth setup \
  --name my-org-app \
  --app-id 12345 \
  --installation-id 67890 \
  --private-key-path ~/.ssh/github-app.pem \
  --pattern "github.com/myorg/*"

# File remains on disk, config stores reference
```

#### Example 3: Migration

```bash
# Migrate existing config to use keyring
gh app-auth migrate

# Or force filesystem storage
gh app-auth migrate --storage filesystem
```

### Security Benefits

1. **Private Keys Encrypted at Rest**
   - macOS: Keychain with AES-256
   - Windows: DPAPI encryption
   - Linux: Secret Service (GNOME Keyring, etc.)

2. **No Plain Text Keys in Config Files**
   - Config only stores metadata
   - Keys never written to YAML

3. **Environment Variable Support**
   - Keys can be injected from CI/CD
   - No filesystem access needed

4. **Graceful Degradation**
   - Works even if keyring unavailable
   - User informed of storage method

5. **Per-App Isolation**
   - Each app's keys stored separately
   - Multi-organization support

### Backward Compatibility

- **Old configs continue to work**: `private_key_path` still supported
- **Opt-in migration**: Users can choose when to migrate
- **No breaking changes**: Existing workflows unaffected
- **Progressive enhancement**: New installs use keyring by default

### Testing Strategy

```go
// Mock keyring for testing
func TestWithMockKeyring(t *testing.T) {
    keyring.MockInit()
    defer keyring.MockInitWithError(nil)
    
    // Test encrypted storage
    mgr := secrets.NewManager("/tmp/test")
    backend, err := mgr.Store("test-app", secrets.SecretTypePrivateKey, "test-key")
    require.NoError(t, err)
    require.Equal(t, secrets.StorageBackendKeyring, backend)
}

// Test fallback behavior
func TestKeyringUnavailable(t *testing.T) {
    keyring.MockInitWithError(errors.New("keyring unavailable"))
    defer keyring.MockInitWithError(nil)
    
    // Should fall back to filesystem
    mgr := secrets.NewManager("/tmp/test")
    backend, err := mgr.Store("test-app", secrets.SecretTypePrivateKey, "test-key")
    require.NoError(t, err)
    require.Equal(t, secrets.StorageBackendFilesystem, backend)
}
```

### Performance Considerations

1. **Caching**: Token cache remains in memory during session
2. **Timeouts**: 3-second limit prevents hangs
3. **Lazy Loading**: Keys only loaded when needed
4. **No Performance Impact**: Fallback to filesystem maintains current performance

### Future Enhancements

1. **Key Rotation**: Support for rotating keys without downtime
2. **Multiple Keys**: Primary and backup keys per app
3. **Audit Logging**: Track key access and usage
4. **Remote Secrets**: Integration with HashiCorp Vault, AWS Secrets Manager
5. **Hardware Security**: Support for HSM/TPM storage

---

## Summary

This proposal brings **enterprise-grade security** to gh-app-auth while maintaining **100% backward compatibility** and **graceful degradation**. The implementation follows gh-cli's proven architecture patterns and provides a smooth migration path for existing users.

**Key Wins:**

- ✅ Private keys encrypted using OS-native secure storage
- ✅ Zero-touch experience for users with working keyrings
- ✅ Automatic fallback maintains existing functionality
- ✅ Environment variable support for CI/CD pipelines
- ✅ No breaking changes to existing workflows
- ✅ Progressive enhancement: new installs benefit immediately

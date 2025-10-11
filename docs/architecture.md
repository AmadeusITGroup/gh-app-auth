# Architecture Overview

## System Design

The `gh-app-auth` extension implements GitHub App authentication for Git operations through a modular architecture:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Git Client    │───▶│ gh-app-auth      │───▶│ GitHub API      │
│                 │    │ (Credential      │    │                 │
│ git clone/push  │    │  Helper)         │    │ Installation    │
└─────────────────┘    └──────────────────┘    │ Access Tokens   │
                                               └─────────────────┘
```

## Core Components

### 1. Command Layer (`cmd/`)
- **CLI Interface**: Cobra-based command structure
- **User Commands**: setup, list, remove, test
- **Git Integration**: credential helper implementation

### 2. Authentication Layer (`pkg/auth/`)
- **JWT Generation**: GitHub App JWT token creation
- **Token Exchange**: JWT → Installation access token
- **Credential Provider**: Git credential helper interface

### 3. Configuration Layer (`pkg/config/`)
- **App Management**: GitHub App configurations
- **Pattern Matching**: Repository-specific app selection
- **Persistence**: YAML/JSON configuration storage

### 4. Security Layer
- **Key Validation**: Private key file permission checks
- **Token Security**: Secure token caching and cleanup
- **Path Validation**: Prevention of path traversal attacks

## Data Flow

### Authentication Flow
1. **Git Request**: Git requests credentials for repository
2. **Pattern Match**: Extension matches repository to configured GitHub App
3. **JWT Generation**: Creates signed JWT using App's private key
4. **Token Exchange**: Exchanges JWT for installation access token
5. **Credential Response**: Returns token to Git client

### Configuration Flow
1. **App Setup**: User configures GitHub App with patterns
2. **Validation**: Extension validates configuration and permissions  
3. **Storage**: Configuration saved to user's config directory
4. **Runtime**: Configuration loaded during authentication

## Security Architecture

### Private Key Security
- File permission validation (600/400 only)
- No key data stored in memory longer than necessary
- Secure path handling with traversal protection

### Token Security
- Installation tokens cached with expiration (55 minutes)
- Memory cleanup after token use
- No persistent token storage

### Configuration Security
- Config files stored with restricted permissions (600)
- Input validation for all configuration values
- Safe path expansion for user directories

## Integration Points

### GitHub CLI Integration
- Uses `go-gh` library for GitHub API access
- Follows GitHub CLI extension conventions
- Compatible with GitHub CLI configuration system

### Git Integration
- Implements Git credential helper protocol
- Supports repository-specific configuration
- Seamless integration with existing Git workflows

### GitHub API Integration
- GitHub App JWT authentication
- Installation access token management
- Repository installation discovery

## Performance Considerations

### Token Caching
- JWT tokens cached to avoid regeneration
- Installation tokens cached with proper expiration
- Cache invalidation on configuration changes

### Pattern Matching
- Efficient regex-based pattern matching
- Priority-based app selection
- Minimal overhead for credential requests

### Error Handling
- Graceful fallback to other credential helpers
- Comprehensive error reporting with suggestions
- Silent failures for non-matching repositories

## Extensibility

### Plugin Architecture
- Modular package design allows for extensions
- Interface-based authentication providers
- Configurable pattern matching strategies

### Configuration Extension
- YAML/JSON configuration format
- Environment variable overrides
- Multiple GitHub instance support

## Deployment Architecture

### Local Installation
```
~/.config/gh/extensions/gh-app-auth/
├── config.yml              # User configuration
└── gh-app-auth             # Extension binary
```

### Git Configuration
```
[credential "https://github.com/myorg"]
    helper = app-auth git-credential
    useHttpPath = true
```

## Monitoring and Observability

### Logging
- Debug logging support via --debug flag
- Error logging with context information
- No logging of sensitive data (keys, tokens)

### Metrics
- Authentication success/failure tracking
- Token cache hit/miss ratios
- Configuration validation metrics

### Troubleshooting
- Built-in test command for authentication validation
- Detailed error messages with remediation suggestions
- Verbose mode for debugging authentication flows

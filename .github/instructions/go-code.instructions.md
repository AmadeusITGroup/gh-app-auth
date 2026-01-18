---
applyTo: "**/*.go"
---

# Go Code Guidelines for gh-app-auth

## Code Style

- Use `gofmt` and `goimports` formatting
- Error strings should not be capitalized (staticcheck ST1005)
- Prefer explicit returns over named return values
- Use `context.Context` for cancellation and timeouts
- Wrap errors with `fmt.Errorf("context: %w", err)` for stack traces

## Imports

Always organize imports in three groups:

1. Standard library
2. External packages
3. Internal packages (github.com/AmadeusITGroup/gh-app-auth/...)

```go
import (
    "context"
    "fmt"

    "github.com/spf13/cobra"

    "github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)
```

## Error Handling

- Return errors rather than panicking
- Provide actionable error messages
- Use custom error types for recoverable errors
- Log errors with context but never expose secrets

```go
// Good
if err != nil {
    return fmt.Errorf("failed to load config from %s: %w", path, err)
}

// Bad - no context
if err != nil {
    return err
}
```

## Security-Critical Code

When working with authentication code:

- Never log tokens, keys, or passwords in plain text
- Use `secrets.HashToken(token)` for debug logging
- Validate file permissions before reading private keys
- Zero sensitive byte slices after use when possible

```go
// Good - hash tokens for logging
logger.Debug("token retrieved", "hash", secrets.HashToken(token))

// Bad - exposes token
logger.Debug("token retrieved", "token", token)
```

## Testing

- Use table-driven tests for multiple scenarios
- Use `t.TempDir()` for file system tests
- Use `t.Setenv()` for environment variable tests
- Prefer interfaces for mockable dependencies

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "foo", "bar", false},
        {"empty input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Feature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Feature() error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("Feature() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Cobra Commands

Each command follows this pattern:

```go
func NewExampleCmd() *cobra.Command {
    var flagValue string
    
    cmd := &cobra.Command{
        Use:   "example",
        Short: "Brief description",
        Long:  `Detailed description with examples.`,
        RunE: func(cmd *cobra.Command, args []string) error {
            return exampleRun(cmd, flagValue)
        },
    }
    
    cmd.Flags().StringVar(&flagValue, "flag", "", "Flag description")
    
    return cmd
}

func exampleRun(cmd *cobra.Command, flagValue string) error {
    // Implementation
    return nil
}
```

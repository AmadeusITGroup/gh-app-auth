---
mode: agent
description: "Add a new CLI command to gh-app-auth"
---

# Add New CLI Command

Create a new command for the gh-app-auth CLI extension.

## Requirements

Ask the user for:

1. Command name (e.g., `verify`, `export`)
2. Brief description of what the command does
3. Required flags and their types
4. Expected behavior

## Implementation Steps

1. Create `cmd/{command}.go` with:
   - `New{Command}Cmd()` constructor function
   - `{command}Run()` execution function
   - Cobra command with Use, Short, Long, Example, RunE
   - Flag definitions

2. Register in `cmd/root.go`:
   - Add import if needed
   - Add `rootCmd.AddCommand(New{Command}Cmd())`

3. Create `cmd/{command}_test.go` with:
   - Test for command construction
   - Tests for flag parsing
   - Tests for the run function (table-driven)
   - Error case tests

4. Update documentation:
   - Add to README.md command reference
   - Create or update relevant docs

## Code Template

Use this structure for the command file:

```go
package cmd

import (
    "fmt"

    "github.com/spf13/cobra"
)

func New{Command}Cmd() *cobra.Command {
    var (
        flagName string
    )

    cmd := &cobra.Command{
        Use:   "{command}",
        Short: "Brief description",
        Long: `Detailed description.

Examples:
  gh app-auth {command}
  gh app-auth {command} --flag value`,
        RunE: func(cmd *cobra.Command, args []string) error {
            return {command}Run(cmd, flagName)
        },
    }

    cmd.Flags().StringVar(&flagName, "flag", "", "Flag description")

    return cmd
}

func {command}Run(cmd *cobra.Command, flagName string) error {
    // Implementation
    return nil
}
```

## Validation Checklist

- [ ] Command builds without errors
- [ ] `gh app-auth {command} --help` shows correct usage
- [ ] Tests pass (`go test ./cmd/...`)
- [ ] Linting passes (`make quality`)

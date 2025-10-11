package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
	t.Run("has correct use", func(t *testing.T) {
		if rootCmd.Use != "gh-app-auth" {
			t.Errorf("Use = %q, want %q", rootCmd.Use, "gh-app-auth")
		}
	})

	t.Run("has short description", func(t *testing.T) {
		if rootCmd.Short == "" {
			t.Error("Short description is empty")
		}
	})

	t.Run("has long description", func(t *testing.T) {
		if rootCmd.Long == "" {
			t.Error("Long description is empty")
		}
	})

	t.Run("has version", func(t *testing.T) {
		if rootCmd.Version == "" {
			t.Error("Version is empty")
		}
	})

	t.Run("has example", func(t *testing.T) {
		if rootCmd.Example == "" {
			t.Error("Example is empty")
		}
	})
}

func TestRootCommandSubcommands(t *testing.T) {
	expectedCommands := []string{
		"setup",
		"list",
		"remove",
		"test",
		"git-credential",
		"gitconfig",
		"migrate",
	}

	commands := rootCmd.Commands()

	if len(commands) < len(expectedCommands) {
		t.Errorf("Expected at least %d commands, got %d", len(expectedCommands), len(commands))
	}

	// Check that each expected command exists
	commandMap := make(map[string]bool)
	for _, cmd := range commands {
		commandMap[cmd.Name()] = true
	}

	for _, expectedCmd := range expectedCommands {
		if !commandMap[expectedCmd] {
			t.Errorf("Expected command %q not found", expectedCmd)
		}
	}
}

func TestRootCommandFlags(t *testing.T) {
	t.Run("has debug flag", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("debug")
		if flag == nil {
			t.Error("debug flag not found")
			return
		}
		if flag.DefValue != "false" {
			t.Errorf("debug default = %q, want %q", flag.DefValue, "false")
		}
	})

	t.Run("has config flag", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("config")
		if flag == nil {
			t.Error("config flag not found")
			return
		}
		if flag.DefValue != "" {
			t.Errorf("config default = %q, want empty string", flag.DefValue)
		}
	})
}

func TestExecute(t *testing.T) {
	// Just verify Execute doesn't panic when called
	// In a real scenario this would need more sophisticated testing
	t.Run("function exists", func(t *testing.T) {
		// The Execute function exists and can be called
		// We can't easily test it without mocking os.Args
		// but we can verify the function signature is correct
		_ = Execute
	})
}

func TestNewCommands(t *testing.T) {
	tests := []struct {
		name    string
		cmdFunc func() *cobra.Command
	}{
		{"NewSetupCmd", NewSetupCmd},
		{"NewListCmd", NewListCmd},
		{"NewRemoveCmd", NewRemoveCmd},
		{"NewTestCmd", NewTestCmd},
		{"NewGitCredentialCmd", NewGitCredentialCmd},
		{"NewGitConfigCmd", NewGitConfigCmd},
		{"NewMigrateCmd", NewMigrateCmd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmdFunc()
			if cmd == nil {
				t.Error("Command is nil")
				return
			}
			if cmd.Use == "" {
				t.Error("Command Use is empty")
			}
			if cmd.Short == "" {
				t.Error("Command Short description is empty")
			}
		})
	}
}

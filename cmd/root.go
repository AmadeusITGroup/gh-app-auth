package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gh-app-auth",
	Short: "GitHub App authentication for GitHub CLI",
	Long: `GitHub App Authentication Extension

This extension enables GitHub App authentication for Git operations and API access.
It provides repository-specific authentication using GitHub Apps with pattern matching
and secure token caching.

Features:
  • GitHub App JWT token generation and caching
  • Git credential helper integration  
  • Repository-specific app configuration
  • Secure private key handling
  • Multi-organization support`,
	Version: "1.0.0",
	Example: `  # Configure GitHub App
  gh app-auth setup --app-id 123456 --key-file app.pem --patterns "github.com/myorg/*"
  
  # List configured apps
  gh app-auth list
  
  # Test authentication
  gh app-auth test --repo github.com/myorg/private-repo`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(NewSetupCmd())
	rootCmd.AddCommand(NewListCmd())
	rootCmd.AddCommand(NewRemoveCmd())
	rootCmd.AddCommand(NewTestCmd())
	rootCmd.AddCommand(NewGitCredentialCmd())
	
	// Global flags
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug output")
	rootCmd.PersistentFlags().String("config", "", "Path to configuration file")
}

// Version information
func init() {
	rootCmd.SetVersionTemplate(`{{printf "%s version %s\n" .Name .Version}}`)
}

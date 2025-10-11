package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/spf13/cobra"
	"github.com/wherka-ama/gh-app-auth/pkg/config"
)

func NewListCmd() *cobra.Command {
	var (
		format string
		quiet  bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured GitHub Apps",
		Long: `List all configured GitHub Apps with their settings.

Shows the configured GitHub Apps, their patterns, priorities, and status.`,
		Aliases: []string{"ls"},
		Example: `  # List all configured apps
  gh app-auth list
  
  # List with JSON output
  gh app-auth list --format json
  
  # Quiet output (just app IDs)
  gh app-auth list --quiet`,
		RunE: listRun(&format, &quiet),
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json, yaml")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only show app IDs")

	return cmd
}

func listRun(format *string, quiet *bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if os.IsNotExist(err) {
			fmt.Printf("No GitHub Apps configured. Run 'gh app-auth setup' to add one.\n")
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		if len(cfg.GitHubApps) == 0 {
			fmt.Printf("No GitHub Apps configured. Run 'gh app-auth setup' to add one.\n")
			return nil
		}

		// Quiet mode - just show app IDs
		if *quiet {
			for _, app := range cfg.GitHubApps {
				fmt.Printf("%d\n", app.AppID)
			}
			return nil
		}

		// Handle different output formats
		switch *format {
		case "json":
			return outputJSON(cfg.GitHubApps)
		case "yaml":
			return outputYAML(cfg.GitHubApps)
		case "table":
			return outputTable(cfg.GitHubApps)
		default:
			return fmt.Errorf("unsupported format: %s (supported: table, json, yaml)", *format)
		}
	}
}

func outputTable(apps []config.GitHubApp) error {
	// Create table printer
	terminal := os.Stdout
	width := 120 // Default width
	tp := tableprinter.New(terminal, false, width)

	// Add headers
	tp.AddField("NAME", tableprinter.WithTruncate(nil))
	tp.AddField("APP ID", tableprinter.WithTruncate(nil))
	tp.AddField("INSTALLATION ID", tableprinter.WithTruncate(nil))
	tp.AddField("PATTERNS", tableprinter.WithTruncate(nil))
	tp.AddField("PRIORITY", tableprinter.WithTruncate(nil))
	tp.AddField("KEY FILE", tableprinter.WithTruncate(nil))
	tp.EndRow()

	// Add data rows
	for _, app := range apps {
		tp.AddField(app.Name, tableprinter.WithTruncate(nil))
		tp.AddField(fmt.Sprintf("%d", app.AppID), tableprinter.WithTruncate(nil))
		
		installationDisplay := "auto-detect"
		if app.InstallationID > 0 {
			installationDisplay = fmt.Sprintf("%d", app.InstallationID)
		}
		tp.AddField(installationDisplay, tableprinter.WithTruncate(nil))
		
		tp.AddField(strings.Join(app.Patterns, ", "), tableprinter.WithTruncate(nil))
		tp.AddField(fmt.Sprintf("%d", app.Priority), tableprinter.WithTruncate(nil))
		tp.AddField(app.PrivateKeyPath, tableprinter.WithTruncate(nil))
		tp.EndRow()
	}

	return tp.Render()
}

func outputJSON(apps []config.GitHubApp) error {
	return config.OutputJSON(os.Stdout, apps)
}

func outputYAML(apps []config.GitHubApp) error {
	return config.OutputYAML(os.Stdout, apps)
}

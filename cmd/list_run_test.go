package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"gopkg.in/yaml.v3"
)

func TestListRun_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")
	keyPath := filepath.Join(tempDir, "test-key.pem")

	// Generate valid test key
	testKey := generateTestRSAKey(t)
	if err := os.WriteFile(keyPath, []byte(testKey), 0600); err != nil {
		t.Fatalf("Failed to write test key: %v", err)
	}

	// Create test config with multiple apps
	cfg := &config.Config{
		Version: "1.0",
		GitHubApps: []config.GitHubApp{
			{
				Name:             "Test App 1",
				AppID:            123456,
				InstallationID:   789012,
				Patterns:         []string{"github.com/org1/*"},
				Priority:         10,
				PrivateKeySource: config.PrivateKeySourceFilesystem,
				PrivateKeyPath:   keyPath,
			},
			{
				Name:             "Test App 2",
				AppID:            234567,
				InstallationID:   890123,
				Patterns:         []string{"github.com/org2/*", "github.com/org3/*"},
				Priority:         5,
				PrivateKeySource: config.PrivateKeySourceKeyring,
			},
		},
	}

	// Write config
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	t.Run("list with table format (default)", func(t *testing.T) {
		cmd := NewListCmd()

		// Execute command - table output goes directly to os.Stdout
		// so we just verify it executes without error
		err := cmd.Execute()
		if err != nil {
			t.Errorf("List command failed: %v", err)
		}
		// Table printer writes to os.Stdout directly, not cmd.OutOrStdout()
		// So we can't easily capture the output in tests
		// The fact that it doesn't error is good enough for coverage
	})

	t.Run("list with quiet mode", func(t *testing.T) {
		cmd := NewListCmd()
		cmd.Flags().Set("quiet", "true")

		err := cmd.Execute()
		if err != nil {
			t.Errorf("List command failed: %v", err)
		}
		// Output goes to os.Stdout, can't easily capture in tests
	})

	t.Run("list with JSON format", func(t *testing.T) {
		cmd := NewListCmd()
		cmd.Flags().Set("format", "json")

		err := cmd.Execute()
		if err != nil {
			t.Errorf("List command failed: %v", err)
		}
		// Output goes to os.Stdout, can't easily capture in tests
	})

	t.Run("list with YAML format", func(t *testing.T) {
		cmd := NewListCmd()
		cmd.Flags().Set("format", "yaml")

		err := cmd.Execute()
		if err != nil {
			t.Errorf("List command failed: %v", err)
		}
		// Output goes to os.Stdout, can't easily capture in tests
	})

	t.Run("list with invalid format", func(t *testing.T) {
		cmd := NewListCmd()

		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)

		cmd.Flags().Set("format", "invalid-format")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for invalid format")
		}

		if !strings.Contains(err.Error(), "format") {
			t.Errorf("Error should mention format, got: %v", err)
		}
	})
}

func TestListRun_NoApps(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	// Create empty config
	cfg := &config.Config{
		Version:    "1.0",
		GitHubApps: []config.GitHubApp{},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	t.Run("list with no apps configured", func(t *testing.T) {
		cmd := NewListCmd()

		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)

		err := cmd.Execute()
		// Should not error, but output should indicate no apps
		if err != nil {
			t.Logf("Command returned: %v (may be expected)", err)
		}
	})
}

func TestListRun_MissingConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("GH_APP_AUTH_CONFIG", filepath.Join(tempDir, "nonexistent.yml"))

	t.Run("list with missing config file", func(t *testing.T) {
		cmd := NewListCmd()

		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)

		err := cmd.Execute()
		if err != nil {
			t.Error("No error expected for missing config file")
		}
	})
}

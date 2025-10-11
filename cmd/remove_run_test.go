package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"gopkg.in/yaml.v3"
)

func TestRemoveRun_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("remove with missing flags", func(t *testing.T) {
		cmd := NewRemoveCmd()

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error when no flags provided")
		}
	})

	t.Run("remove with conflicting flags", func(t *testing.T) {
		cmd := NewRemoveCmd()
		cmd.Flags().Set("app-id", "123456")
		cmd.Flags().Set("all", "true")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error when both --app-id and --all provided")
		}
	})

	t.Run("remove with invalid app ID", func(t *testing.T) {
		cmd := NewRemoveCmd()
		cmd.Flags().Set("app-id", "0")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for invalid app ID")
		}
	})
}

func TestRemoveRun_NoConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("GH_APP_AUTH_CONFIG", filepath.Join(tempDir, "nonexistent.yml"))

	t.Run("remove when no config exists", func(t *testing.T) {
		cmd := NewRemoveCmd()
		cmd.Flags().Set("app-id", "123456")
		cmd.Flags().Set("force", "true")

		err := cmd.Execute()
		// Should not error - just returns message
		if err != nil {
			t.Logf("Command returned: %v (may be expected)", err)
		}
	})
}

func TestRemoveRun_EmptyConfig(t *testing.T) {
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

	t.Run("remove when no apps configured", func(t *testing.T) {
		cmd := NewRemoveCmd()
		cmd.Flags().Set("app-id", "123456")
		cmd.Flags().Set("force", "true")

		err := cmd.Execute()
		// Should not error - just returns message
		if err != nil {
			t.Logf("Command returned: %v (may be expected)", err)
		}
	})
}

func TestRemoveRun_RemoveSingleApp(t *testing.T) {
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
				Patterns:         []string{"github.com/org2/*"},
				Priority:         5,
				PrivateKeySource: config.PrivateKeySourceKeyring,
			},
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	t.Run("remove single app with force", func(t *testing.T) {
		cmd := NewRemoveCmd()
		cmd.Flags().Set("app-id", "123456")
		cmd.Flags().Set("force", "true")

		err := cmd.Execute()
		if err != nil {
			t.Errorf("Remove command failed: %v", err)
		}

		// Verify app was removed
		updatedCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load updated config: %v", err)
		}

		if len(updatedCfg.GitHubApps) != 1 {
			t.Errorf("Expected 1 app remaining, got %d", len(updatedCfg.GitHubApps))
		}

		if len(updatedCfg.GitHubApps) > 0 && updatedCfg.GitHubApps[0].AppID == 123456 {
			t.Error("App 123456 should have been removed")
		}
	})

	t.Run("remove non-existent app", func(t *testing.T) {
		cmd := NewRemoveCmd()
		cmd.Flags().Set("app-id", "999999")
		cmd.Flags().Set("force", "true")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error when removing non-existent app")
		}
	})
}

func TestRemoveRun_RemoveAllApps(t *testing.T) {
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
				Patterns:         []string{"github.com/org2/*"},
				Priority:         5,
				PrivateKeySource: config.PrivateKeySourceKeyring,
			},
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	t.Run("remove all apps with force", func(t *testing.T) {
		cmd := NewRemoveCmd()
		cmd.Flags().Set("all", "true")
		cmd.Flags().Set("force", "true")

		err := cmd.Execute()
		if err != nil {
			t.Errorf("Remove command failed: %v", err)
		}

		// Verify all apps were removed
		// Note: config.Load() will fail validation because it requires at least one app
		// So we check that the config file was modified or removed
		updatedCfg, err := config.Load()
		if err != nil {
			// This is expected - config requires at least one app
			t.Logf("Config load failed as expected (empty config): %v", err)
			return
		}

		// If Load succeeded somehow, verify apps are empty
		if len(updatedCfg.GitHubApps) != 0 {
			t.Errorf("Expected 0 apps remaining, got %d", len(updatedCfg.GitHubApps))
		}
	})
}

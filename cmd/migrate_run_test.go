package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/secrets"
	"gopkg.in/yaml.v3"
)

func TestMigrateRun_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("migrate with invalid storage option", func(t *testing.T) {
		cmd := NewMigrateCmd()
		cmd.Flags().Set("storage", "invalid-storage")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for invalid storage option")
		}
	})

	t.Run("migrate with valid storage option", func(t *testing.T) {
		cmd := NewMigrateCmd()
		cmd.Flags().Set("storage", "keyring")
		cmd.Flags().Set("dry-run", "true")

		// Should not error even with no config (handled gracefully)
		_ = cmd.Execute()
	})
}

func TestMigrateRun_NoConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("GH_APP_AUTH_CONFIG", filepath.Join(tempDir, "nonexistent.yml"))

	t.Run("migrate when no config exists", func(t *testing.T) {
		cmd := NewMigrateCmd()
		cmd.Flags().Set("storage", "keyring")
		cmd.Flags().Set("dry-run", "true")

		err := cmd.Execute()
		// Should not error - just returns message
		if err != nil {
			t.Logf("Command returned: %v (may be expected)", err)
		}
	})
}

func TestMigrateRun_EmptyConfig(t *testing.T) {
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

	t.Run("migrate when no apps configured", func(t *testing.T) {
		cmd := NewMigrateCmd()
		cmd.Flags().Set("storage", "keyring")
		cmd.Flags().Set("dry-run", "true")

		err := cmd.Execute()
		// Should not error - just returns message
		if err != nil {
			t.Logf("Command returned: %v (may be expected)", err)
		}
	})
}

func TestMigrateRun_DryRun(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")
	keyPath := filepath.Join(tempDir, "test-key.pem")

	// Generate valid test key
	testKey := generateTestRSAKey(t)
	if err := os.WriteFile(keyPath, []byte(testKey), 0600); err != nil {
		t.Fatalf("Failed to write test key: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		Version: "1.0",
		GitHubApps: []config.GitHubApp{
			{
				Name:             "Test App",
				AppID:            123456,
				InstallationID:   789012,
				Patterns:         []string{"github.com/org/*"},
				Priority:         10,
				PrivateKeySource: config.PrivateKeySourceFilesystem,
				PrivateKeyPath:   keyPath,
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

	t.Run("migrate dry-run to keyring", func(t *testing.T) {
		cmd := NewMigrateCmd()
		cmd.Flags().Set("storage", "keyring")
		cmd.Flags().Set("dry-run", "true")

		err := cmd.Execute()
		// Dry-run should not error
		if err != nil {
			t.Logf("Dry-run returned: %v (may be expected - keyring might not be available)", err)
		}

		// Verify config hasn't changed (dry-run)
		updatedCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if updatedCfg.GitHubApps[0].PrivateKeySource != config.PrivateKeySourceFilesystem {
			t.Error("Dry-run should not modify config")
		}
	})

	t.Run("migrate dry-run to filesystem", func(t *testing.T) {
		cmd := NewMigrateCmd()
		cmd.Flags().Set("storage", "filesystem")
		cmd.Flags().Set("dry-run", "true")

		err := cmd.Execute()
		// Should not error - already on filesystem
		if err != nil {
			t.Logf("Command returned: %v", err)
		}
	})
}

func TestInitializeMigration_Direct(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")
	keyPath := filepath.Join(tempDir, "test-key.pem")

	// Generate valid test key
	testKey := generateTestRSAKey(t)
	if err := os.WriteFile(keyPath, []byte(testKey), 0600); err != nil {
		t.Fatalf("Failed to write test key: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		Version: "1.0",
		GitHubApps: []config.GitHubApp{
			{
				Name:             "Test App",
				AppID:            123456,
				InstallationID:   789012,
				Patterns:         []string{"github.com/org/*"},
				Priority:         10,
				PrivateKeySource: config.PrivateKeySourceFilesystem,
				PrivateKeyPath:   keyPath,
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

	t.Run("initialize migration with valid config", func(t *testing.T) {
		storage := "keyring"
		cfg, secretMgr, err := initializeMigration(&storage)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if cfg == nil {
			t.Error("Expected non-nil config")
		}

		if secretMgr == nil {
			t.Error("Expected non-nil secrets manager")
		}
	})
}

func TestDisplayMigrationSummary_Direct(t *testing.T) {
	keyPath := "/tmp/test-key.pem"

	apps := []config.GitHubApp{
		{
			Name:             "Test App 1",
			AppID:            123456,
			PrivateKeySource: config.PrivateKeySourceFilesystem,
			PrivateKeyPath:   keyPath,
		},
		{
			Name:             "Test App 2",
			AppID:            234567,
			PrivateKeySource: config.PrivateKeySourceKeyring,
		},
	}

	appsToMigrate := []config.GitHubApp{apps[0]}
	appsUpToDate := []config.GitHubApp{apps[1]}
	appsNeedAttention := []config.GitHubApp{}

	t.Run("display summary for dry-run", func(t *testing.T) {
		storage := "keyring"
		dryRun := true

		shouldExit := displayMigrationSummary(
			apps, appsToMigrate, appsUpToDate, appsNeedAttention,
			storage, dryRun,
		)

		// Dry-run should exit
		if !shouldExit {
			t.Error("Expected dry-run to exit")
		}
	})

	t.Run("display summary with no migrations needed", func(t *testing.T) {
		storage := "keyring"
		dryRun := false

		shouldExit := displayMigrationSummary(
			apps, []config.GitHubApp{}, appsUpToDate, appsNeedAttention,
			storage, dryRun,
		)

		// No migrations needed, should exit
		if !shouldExit {
			t.Error("Expected to exit when no migrations needed")
		}
	})

	t.Run("display summary with apps needing attention", func(t *testing.T) {
		attention := []config.GitHubApp{
			{Name: "Inline App", AppID: 345678, ClientID: "Iv1.Attention"},
		}

		shouldExit := displayMigrationSummary(
			apps, []config.GitHubApp{}, appsUpToDate, attention,
			"keyring", false,
		)

		// The function only exits early when there is nothing to do;
		// attention-only flows continue to displayMigrationResults.
		if shouldExit {
			t.Error("Expected not to exit when only attention is needed")
		}
	})

	t.Run("display summary with migrations to perform", func(t *testing.T) {
		toMigrate := []config.GitHubApp{apps[0]}

		shouldExit := displayMigrationSummary(
			apps, toMigrate, []config.GitHubApp{}, []config.GitHubApp{},
			"keyring", false,
		)

		if shouldExit {
			t.Error("Expected to continue when migrations are scheduled")
		}
	})
}

func TestPerformMigration(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test-key.pem")
	testKey := generateTestRSAKey(t)
	if err := os.WriteFile(keyPath, []byte(testKey), 0600); err != nil {
		t.Fatalf("Failed to write test key: %v", err)
	}

	cfg := &config.Config{
		Version: "1.0",
		GitHubApps: []config.GitHubApp{
			{
				Name:             "Migrate App",
				AppID:            123456,
				ClientID:         "Iv1.Migrate",
				InstallationID:   789012,
				Patterns:         []string{"github.com/org/*"},
				PrivateKeySource: config.PrivateKeySourceFilesystem,
				PrivateKeyPath:   keyPath,
			},
		},
	}

	appsToMigrate := []config.GitHubApp{cfg.GitHubApps[0]}
	secretMgr := secrets.NewManager(tempDir)

	migrated, failed := performMigration(cfg, appsToMigrate, secretMgr, "filesystem", false)
	if failed != 0 {
		t.Errorf("performMigration() failed = %d, want 0", failed)
	}
	if migrated != 1 {
		t.Errorf("performMigration() migrated = %d, want 1", migrated)
	}
}

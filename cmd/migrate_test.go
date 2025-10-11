package cmd

import (
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

func TestValidateStorageOption(t *testing.T) {
	tests := []struct {
		name    string
		storage string
		wantErr bool
	}{
		{
			name:    "valid keyring",
			storage: "keyring",
			wantErr: false,
		},
		{
			name:    "valid filesystem",
			storage: "filesystem",
			wantErr: false,
		},
		{
			name:    "invalid storage",
			storage: "invalid",
			wantErr: true,
		},
		{
			name:    "empty string",
			storage: "",
			wantErr: true,
		},
		{
			name:    "uppercase",
			storage: "KEYRING",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStorageOption(tt.storage)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateStorageOption() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalyzeAppsForMigration(t *testing.T) {
	tests := []struct {
		name                   string
		apps                   []config.GitHubApp
		targetStorage          string
		wantToMigrateCount     int
		wantUpToDateCount      int
		wantNeedAttentionCount int
	}{
		{
			name: "all inline to keyring",
			apps: []config.GitHubApp{
				{Name: "App1", PrivateKeySource: config.PrivateKeySourceInline},
				{Name: "App2", PrivateKeySource: config.PrivateKeySourceInline},
			},
			targetStorage:          "keyring",
			wantToMigrateCount:     2,
			wantUpToDateCount:      0,
			wantNeedAttentionCount: 0,
		},
		{
			name: "all already keyring",
			apps: []config.GitHubApp{
				{Name: "App1", PrivateKeySource: config.PrivateKeySourceKeyring},
				{Name: "App2", PrivateKeySource: config.PrivateKeySourceKeyring},
			},
			targetStorage:          "keyring",
			wantToMigrateCount:     0,
			wantUpToDateCount:      2,
			wantNeedAttentionCount: 0,
		},
		{
			name: "all already filesystem",
			apps: []config.GitHubApp{
				{Name: "App1", PrivateKeySource: config.PrivateKeySourceFilesystem, PrivateKeyPath: "/path1"},
				{Name: "App2", PrivateKeySource: config.PrivateKeySourceFilesystem, PrivateKeyPath: "/path2"},
			},
			targetStorage:          "filesystem",
			wantToMigrateCount:     0,
			wantUpToDateCount:      2,
			wantNeedAttentionCount: 0,
		},
		{
			name: "mixed sources to keyring",
			apps: []config.GitHubApp{
				{Name: "App1", PrivateKeySource: config.PrivateKeySourceInline},
				{Name: "App2", PrivateKeySource: config.PrivateKeySourceKeyring},
				{Name: "App3", PrivateKeySource: config.PrivateKeySourceFilesystem, PrivateKeyPath: "/path"},
			},
			targetStorage:          "keyring",
			wantToMigrateCount:     2,
			wantUpToDateCount:      1,
			wantNeedAttentionCount: 0,
		},
		{
			name: "filesystem to keyring",
			apps: []config.GitHubApp{
				{Name: "App1", PrivateKeySource: config.PrivateKeySourceFilesystem, PrivateKeyPath: "/path"},
			},
			targetStorage:          "keyring",
			wantToMigrateCount:     1,
			wantUpToDateCount:      0,
			wantNeedAttentionCount: 0,
		},
		{
			name: "keyring to filesystem without path",
			apps: []config.GitHubApp{
				{Name: "App1", PrivateKeySource: config.PrivateKeySourceKeyring, PrivateKeyPath: ""},
			},
			targetStorage:          "filesystem",
			wantToMigrateCount:     1, // Will be added to migrate, but will fail during migration
			wantUpToDateCount:      0,
			wantNeedAttentionCount: 0,
		},
		{
			name: "keyring to filesystem with path",
			apps: []config.GitHubApp{
				{Name: "App1", PrivateKeySource: config.PrivateKeySourceKeyring, PrivateKeyPath: "/path"},
			},
			targetStorage:          "filesystem",
			wantToMigrateCount:     1,
			wantUpToDateCount:      0,
			wantNeedAttentionCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toMigrate, upToDate, needAttention := analyzeAppsForMigration(tt.apps, tt.targetStorage)

			if len(toMigrate) != tt.wantToMigrateCount {
				t.Errorf("toMigrate count = %d, want %d", len(toMigrate), tt.wantToMigrateCount)
			}

			if len(upToDate) != tt.wantUpToDateCount {
				t.Errorf("upToDate count = %d, want %d", len(upToDate), tt.wantUpToDateCount)
			}

			if len(needAttention) != tt.wantNeedAttentionCount {
				t.Errorf("needAttention count = %d, want %d", len(needAttention), tt.wantNeedAttentionCount)
			}
		})
	}
}

func TestNeedsMigration(t *testing.T) {
	appsToMigrate := []config.GitHubApp{
		{Name: "App1", AppID: 111},
		{Name: "App2", AppID: 222},
	}

	tests := []struct {
		name string
		app  *config.GitHubApp
		want bool
	}{
		{
			name: "needs migration - in list",
			app:  &config.GitHubApp{Name: "App1", AppID: 111},
			want: true,
		},
		{
			name: "needs migration - second in list",
			app:  &config.GitHubApp{Name: "App2", AppID: 222},
			want: true,
		},
		{
			name: "does not need migration",
			app:  &config.GitHubApp{Name: "App3", AppID: 333},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsMigration(tt.app, appsToMigrate)
			if got != tt.want {
				t.Errorf("needsMigration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMigrateToFilesystem(t *testing.T) {
	tests := []struct {
		name    string
		app     *config.GitHubApp
		wantErr bool
	}{
		{
			name: "with path",
			app: &config.GitHubApp{
				Name:           "Test App",
				PrivateKeyPath: "/path/to/key.pem",
			},
			wantErr: false,
		},
		{
			name: "without path",
			app: &config.GitHubApp{
				Name:           "Test App",
				PrivateKeyPath: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := migrateToFilesystem(tt.app)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.app.PrivateKeySource != config.PrivateKeySourceFilesystem {
				t.Errorf("PrivateKeySource = %v, want %v",
					tt.app.PrivateKeySource, config.PrivateKeySourceFilesystem)
			}
		})
	}
}

func TestHandleOriginalKeyFile(t *testing.T) {
	t.Run("force false", func(t *testing.T) {
		app := &config.GitHubApp{
			Name:           "Test",
			PrivateKeyPath: "/path/to/key.pem",
		}
		// Should not panic
		handleOriginalKeyFile(app, false)
	})

	t.Run("force true with path", func(t *testing.T) {
		app := &config.GitHubApp{
			Name:           "Test",
			PrivateKeyPath: "/nonexistent/key.pem",
		}
		// Should not panic even if file doesn't exist
		handleOriginalKeyFile(app, true)
	})

	t.Run("force true without path", func(t *testing.T) {
		app := &config.GitHubApp{
			Name:           "Test",
			PrivateKeyPath: "",
		}
		// Should not panic
		handleOriginalKeyFile(app, true)
	})
}

func TestDisplayMigrationResults(t *testing.T) {
	tests := []struct {
		name     string
		migrated int
		failed   int
		force    bool
		storage  string
	}{
		{
			name:     "successful migration to keyring",
			migrated: 3,
			failed:   0,
			force:    false,
			storage:  "keyring",
		},
		{
			name:     "partial failure",
			migrated: 2,
			failed:   1,
			force:    false,
			storage:  "keyring",
		},
		{
			name:     "forced migration",
			migrated: 1,
			failed:   0,
			force:    true,
			storage:  "filesystem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic
			displayMigrationResults(tt.migrated, tt.failed, tt.force, tt.storage)
		})
	}
}

package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/secrets"
	"github.com/zalando/go-keyring"
)

func TestGitHubApp_SetPrivateKey_Keyring(t *testing.T) {
	// Given: Keyring is available
	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	secretMgr := secrets.NewManager(tempDir)

	app := &GitHubApp{
		Name:           "test-app",
		AppID:          12345,
		InstallationID: 67890,
	}

	// When: We set a private key
	backend, err := app.SetPrivateKey(secretMgr, "test-private-key")

	// Then: It succeeds and uses keyring
	if err != nil {
		t.Fatalf("SetPrivateKey() failed: %v", err)
	}
	if backend != secrets.StorageBackendKeyring {
		t.Errorf("SetPrivateKey() backend = %v, want %v", backend, secrets.StorageBackendKeyring)
	}
	if app.PrivateKeySource != PrivateKeySourceKeyring {
		t.Errorf("app.PrivateKeySource = %v, want %v", app.PrivateKeySource, PrivateKeySourceKeyring)
	}

	// And: Key is stored in keyring
	storedKey, err := keyring.Get("gh-app-auth:test-app", "private_key")
	if err != nil {
		t.Fatalf("Failed to get from keyring: %v", err)
	}
	if storedKey != "test-private-key" {
		t.Errorf("Stored key = %v, want %v", storedKey, "test-private-key")
	}
}

func TestGitHubApp_SetPrivateKey_FilesystemFallback(t *testing.T) {
	// Given: Keyring is unavailable
	keyring.MockInitWithError(errors.New("keyring unavailable"))
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	secretMgr := secrets.NewManager(tempDir)

	app := &GitHubApp{
		Name:           "test-app",
		AppID:          12345,
		InstallationID: 67890,
	}

	// When: We set a private key
	backend, err := app.SetPrivateKey(secretMgr, "test-private-key")

	// Then: It succeeds and uses filesystem
	if err != nil {
		t.Fatalf("SetPrivateKey() failed: %v", err)
	}
	if backend != secrets.StorageBackendFilesystem {
		t.Errorf("SetPrivateKey() backend = %v, want %v", backend, secrets.StorageBackendFilesystem)
	}
	if app.PrivateKeySource != PrivateKeySourceFilesystem {
		t.Errorf("app.PrivateKeySource = %v, want %v", app.PrivateKeySource, PrivateKeySourceFilesystem)
	}
}

func TestGitHubApp_GetPrivateKey_FromKeyring(t *testing.T) {
	// Given: Key is stored in keyring
	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	secretMgr := secrets.NewManager(tempDir)

	app := &GitHubApp{
		Name:             "test-app",
		PrivateKeySource: PrivateKeySourceKeyring,
	}

	// Store key in keyring
	err := keyring.Set("gh-app-auth:test-app", "private_key", "keyring-stored-key")
	if err != nil {
		t.Fatalf("Failed to set in keyring: %v", err)
	}

	// When: We get the private key
	key, err := app.GetPrivateKey(secretMgr)

	// Then: It succeeds and returns the key
	if err != nil {
		t.Fatalf("GetPrivateKey() failed: %v", err)
	}
	if key != "keyring-stored-key" {
		t.Errorf("GetPrivateKey() = %v, want %v", key, "keyring-stored-key")
	}
}

func TestGitHubApp_GetPrivateKey_FromFilesystem(t *testing.T) {
	// Given: Key is stored in filesystem
	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	secretMgr := secrets.NewManager(tempDir)

	keyPath := filepath.Join(tempDir, "test-key.pem")
	err := os.WriteFile(keyPath, []byte("filesystem-stored-key"), 0600)
	if err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	app := &GitHubApp{
		Name:             "test-app",
		PrivateKeySource: PrivateKeySourceFilesystem,
		PrivateKeyPath:   keyPath,
	}

	// When: We get the private key
	key, err := app.GetPrivateKey(secretMgr)

	// Then: It succeeds and returns the key
	if err != nil {
		t.Fatalf("GetPrivateKey() failed: %v", err)
	}
	if key != "filesystem-stored-key" {
		t.Errorf("GetPrivateKey() = %v, want %v", key, "filesystem-stored-key")
	}
}

func TestGitHubApp_GetPrivateKey_KeyringWithFilesystemFallback(t *testing.T) {
	// Given: Keyring source configured but key only in filesystem
	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	secretMgr := secrets.NewManager(tempDir)

	keyPath := filepath.Join(tempDir, "test-key.pem")
	err := os.WriteFile(keyPath, []byte("fallback-key"), 0600)
	if err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	app := &GitHubApp{
		Name:             "test-app",
		PrivateKeySource: PrivateKeySourceKeyring,
		PrivateKeyPath:   keyPath, // Fallback path
	}

	// When: We get the private key (keyring empty, should fall back)
	key, err := app.GetPrivateKey(secretMgr)

	// Then: It succeeds using filesystem fallback
	if err != nil {
		t.Fatalf("GetPrivateKey() failed: %v", err)
	}
	if key != "fallback-key" {
		t.Errorf("GetPrivateKey() = %v, want %v", key, "fallback-key")
	}
}

func TestGitHubApp_DeletePrivateKey(t *testing.T) {
	// Given: Key is stored in both keyring and filesystem
	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	secretMgr := secrets.NewManager(tempDir)

	app := &GitHubApp{
		Name:             "test-app",
		PrivateKeySource: PrivateKeySourceKeyring,
	}

	// Store in keyring
	_, err := app.SetPrivateKey(secretMgr, "test-key")
	if err != nil {
		t.Fatalf("SetPrivateKey() failed: %v", err)
	}

	// When: We delete the private key
	err = app.DeletePrivateKey(secretMgr)

	// Then: It succeeds
	if err != nil {
		t.Fatalf("DeletePrivateKey() failed: %v", err)
	}

	// And: Key is removed from keyring
	_, err = keyring.Get("gh-app-auth:test-app", "private_key")
	if !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("Key should be removed from keyring")
	}
}

func TestGitHubApp_HasPrivateKey_Keyring(t *testing.T) {
	// Given: Key is in keyring
	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	secretMgr := secrets.NewManager(tempDir)

	app := &GitHubApp{
		Name:             "test-app",
		PrivateKeySource: PrivateKeySourceKeyring,
	}

	// No key yet
	if app.HasPrivateKey(secretMgr) {
		t.Error("HasPrivateKey() should return false when no key stored")
	}

	// Store key
	_, err := app.SetPrivateKey(secretMgr, "test-key")
	if err != nil {
		t.Fatalf("SetPrivateKey() failed: %v", err)
	}

	// Now has key
	if !app.HasPrivateKey(secretMgr) {
		t.Error("HasPrivateKey() should return true after storing key")
	}
}

func TestGitHubApp_HasPrivateKey_Filesystem(t *testing.T) {
	// Given: Key is on filesystem
	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	secretMgr := secrets.NewManager(tempDir)

	keyPath := filepath.Join(tempDir, "test-key.pem")

	app := &GitHubApp{
		Name:             "test-app",
		PrivateKeySource: PrivateKeySourceFilesystem,
		PrivateKeyPath:   keyPath,
	}

	// No key yet
	if app.HasPrivateKey(secretMgr) {
		t.Error("HasPrivateKey() should return false when file doesn't exist")
	}

	// Create key file
	err := os.WriteFile(keyPath, []byte("test-key"), 0600)
	if err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Now has key
	if !app.HasPrivateKey(secretMgr) {
		t.Error("HasPrivateKey() should return true when file exists")
	}
}

func TestGitHubApp_Validate_WithPrivateKeySource(t *testing.T) {
	tests := []struct {
		name    string
		app     GitHubApp
		wantErr bool
	}{
		{
			name: "valid keyring source",
			app: GitHubApp{
				Name:             "test-app",
				AppID:            12345,
				InstallationID:   67890,
				PrivateKeySource: PrivateKeySourceKeyring,
				Patterns:         []string{"github.com/org/*"},
			},
			wantErr: false,
		},
		{
			name: "valid filesystem source with path",
			app: GitHubApp{
				Name:             "test-app",
				AppID:            12345,
				InstallationID:   67890,
				PrivateKeySource: PrivateKeySourceFilesystem,
				PrivateKeyPath:   "/tmp/key.pem",
				Patterns:         []string{"github.com/org/*"},
			},
			wantErr: false,
		},
		{
			name: "filesystem source missing path",
			app: GitHubApp{
				Name:             "test-app",
				AppID:            12345,
				InstallationID:   67890,
				PrivateKeySource: PrivateKeySourceFilesystem,
				Patterns:         []string{"github.com/org/*"},
			},
			wantErr: true,
		},
		{
			name: "legacy config with path only",
			app: GitHubApp{
				Name:           "test-app",
				AppID:          12345,
				InstallationID: 67890,
				PrivateKeyPath: "/tmp/key.pem",
				Patterns:       []string{"github.com/org/*"},
			},
			wantErr: false,
		},
		{
			name: "inline source rejected",
			app: GitHubApp{
				Name:             "test-app",
				AppID:            12345,
				InstallationID:   67890,
				PrivateKeySource: PrivateKeySourceInline,
				Patterns:         []string{"github.com/org/*"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.app.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check that legacy configs are auto-upgraded
			if tt.name == "legacy config with path only" && err == nil {
				if tt.app.PrivateKeySource != PrivateKeySourceFilesystem {
					t.Errorf("Legacy config should auto-upgrade to filesystem source")
				}
			}
		})
	}
}

package secrets

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestManager_Store_Keyring(t *testing.T) {
	// Given: Keyring is available
	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	mgr := NewManager(tempDir)

	// When: We store a secret
	backend, err := mgr.Store("test-app", SecretTypePrivateKey, "test-key-value")

	// Then: It succeeds and uses keyring
	if err != nil {
		t.Fatalf("Store() failed: %v", err)
	}
	if backend != StorageBackendKeyring {
		t.Errorf("Store() backend = %v, want %v", backend, StorageBackendKeyring)
	}

	// And: The secret can be retrieved from keyring
	value, err := keyring.Get("gh-app-auth:test-app", string(SecretTypePrivateKey))
	if err != nil {
		t.Fatalf("Failed to get from keyring: %v", err)
	}
	if value != "test-key-value" {
		t.Errorf("Keyring value = %v, want %v", value, "test-key-value")
	}
}

func TestManager_Store_FilesystemFallback(t *testing.T) {
	// Given: Keyring is unavailable
	keyring.MockInitWithError(errors.New("keyring unavailable"))
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	mgr := NewManager(tempDir)

	// When: We store a secret
	backend, err := mgr.Store("test-app", SecretTypePrivateKey, "test-key-value")

	// Then: It succeeds and uses filesystem
	if err != nil {
		t.Fatalf("Store() failed: %v", err)
	}
	if backend != StorageBackendFilesystem {
		t.Errorf("Store() backend = %v, want %v", backend, StorageBackendFilesystem)
	}

	// And: The secret exists on filesystem
	path := mgr.filesystemPath("test-app", SecretTypePrivateKey)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read filesystem: %v", err)
	}
	if string(data) != "test-key-value" {
		t.Errorf("Filesystem value = %v, want %v", string(data), "test-key-value")
	}

	// And: File has secure permissions (owner read-only)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	// 0400 = owner read-only
	if info.Mode().Perm() != 0400 {
		t.Errorf("File permissions = %o, want %o", info.Mode().Perm(), 0400)
	}
}

func TestManager_Get_FromKeyring(t *testing.T) {
	// Given: Secret is in keyring
	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	mgr := NewManager(tempDir)

	// Store in keyring
	err := keyring.Set("gh-app-auth:test-app", string(SecretTypePrivateKey), "keyring-value")
	if err != nil {
		t.Fatalf("Failed to set in keyring: %v", err)
	}

	// When: We get the secret
	value, backend, err := mgr.Get("test-app", SecretTypePrivateKey)

	// Then: It succeeds and returns from keyring
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	if backend != StorageBackendKeyring {
		t.Errorf("Get() backend = %v, want %v", backend, StorageBackendKeyring)
	}
	if value != "keyring-value" {
		t.Errorf("Get() value = %v, want %v", value, "keyring-value")
	}
}

func TestManager_Get_FromFilesystem(t *testing.T) {
	// Given: Secret is on filesystem
	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	mgr := NewManager(tempDir)

	// Store on filesystem
	path := mgr.filesystemPath("test-app", SecretTypePrivateKey)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("filesystem-value"), 0400); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// When: We get the secret
	value, backend, err := mgr.Get("test-app", SecretTypePrivateKey)

	// Then: It succeeds and returns from filesystem
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	if backend != StorageBackendFilesystem {
		t.Errorf("Get() backend = %v, want %v", backend, StorageBackendFilesystem)
	}
	if value != "filesystem-value" {
		t.Errorf("Get() value = %v, want %v", value, "filesystem-value")
	}
}

func TestManager_Get_NotFound(t *testing.T) {
	// Given: No secret stored
	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	mgr := NewManager(tempDir)

	// When: We try to get a non-existent secret
	_, _, err := mgr.Get("test-app", SecretTypePrivateKey)

	// Then: We get ErrNotFound
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want %v", err, ErrNotFound)
	}
}

func TestManager_Delete(t *testing.T) {
	// Given: Secret stored in both keyring and filesystem
	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	mgr := NewManager(tempDir)

	// Store in keyring
	err := keyring.Set("gh-app-auth:test-app", string(SecretTypePrivateKey), "test-value")
	if err != nil {
		t.Fatalf("Failed to set in keyring: %v", err)
	}

	// Store on filesystem
	path := mgr.filesystemPath("test-app", SecretTypePrivateKey)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("test-value"), 0400); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// When: We delete the secret
	err = mgr.Delete("test-app", SecretTypePrivateKey)

	// Then: It succeeds
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// And: Secret is removed from keyring
	_, err = keyring.Get("gh-app-auth:test-app", string(SecretTypePrivateKey))
	if !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("Keyring should be empty, got error: %v", err)
	}

	// And: Secret is removed from filesystem
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Filesystem file should not exist")
	}
}

func TestManager_IsAvailable_True(t *testing.T) {
	// Given: Keyring is available
	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	mgr := NewManager(tempDir)

	// When: We check availability
	available := mgr.IsAvailable()

	// Then: It returns true
	if !available {
		t.Error("IsAvailable() = false, want true")
	}
}

func TestManager_IsAvailable_False(t *testing.T) {
	// Given: Keyring is unavailable
	keyring.MockInitWithError(errors.New("keyring unavailable"))
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	mgr := NewManager(tempDir)

	// When: We check availability
	available := mgr.IsAvailable()

	// Then: It returns false
	if available {
		t.Error("IsAvailable() = true, want false")
	}
}

func TestManager_Store_CleansUpFilesystemWhenKeyringSucceeds(t *testing.T) {
	// Given: Secret exists on filesystem
	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	mgr := NewManager(tempDir)

	// Create filesystem secret
	path := mgr.filesystemPath("test-app", SecretTypePrivateKey)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("old-value"), 0400); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// When: We store in keyring
	backend, err := mgr.Store("test-app", SecretTypePrivateKey, "new-value")

	// Then: It succeeds and uses keyring
	if err != nil {
		t.Fatalf("Store() failed: %v", err)
	}
	if backend != StorageBackendKeyring {
		t.Errorf("Store() backend = %v, want %v", backend, StorageBackendKeyring)
	}

	// And: Filesystem file is cleaned up
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Filesystem file should be removed")
	}
}

func TestSecretTypes(t *testing.T) {
	tests := []struct {
		name       string
		secretType SecretType
		value      string
	}{
		{
			name:       "private key",
			secretType: SecretTypePrivateKey,
			value:      "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
		},
		{
			name:       "access token",
			secretType: SecretTypeAccessToken,
			value:      "ghs_test_access_token",
		},
		{
			name:       "installation token",
			secretType: SecretTypeInstallToken,
			value:      "ghs_test_installation_token",
		},
	}

	keyring.MockInit()
	defer keyring.MockInitWithError(nil)

	tempDir := t.TempDir()
	mgr := NewManager(tempDir)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store secret
			_, err := mgr.Store("test-app", tt.secretType, tt.value)
			if err != nil {
				t.Fatalf("Store() failed: %v", err)
			}

			// Retrieve secret
			value, _, err := mgr.Get("test-app", tt.secretType)
			if err != nil {
				t.Fatalf("Get() failed: %v", err)
			}

			if value != tt.value {
				t.Errorf("Get() value = %v, want %v", value, tt.value)
			}
		})
	}
}

func TestManager_FilesystemPathSanitization(t *testing.T) {
	tempDir := t.TempDir()
	mgr := NewManager(tempDir)

	tests := []struct {
		name       string
		appName    string
		wantSafe   bool
		wantParent string
	}{
		{
			name:       "safe name",
			appName:    "my-app",
			wantSafe:   true,
			wantParent: filepath.Join(tempDir, "secrets"),
		},
		{
			name:       "path traversal attempt",
			appName:    "../../../etc/passwd",
			wantSafe:   true,
			wantParent: filepath.Join(tempDir, "secrets"),
		},
		{
			name:       "absolute path attempt",
			appName:    "/etc/passwd",
			wantSafe:   true,
			wantParent: filepath.Join(tempDir, "secrets"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := mgr.filesystemPath(tt.appName, SecretTypePrivateKey)

			// Check that path is under tempDir/secrets
			parent := filepath.Dir(path)
			if parent != tt.wantParent {
				t.Errorf("Path parent = %v, want %v", parent, tt.wantParent)
			}

			// Check that path doesn't escape
			if !filepath.IsAbs(path) {
				t.Errorf("Path should be absolute: %v", path)
			}

			rel, err := filepath.Rel(tempDir, path)
			if err != nil {
				t.Fatalf("Failed to get relative path: %v", err)
			}

			// Should start with "secrets/"
			if !strings.HasPrefix(rel, "secrets"+string(filepath.Separator)) &&
				rel != "secrets" {
				t.Errorf("Path escaped tempDir: %v", rel)
			}
		})
	}
}

// Package secrets provides secure storage for sensitive data using OS-native keyrings
// with automatic fallback to filesystem storage.
package secrets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zalando/go-keyring"
)

// Common errors returned by the secrets manager
var (
	ErrNotFound           = errors.New("secret not found")
	ErrTimeout            = errors.New("keyring operation timeout")
	ErrStorageUnavailable = errors.New("encrypted storage unavailable")
)

// SecretType identifies the type of secret being stored
type SecretType string

const (
	// SecretTypePrivateKey represents a GitHub App RSA private key
	SecretTypePrivateKey SecretType = "private_key"
	// SecretTypeAccessToken represents a GitHub App access token
	SecretTypeAccessToken SecretType = "access_token"
	// SecretTypeInstallToken represents a GitHub App installation token
	SecretTypeInstallToken SecretType = "installation_token"
	// SecretTypePAT represents a Personal Access Token
	SecretTypePAT SecretType = "pat"
)

// StorageBackend identifies where a secret is stored
type StorageBackend string

const (
	// StorageBackendKeyring indicates the secret is in the OS keyring
	StorageBackendKeyring StorageBackend = "keyring"
	// StorageBackendFilesystem indicates the secret is on the filesystem
	StorageBackendFilesystem StorageBackend = "filesystem"
)

// Manager handles secure storage and retrieval of secrets
type Manager struct {
	keyringTimeout time.Duration
	fallbackDir    string
}

// NewManager creates a new secrets manager with the specified fallback directory
func NewManager(fallbackDir string) *Manager {
	return &Manager{
		keyringTimeout: 3 * time.Second,
		fallbackDir:    fallbackDir,
	}
}

// Store attempts to store a secret in the OS keyring first, falling back to
// filesystem if keyring is unavailable. Returns the storage backend used.
func (m *Manager) Store(appName string, secretType SecretType, value string) (StorageBackend, error) {
	// Try encrypted storage first
	if err := m.storeInKeyring(appName, secretType, value); err == nil {
		// Success! Clean up any filesystem version
		_ = m.deleteFromFilesystem(appName, secretType)
		return StorageBackendKeyring, nil
	}

	// Fallback to filesystem
	if err := m.storeInFilesystem(appName, secretType, value); err != nil {
		return "", fmt.Errorf("failed to store in both keyring and filesystem: %w", err)
	}

	return StorageBackendFilesystem, nil
}

// Get retrieves a secret, trying keyring first then filesystem
func (m *Manager) Get(appName string, secretType SecretType) (string, StorageBackend, error) {
	// Try keyring first
	if value, err := m.getFromKeyring(appName, secretType); err == nil {
		return value, StorageBackendKeyring, nil
	}

	// Try filesystem
	if value, err := m.getFromFilesystem(appName, secretType); err == nil {
		return value, StorageBackendFilesystem, nil
	}

	return "", "", ErrNotFound
}

// Delete removes a secret from both keyring and filesystem
func (m *Manager) Delete(appName string, secretType SecretType) error {
	keyringErr := m.deleteFromKeyring(appName, secretType)
	filesystemErr := m.deleteFromFilesystem(appName, secretType)

	// If both fail, return the first error
	if keyringErr != nil && filesystemErr != nil {
		return keyringErr
	}

	// If either succeeds, consider it a success
	return nil
}

// IsAvailable checks if encrypted keyring storage is available
func (m *Manager) IsAvailable() bool {
	// Test keyring availability with a quick operation
	testService := "gh-app-auth:availability-test"
	testUser := "test"
	testValue := "test"

	ch := make(chan bool, 1)
	go func() {
		defer close(ch)
		if err := keyring.Set(testService, testUser, testValue); err != nil {
			ch <- false
			return
		}
		_ = keyring.Delete(testService, testUser)
		ch <- true
	}()

	select {
	case available := <-ch:
		return available
	case <-time.After(m.keyringTimeout):
		return false
	}
}

// storeInKeyring stores a secret in the OS keyring with timeout protection
func (m *Manager) storeInKeyring(appName string, secretType SecretType, value string) error {
	service := m.keyringService(appName)
	user := string(secretType)

	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		ch <- keyring.Set(service, user, value)
	}()

	select {
	case err := <-ch:
		return err
	case <-time.After(m.keyringTimeout):
		return ErrTimeout
	}
}

// getFromKeyring retrieves a secret from the OS keyring with timeout protection
func (m *Manager) getFromKeyring(appName string, secretType SecretType) (string, error) {
	service := m.keyringService(appName)
	user := string(secretType)

	ch := make(chan struct {
		val string
		err error
	}, 1)

	go func() {
		defer close(ch)
		val, err := keyring.Get(service, user)
		ch <- struct {
			val string
			err error
		}{val, err}
	}()

	select {
	case res := <-ch:
		if errors.Is(res.err, keyring.ErrNotFound) {
			return "", ErrNotFound
		}
		return res.val, res.err
	case <-time.After(m.keyringTimeout):
		return "", ErrTimeout
	}
}

// deleteFromKeyring removes a secret from the OS keyring with timeout protection
func (m *Manager) deleteFromKeyring(appName string, secretType SecretType) error {
	service := m.keyringService(appName)
	user := string(secretType)

	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		ch <- keyring.Delete(service, user)
	}()

	select {
	case err := <-ch:
		if errors.Is(err, keyring.ErrNotFound) {
			return ErrNotFound
		}
		return err
	case <-time.After(m.keyringTimeout):
		return ErrTimeout
	}
}

// storeInFilesystem stores a secret on the filesystem with secure permissions
func (m *Manager) storeInFilesystem(appName string, secretType SecretType, value string) error {
	path := m.filesystemPath(appName, secretType)

	// Ensure directory exists with secure permissions
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create secrets directory: %w", err)
	}

	// Write with secure permissions (owner read-only)
	if err := os.WriteFile(path, []byte(value), 0400); err != nil {
		return fmt.Errorf("failed to write secret file: %w", err)
	}

	return nil
}

// getFromFilesystem retrieves a secret from the filesystem
func (m *Manager) getFromFilesystem(appName string, secretType SecretType) (string, error) {
	path := m.filesystemPath(appName, secretType)

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to read secret file: %w", err)
	}

	return string(data), nil
}

// deleteFromFilesystem removes a secret from the filesystem
func (m *Manager) deleteFromFilesystem(appName string, secretType SecretType) error {
	path := m.filesystemPath(appName, secretType)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return ErrNotFound
	}
	return err
}

// keyringService returns the keyring service name for an app
func (m *Manager) keyringService(appName string) string {
	return fmt.Sprintf("gh-app-auth:%s", appName)
}

// filesystemPath returns the filesystem path for a secret
func (m *Manager) filesystemPath(appName string, secretType SecretType) string {
	// Sanitize app name to be filesystem-safe
	safeAppName := filepath.Base(appName)
	filename := fmt.Sprintf("%s.%s", safeAppName, secretType)
	return filepath.Join(m.fallbackDir, "secrets", filename)
}

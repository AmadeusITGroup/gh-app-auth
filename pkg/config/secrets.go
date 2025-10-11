package config

import (
	"fmt"
	"os"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/secrets"
)

// GetPrivateKey retrieves the private key from the appropriate source
// based on the PrivateKeySource configuration
func (app *GitHubApp) GetPrivateKey(secretMgr *secrets.Manager) (string, error) {
	switch app.PrivateKeySource {
	case PrivateKeySourceKeyring:
		key, _, err := secretMgr.Get(app.Name, secrets.SecretTypePrivateKey)
		if err == nil {
			return key, nil
		}
		// If keyring fails and we have a path, try filesystem as fallback
		if app.PrivateKeyPath != "" {
			return app.getPrivateKeyFromFilesystem()
		}
		return "", fmt.Errorf("failed to get private key from keyring: %w", err)

	case PrivateKeySourceFilesystem:
		return app.getPrivateKeyFromFilesystem()

	case PrivateKeySourceInline:
		return "", fmt.Errorf("inline private keys should be migrated to secure storage")

	default:
		return "", fmt.Errorf("unknown private key source: %s", app.PrivateKeySource)
	}
}

// getPrivateKeyFromFilesystem reads the private key from the filesystem
func (app *GitHubApp) getPrivateKeyFromFilesystem() (string, error) {
	if app.PrivateKeyPath == "" {
		return "", fmt.Errorf("private_key_path not set for filesystem source")
	}

	expandedPath, err := expandPath(app.PrivateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to expand path: %w", err)
	}

	keyData, err := os.ReadFile(expandedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key: %w", err)
	}

	return string(keyData), nil
}

// SetPrivateKey stores the private key securely
// It attempts to use keyring first, falling back to filesystem if unavailable
func (app *GitHubApp) SetPrivateKey(secretMgr *secrets.Manager, privateKey string) (secrets.StorageBackend, error) {
	backend, err := secretMgr.Store(app.Name, secrets.SecretTypePrivateKey, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to store private key: %w", err)
	}

	// Update the app's configuration based on storage backend used
	if backend == secrets.StorageBackendKeyring {
		app.PrivateKeySource = PrivateKeySourceKeyring
		// Clear path since key is in keyring (but keep it as fallback if already set)
		// app.PrivateKeyPath = ""
	} else {
		app.PrivateKeySource = PrivateKeySourceFilesystem
		// Filesystem backend has stored the key, path is managed by secrets manager
	}

	return backend, nil
}

// DeletePrivateKey removes the private key from secure storage
func (app *GitHubApp) DeletePrivateKey(secretMgr *secrets.Manager) error {
	return secretMgr.Delete(app.Name, secrets.SecretTypePrivateKey)
}

// HasPrivateKey checks if the app has a private key configured
func (app *GitHubApp) HasPrivateKey(secretMgr *secrets.Manager) bool {
	switch app.PrivateKeySource {
	case PrivateKeySourceKeyring:
		_, _, err := secretMgr.Get(app.Name, secrets.SecretTypePrivateKey)
		return err == nil
	case PrivateKeySourceFilesystem:
		if app.PrivateKeyPath == "" {
			return false
		}
		expandedPath, err := expandPath(app.PrivateKeyPath)
		if err != nil {
			return false
		}
		_, err = os.Stat(expandedPath)
		return err == nil
	default:
		return false
	}
}

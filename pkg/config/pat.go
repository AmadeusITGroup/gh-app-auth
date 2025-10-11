package config

import (
	"fmt"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/secrets"
)

// GetPAT retrieves the Personal Access Token from the appropriate source
func (p *PersonalAccessToken) GetPAT(secretMgr *secrets.Manager) (string, error) {
	switch p.TokenSource {
	case PrivateKeySourceKeyring, "": // Empty defaults to keyring
		token, _, err := secretMgr.Get(p.Name, secrets.SecretTypePAT)
		if err == nil {
			return token, nil
		}
		return "", fmt.Errorf("failed to get PAT from keyring: %w", err)

	case PrivateKeySourceFilesystem:
		return "", fmt.Errorf("filesystem storage for PATs is not yet implemented")

	default:
		return "", fmt.Errorf("unknown token source: %s", p.TokenSource)
	}
}

// SetPAT stores the Personal Access Token securely
func (p *PersonalAccessToken) SetPAT(secretMgr *secrets.Manager, token string) (secrets.StorageBackend, error) {
	backend, err := secretMgr.Store(p.Name, secrets.SecretTypePAT, token)
	if err != nil {
		return "", fmt.Errorf("failed to store PAT: %w", err)
	}

	// Update the PAT's configuration based on storage backend used
	if backend == secrets.StorageBackendKeyring {
		p.TokenSource = PrivateKeySourceKeyring
	} else {
		p.TokenSource = PrivateKeySourceFilesystem
	}

	return backend, nil
}

// DeletePAT removes the PAT from secure storage
func (p *PersonalAccessToken) DeletePAT(secretMgr *secrets.Manager) error {
	return secretMgr.Delete(p.Name, secrets.SecretTypePAT)
}

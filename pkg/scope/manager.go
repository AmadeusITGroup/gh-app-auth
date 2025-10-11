package scope

import (
	"fmt"
	"time"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"github.com/cli/go-gh/v2/pkg/api"
)

// Manager handles installation scope detection and caching
type Manager struct {
	clientFactory func(api.ClientOptions) (*api.RESTClient, error)
}

// NewManager creates a new scope manager
func NewManager() *Manager {
	return &Manager{
		clientFactory: api.NewRESTClient,
	}
}

// FetchScope retrieves and caches installation scope information
func (m *Manager) FetchScope(app *config.GitHubApp, jwtToken string) error {
	// Create API client with JWT
	client, err := m.clientFactory(api.ClientOptions{
		Headers: map[string]string{
			"Authorization": "Bearer " + jwtToken,
			"Accept":        "application/vnd.github+json",
		},
		Host: "api.github.com",
	})
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Fetch installation details
	installation, err := m.getInstallation(client, app.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to get installation: %w", err)
	}

	// Build scope object
	scope := &config.InstallationScope{
		RepositorySelection: installation.RepositorySelection,
		AccountLogin:        installation.Account.Login,
		AccountType:         installation.Account.Type,
		LastFetched:         time.Now(),
		LastUpdated:         installation.UpdatedAt,
		CacheExpiry:         time.Now().Add(24 * time.Hour), // 24-hour cache
	}

	// If "selected", fetch repository list
	if installation.RepositorySelection == "selected" {
		repos, err := m.getRepositories(app.InstallationID, jwtToken)
		if err != nil {
			return fmt.Errorf("failed to get repositories: %w", err)
		}
		scope.Repositories = repos
	}

	app.Scope = scope
	return nil
}

// getInstallation fetches installation metadata
func (m *Manager) getInstallation(client *api.RESTClient, installationID int64) (*InstallationResponse, error) {
	var installation InstallationResponse
	err := client.Get(fmt.Sprintf("app/installations/%d", installationID), &installation)
	if err != nil {
		return nil, err
	}
	return &installation, nil
}

// getRepositories fetches repository list for "selected" installations
func (m *Manager) getRepositories(installationID int64, jwtToken string) ([]config.RepositoryInfo, error) {
	// This requires an installation access token, not JWT
	// We need to generate one first
	installToken, err := m.getInstallationToken(jwtToken, installationID)
	if err != nil {
		return nil, err
	}

	// Create client with installation token
	client, err := m.clientFactory(api.ClientOptions{
		Headers: map[string]string{
			"Authorization": "token " + installToken,
			"Accept":        "application/vnd.github+json",
		},
		Host: "api.github.com",
	})
	if err != nil {
		return nil, err
	}

	// Fetch repositories with pagination
	var allRepos []config.RepositoryInfo
	page := 1
	perPage := 100

	for {
		var response RepositoriesResponse
		err := client.Get(
			fmt.Sprintf("installation/repositories?per_page=%d&page=%d", perPage, page),
			&response,
		)
		if err != nil {
			return nil, err
		}

		for _, repo := range response.Repositories {
			allRepos = append(allRepos, config.RepositoryInfo{
				FullName: repo.FullName,
				Private:  repo.Private,
			})
		}

		// Check if we've fetched all repos
		if len(response.Repositories) < perPage {
			break
		}
		page++
	}

	return allRepos, nil
}

// getInstallationToken exchanges JWT for installation access token
func (m *Manager) getInstallationToken(jwtToken string, installationID int64) (string, error) {
	client, err := m.clientFactory(api.ClientOptions{
		Headers: map[string]string{
			"Authorization": "Bearer " + jwtToken,
			"Accept":        "application/vnd.github+json",
		},
		Host: "api.github.com",
	})
	if err != nil {
		return "", err
	}

	var response struct {
		Token string `json:"token"`
	}

	err = client.Post(
		fmt.Sprintf("app/installations/%d/access_tokens", installationID),
		nil,
		&response,
	)
	if err != nil {
		return "", err
	}

	return response.Token, nil
}

// NeedsRefresh checks if scope cache needs refreshing
func (m *Manager) NeedsRefresh(app *config.GitHubApp) bool {
	if app.Scope == nil {
		return true
	}
	return time.Now().After(app.Scope.CacheExpiry)
}

// API Response types
type InstallationResponse struct {
	ID                  int64     `json:"id"`
	Account             Account   `json:"account"`
	RepositorySelection string    `json:"repository_selection"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Account struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

type RepositoriesResponse struct {
	TotalCount   int          `json:"total_count"`
	Repositories []Repository `json:"repositories"`
}

type Repository struct {
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
}

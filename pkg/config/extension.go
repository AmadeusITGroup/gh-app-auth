package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadOrCreate loads existing configuration or creates a new one
func LoadOrCreate() (*Config, error) {
	cfg, err := Load()
	if err != nil {
		// Check if error is due to file not existing
		if os.IsNotExist(err) || strings.Contains(err.Error(), "configuration file not found") {
			// Create new config
			return &Config{
				Version:    "1.0",
				GitHubApps: []GitHubApp{},
			}, nil
		}
		// Check if error is due to empty apps (valid during setup)
		if errors.Is(err, ErrNoGitHubAppDefined) {
			// Load the file without validation
			loader := NewDefaultLoader()
			data, readErr := os.ReadFile(loader.GetConfigPath())
			if readErr != nil {
				return nil, fmt.Errorf("failed to read configuration file: %w", readErr)
			}
			// Parse without validation
			var config Config
			if parseErr := yaml.Unmarshal(data, &config); parseErr != nil {
				return nil, fmt.Errorf("failed to parse configuration: %w", parseErr)
			}
			// Return the empty config (valid during setup)
			return &config, nil
		}
		return nil, err
	}
	return cfg, nil
}

// Load loads configuration using the default loader
func Load() (*Config, error) {
	loader := NewDefaultLoader()
	return loader.Load()
}

// Save saves the configuration to the default location
func (c *Config) Save() error {
	configPath := getDefaultConfigPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save as YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AddOrUpdateApp adds a new app or updates an existing one
func (c *Config) AddOrUpdateApp(app *GitHubApp) {
	// Check if app already exists
	for i, existingApp := range c.GitHubApps {
		// Update existing app
		if existingApp.AppID == app.AppID && existingApp.InstallationID == app.InstallationID {
			// Merge patterns from new app into existing app
			mergedPatterns := mergePatterns(existingApp.Patterns, app.Patterns)
			app.Patterns = mergedPatterns
			// Update existing app with merged patterns
			c.GitHubApps[i] = *app
			return
		}
	}

	// Add new app
	c.GitHubApps = append(c.GitHubApps, *app)
}

func mergePatterns(existingPatterns, newPatterns []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(existingPatterns)+len(newPatterns))

	// Add existing patterns first
	for _, p := range existingPatterns {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}

	// Add new patterns if not already present
	for _, p := range newPatterns {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}

	return result
}

// AddOrUpdatePAT adds a new PAT or updates an existing one
func (c *Config) AddOrUpdatePAT(pat *PersonalAccessToken) {
	// Check if PAT already exists (by name)
	for i, existingPAT := range c.PATs {
		if existingPAT.Name == pat.Name {
			// Update existing PAT
			c.PATs[i] = *pat
			return
		}
	}

	// Add new PAT
	c.PATs = append(c.PATs, *pat)
}

// RemoveApp removes an app by ID
func (c *Config) RemoveApp(appID int64) bool {
	for i, app := range c.GitHubApps {
		if app.AppID == appID {
			c.GitHubApps = append(c.GitHubApps[:i], c.GitHubApps[i+1:]...)
			return true
		}
	}
	return false
}

// RemovePAT removes a PAT by name
func (c *Config) RemovePAT(name string) bool {
	for i, pat := range c.PATs {
		if pat.Name == name {
			c.PATs = append(c.PATs[:i], c.PATs[i+1:]...)
			return true
		}
	}
	return false
}

// GetApp finds an app by ID
func (c *Config) GetApp(appID int64) (*GitHubApp, error) {
	for _, app := range c.GitHubApps {
		if app.AppID == appID {
			return &app, nil
		}
	}
	return nil, fmt.Errorf("app with ID %d not found", appID)
}

// OutputJSON outputs apps as JSON
func OutputJSON(w io.Writer, apps []GitHubApp) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(apps)
}

// OutputYAML outputs apps as YAML
func OutputYAML(w io.Writer, apps []GitHubApp) error {
	encoder := yaml.NewEncoder(w)
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(apps)
}

// getDefaultConfigPath returns the default config path for the extension
func getDefaultConfigPath() string {
	// Check environment variable first
	if path := os.Getenv("GH_APP_AUTH_CONFIG"); path != "" {
		if expanded, err := expandPath(path); err == nil {
			return expanded
		}
		return path
	}

	// Use GitHub CLI extension config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth", "config.yml")
}

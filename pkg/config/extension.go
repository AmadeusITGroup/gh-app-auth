package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadOrCreate loads existing configuration or creates a new one
func LoadOrCreate() (*Config, error) {
	cfg, err := Load()
	if os.IsNotExist(err) {
		// Create new config
		return &Config{
			Version:    "1.0",
			GitHubApps: []GitHubApp{},
		}, nil
	}
	return cfg, err
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
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
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
func (c *Config) AddOrUpdateApp(app GitHubApp) {
	// Check if app already exists
	for i, existingApp := range c.GitHubApps {
		if existingApp.AppID == app.AppID {
			// Update existing app
			c.GitHubApps[i] = app
			return
		}
	}
	
	// Add new app
	c.GitHubApps = append(c.GitHubApps, app)
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
	defer encoder.Close()
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

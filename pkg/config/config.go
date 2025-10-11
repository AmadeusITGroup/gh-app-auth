package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the GitHub App authentication configuration
type Config struct {
	Version    string      `yaml:"version" json:"version"`
	GitHubApps []GitHubApp `yaml:"github_apps" json:"github_apps"`
}

// GitHubApp represents a single GitHub App configuration
type GitHubApp struct {
	Name            string   `yaml:"name" json:"name"`
	AppID           int64    `yaml:"app_id" json:"app_id"`
	InstallationID  int64    `yaml:"installation_id" json:"installation_id"`
	PrivateKeyPath  string   `yaml:"private_key_path" json:"private_key_path"`
	Patterns        []string `yaml:"patterns" json:"patterns"`
	Priority        int      `yaml:"priority" json:"priority"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}

	if len(c.GitHubApps) == 0 {
		return fmt.Errorf("at least one github_app is required")
	}

	for i, app := range c.GitHubApps {
		if err := app.Validate(); err != nil {
			return fmt.Errorf("github_apps[%d]: %w", i, err)
		}
	}

	return nil
}

// Validate validates a single GitHub App configuration
func (g *GitHubApp) Validate() error {
	if g.Name == "" {
		return fmt.Errorf("name is required")
	}

	if g.AppID <= 0 {
		return fmt.Errorf("app_id must be positive")
	}

	if g.InstallationID <= 0 {
		return fmt.Errorf("installation_id must be positive")
	}

	if g.PrivateKeyPath == "" {
		return fmt.Errorf("private_key_path is required")
	}

	// Expand tilde in private key path
	expandedPath, err := expandPath(g.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("invalid private_key_path: %w", err)
	}
	g.PrivateKeyPath = expandedPath

	if len(g.Patterns) == 0 {
		return fmt.Errorf("at least one pattern is required")
	}

	// Validate patterns are not empty
	for i, pattern := range g.Patterns {
		if strings.TrimSpace(pattern) == "" {
			return fmt.Errorf("patterns[%d] cannot be empty", i)
		}
	}
	return nil
}

// expandPath expands ~ to home directory in file paths
func expandPath(path string) (string, error) {
	var expandedPath string
	var err error
	
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to get home directory: %w", err)
		}

		if path == "~" {
			expandedPath = homeDir
		} else if strings.HasPrefix(path, "~/") {
			expandedPath = filepath.Join(homeDir, path[2:])
		} else {
			return "", fmt.Errorf("invalid path format: %s", path)
		}
	} else {
		expandedPath, err = filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}
	}
	
	// Validate path doesn't contain suspicious elements that could lead to path traversal
	cleanPath := filepath.Clean(expandedPath)
	if cleanPath != expandedPath {
		return "", fmt.Errorf("path contains suspicious elements: %s", path)
	}
	
	return cleanPath, nil
}

// GetByPriority returns GitHub Apps sorted by priority (highest first)
func (c *Config) GetByPriority() []GitHubApp {
	apps := make([]GitHubApp, len(c.GitHubApps))
	copy(apps, c.GitHubApps)

	// Sort by priority (highest first), then by name for stable ordering
	for i := 0; i < len(apps)-1; i++ {
		for j := i + 1; j < len(apps); j++ {
			if apps[i].Priority < apps[j].Priority ||
				(apps[i].Priority == apps[j].Priority && apps[i].Name > apps[j].Name) {
				apps[i], apps[j] = apps[j], apps[i]
			}
		}
	}

	return apps
}

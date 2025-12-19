package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Common errors returned by config
var (
	ErrNoGitHubAppDefined = errors.New("at least one github_app or pat is required")
)

// CurrentConfigVersion is the latest configuration schema version
const CurrentConfigVersion = "1"

// Config represents the GitHub App authentication configuration
type Config struct {
	Version    string                `yaml:"version" json:"version"`
	GitHubApps []GitHubApp           `yaml:"github_apps" json:"github_apps"`
	PATs       []PersonalAccessToken `yaml:"pats,omitempty" json:"pats,omitempty"`
}

// PrivateKeySource indicates where the private key is stored
type PrivateKeySource string

const (
	// PrivateKeySourceKeyring indicates the key is in the OS keyring
	PrivateKeySourceKeyring PrivateKeySource = "keyring"
	// PrivateKeySourceFilesystem indicates the key is in a file
	PrivateKeySourceFilesystem PrivateKeySource = "filesystem"
	// PrivateKeySourceInline indicates the key was provided inline (legacy)
	PrivateKeySourceInline PrivateKeySource = "inline"
)

// InstallationScope represents cached GitHub App installation scope information
type InstallationScope struct {
	// Core scope information
	RepositorySelection string `yaml:"repository_selection" json:"repository_selection"` // "all" or "selected"
	AccountLogin        string `yaml:"account_login" json:"account_login"`               // org or user name
	AccountType         string `yaml:"account_type" json:"account_type"`                 // "Organization" or "User"

	// Repository list (only populated if repository_selection == "selected")
	Repositories []RepositoryInfo `yaml:"repositories,omitempty" json:"repositories,omitempty"`

	// Cache metadata
	LastFetched time.Time `yaml:"last_fetched" json:"last_fetched"`
	LastUpdated time.Time `yaml:"last_updated" json:"last_updated"` // From GitHub API
	CacheExpiry time.Time `yaml:"cache_expiry" json:"cache_expiry"`
}

// RepositoryInfo represents a cached repository
type RepositoryInfo struct {
	FullName string `yaml:"full_name" json:"full_name"` // "owner/repo"
	Private  bool   `yaml:"private" json:"private"`
}

// GitHubApp represents a single GitHub App configuration
type GitHubApp struct {
	Name             string             `yaml:"name" json:"name"`
	AppID            int64              `yaml:"app_id" json:"app_id"`
	InstallationID   int64              `yaml:"installation_id" json:"installation_id"`
	PrivateKeyPath   string             `yaml:"private_key_path,omitempty" json:"private_key_path,omitempty"`
	PrivateKeySource PrivateKeySource   `yaml:"private_key_source,omitempty" json:"private_key_source,omitempty"`
	Patterns         []string           `yaml:"patterns" json:"patterns"`
	Priority         int                `yaml:"priority" json:"priority"` // Deprecated: Ignored in favor of longest prefix
	Scope            *InstallationScope `yaml:"scope,omitempty" json:"scope,omitempty"`
}

type PersonalAccessToken struct {
	Name        string           `yaml:"name" json:"name"`
	TokenSource PrivateKeySource `yaml:"private_key_source,omitempty" json:"private_key_source,omitempty"`
	Patterns    []string         `yaml:"patterns" json:"patterns"`
	Priority    int              `yaml:"priority" json:"priority"`
	// Username for HTTP basic auth (optional, defaults to "x-access-token" for GitHub)
	Username string `yaml:"username,omitempty" json:"username,omitempty"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}

	if len(c.GitHubApps) == 0 && len(c.PATs) == 0 {
		return ErrNoGitHubAppDefined
	}

	for i, app := range c.GitHubApps {
		if err := app.Validate(); err != nil {
			return fmt.Errorf("github_apps[%d]: %w", i, err)
		}
	}

	for i, pat := range c.PATs {
		if err := pat.Validate(); err != nil {
			return fmt.Errorf("pats[%d]: %w", i, err)
		}
	}

	return nil
}

// Validate validates a single GitHub App configuration
func (g *GitHubApp) Validate() error {
	// Validate basic fields
	if err := g.validateBasicFields(); err != nil {
		return err
	}

	// Validate private key configuration
	if err := g.validatePrivateKeyConfig(); err != nil {
		return err
	}

	// Validate patterns
	return g.validatePatterns()
}

// expandPath expands ~ to home directory in file paths
func expandPath(path string) (string, error) {
	var expandedPath string
	var err error

	// Handle home directory expansion
	if strings.HasPrefix(path, "~") {
		expandedPath, err = expandHomeDirectory(path)
		if err != nil {
			return "", err
		}
	} else {
		expandedPath, err = filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}
	}

	return expandedPath, nil
}

// expandHomeDirectory expands tilde in path to home directory
func expandHomeDirectory(path string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to get home directory: %w", err)
	}

	if path == "~" {
		return homeDir, nil
	} else if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:]), nil
	}
	return "", fmt.Errorf("invalid path format: %s", path)
}

// GetByPriority returns GitHub Apps sorted by priority (highest first)
func (c *Config) GetByPriority() []GitHubApp {
	apps := make([]GitHubApp, len(c.GitHubApps))
	copy(apps, c.GitHubApps)

	sort.Slice(apps, func(i, j int) bool {
		if apps[i].Priority != apps[j].Priority {
			return apps[i].Priority > apps[j].Priority
		}
		return apps[i].Name < apps[j].Name
	})

	return apps
}

// validateBasicFields validates the basic required fields of a GitHubApp
func (g *GitHubApp) validateBasicFields() error {
	if g.Name == "" {
		return fmt.Errorf("name is required")
	}

	if g.AppID <= 0 {
		return fmt.Errorf("app_id must be positive")
	}

	// InstallationID can be 0 (auto-detected at runtime via GitHub API)
	if g.InstallationID < 0 {
		return fmt.Errorf("installation_id cannot be negative")
	}

	return nil
}

// validatePrivateKeyConfig validates the private key configuration
func (g *GitHubApp) validatePrivateKeyConfig() error {
	// Handle legacy config without source specified
	if g.PrivateKeySource == "" {
		if g.PrivateKeyPath == "" {
			return fmt.Errorf("private_key_path or private_key_source is required")
		}
		g.PrivateKeySource = PrivateKeySourceFilesystem
	}

	// Validate based on source type
	switch g.PrivateKeySource {
	case PrivateKeySourceFilesystem:
		return g.validateFilesystemKeyConfig()
	case PrivateKeySourceKeyring:
		// Key is in keyring, path not needed
		return nil
	case PrivateKeySourceInline:
		return fmt.Errorf("inline private keys must be migrated to keyring or filesystem")
	default:
		return fmt.Errorf("invalid private_key_source: %s", g.PrivateKeySource)
	}
}

// validateFilesystemKeyConfig validates filesystem-based private key configuration
func (g *GitHubApp) validateFilesystemKeyConfig() error {
	if g.PrivateKeyPath == "" {
		return fmt.Errorf("private_key_path is required when using filesystem source")
	}

	// Expand tilde in private key path
	expandedPath, err := expandPath(g.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("invalid private_key_path: %w", err)
	}
	g.PrivateKeyPath = expandedPath
	return nil
}

// validatePatterns validates the repository patterns
func (g *GitHubApp) validatePatterns() error {
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

func (p *PersonalAccessToken) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("name is required")
	}

	if len(p.Patterns) == 0 {
		return fmt.Errorf("at least one pattern is required")
	}

	for i, pattern := range p.Patterns {
		if strings.TrimSpace(pattern) == "" {
			return fmt.Errorf("patterns[%d] cannot be empty", i)
		}
	}

	if p.TokenSource != "" && p.TokenSource != PrivateKeySourceKeyring && p.TokenSource != PrivateKeySourceFilesystem {
		return fmt.Errorf("invalid private_key_source: %s", p.TokenSource)
	}

	return nil
}

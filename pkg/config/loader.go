package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader handles loading GitHub App configurations from files
type Loader struct {
	configPath string
}

// Common errors returned by the loader
var (
	ErrConfigNotExists  = errors.New("configuration file not found: ")
	ErrConfigInvalid    = errors.New("invalid configuration: ")
	ErrConfigUnreadable = errors.New("failed to read configuration file: ")
	ErrConfigUnparsable = errors.New("failed to parse configuration: ")
)

// NewLoader creates a new configuration loader
func NewLoader(configPath string) *Loader {
	return &Loader{
		configPath: configPath,
	}
}

// NewDefaultLoader creates a loader with the default configuration path
func NewDefaultLoader() *Loader {
	defaultPath := getDefaultConfigPath()
	return NewLoader(defaultPath)
}

// Load loads the configuration from the configured path
func (l *Loader) Load() (*Config, error) {
	if l.configPath == "" {
		return nil, fmt.Errorf("no configuration path specified")
	}

	// Check if file exists
	if _, err := os.Stat(l.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s: %w", l.configPath, ErrConfigNotExists)
	}

	// Note: We don't check file extension here, parseConfig will handle different formats

	// Read file content
	data, err := os.ReadFile(l.configPath)
	if err != nil {
		return nil, fmt.Errorf("%w", ErrConfigUnreadable)
	}

	// Parse based on file extension
	config, err := l.parseConfig(data, l.configPath)
	if err != nil {
		return nil, fmt.Errorf("%w", ErrConfigUnparsable)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	return config, nil
}

// LoadWithFallback loads configuration, returns nil if file doesn't exist (no error)
func (l *Loader) LoadWithFallback() (*Config, error) {
	config, err := l.Load()
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "configuration file not found") {
			return nil, nil // No config file is not an error
		}
		return nil, err
	}
	return config, nil
}

// parseConfig parses configuration data based on file extension
func (l *Loader) parseConfig(data []byte, filePath string) (*Config, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	var config Config
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, &config); err != nil {
			if jsonErr := json.Unmarshal(data, &config); jsonErr != nil {
				return nil, fmt.Errorf("failed to parse as YAML or JSON (YAML error: %v, JSON error: %w)", err, jsonErr)
			}
		}
	}

	return &config, nil
}

// ConfigExists checks if a configuration file exists at the given path
func ConfigExists(configPath string) bool {
	if configPath == "" {
		return false
	}
	_, err := os.Stat(configPath)
	return err == nil
}

// GetConfigPath returns the configuration path that would be used
func (l *Loader) GetConfigPath() string {
	return l.configPath
}

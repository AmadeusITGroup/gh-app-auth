package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	t.Run("fileExists of unexisting file", func(t *testing.T) {
		outcome := fileExists("/some/unknown/path")

		if outcome {
			t.Error("/some/path shouldn't exist")
		}
	})

	t.Run("fileExists of existing file", func(t *testing.T) {
		file, err := os.CreateTemp("", "some_file")
		if err != nil {
			t.Error("Failed to create temporary file")
		}

		defer os.Remove(file.Name())
		outcome := fileExists(file.Name())

		if !outcome {
			t.Error(file.Name() + "should exist")
		}
	})
}

func getDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("Failed to get home directory")
	}
	return filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth", "config.yml")
}

func TestGetConfigPath(t *testing.T) {
	t.Run("getConfigPath of default config", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Error("Failed to get home directory")
		}
		expected_path := filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth", "config.yml")
		config_path := getConfigPath()
		if config_path != expected_path {
			t.Error("getConfigPath() returned " + config_path + " instead of " + expected_path)
		}
	})

	t.Run("getConfigPath when GH_APP_AUTH_CONFIG is set but empty", func(t *testing.T) {
		os.Setenv("GH_APP_AUTH_CONFIG", "")
		config_path := getConfigPath()
		if config_path != getDefaultConfigPath() {
			t.Error("getConfigPath() returned " + config_path + " instead of " + getDefaultConfigPath())
		}
	})

	t.Run("getConfigPath when GH_APP_AUTH_CONFIG is set to real path", func(t *testing.T) {
		os.Setenv("GH_APP_AUTH_CONFIG", "/some/path/config.yml")
		config_path := getConfigPath()
		if config_path != "/some/path/config.yml" {
			t.Error("getConfigPath() returned " + config_path + " instead of /some/path/config.yml")
		}
	})
}

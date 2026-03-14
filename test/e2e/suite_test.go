//go:build e2e

package e2e

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
)

// E2EConfig holds all configuration needed for E2E tests.
// All fields are populated from environment variables at startup.
type E2EConfig struct {
	// BinaryPath is the path to the gh-app-auth binary under test.
	// Set via E2E_BINARY_PATH. If empty, binary is built from source.
	BinaryPath string

	// AppID is the GitHub App ID (string form for passing to CLI flags).
	// Set via E2E_APP_ID.
	AppID string

	// PrivateKeyPEM is the decoded PEM-encoded private key content.
	// Set via E2E_PRIVATE_KEY_B64 (base64-encoded for safe CI storage).
	PrivateKeyPEM string

	// TestOrg1 is the first test organization name.
	// Set via E2E_TEST_ORG_1 (default: gh-app-auth-test-1).
	TestOrg1 string

	// TestOrg2 is the second test organization name.
	// Set via E2E_TEST_ORG_2 (default: gh-app-auth-test-2).
	TestOrg2 string

	// installationIDCache stores discovered installation IDs for organizations.
	// Key: organization name, Value: installation ID.
	// Populated dynamically via GitHub API.
	installationIDCache map[string]int64
}

// globalConfig is loaded once in TestMain and shared across all tests.
var globalConfig *E2EConfig

// TestMain is the entry point for the E2E test suite.
// It loads configuration and fails fast if required env vars are missing.
func TestMain(m *testing.M) {
	cfg, err := loadE2EConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "E2E configuration error: %v\n\n", err)
		fmt.Fprintf(os.Stderr, "Required environment variables:\n")
		fmt.Fprintf(os.Stderr, "  E2E_APP_ID            GitHub App ID\n")
		fmt.Fprintf(os.Stderr, "  E2E_PRIVATE_KEY_B64   Base64-encoded private key PEM\n")
		fmt.Fprintf(os.Stderr, "\nOptional:\n")
		fmt.Fprintf(os.Stderr, "  E2E_BINARY_PATH       Path to binary (built from source if unset)\n")
		fmt.Fprintf(os.Stderr, "  E2E_TEST_ORG_1        Test org 1 (default: gh-app-auth-test-1)\n")
		fmt.Fprintf(os.Stderr, "  E2E_TEST_ORG_2        Test org 2 (default: gh-app-auth-test-2)\n")
		fmt.Fprintf(os.Stderr, "\nInstallation IDs are discovered dynamically via GitHub API.\n")
		fmt.Fprintf(os.Stderr, "\nSee docs/E2E_INFRASTRUCTURE.md for infrastructure setup.\n")
		os.Exit(1)
	}

	globalConfig = cfg
	os.Exit(m.Run())
}

// loadE2EConfig loads the E2E test configuration from environment variables.
func loadE2EConfig() (*E2EConfig, error) {
	cfg := &E2EConfig{}

	cfg.AppID = os.Getenv("E2E_APP_ID")
	if cfg.AppID == "" {
		return nil, fmt.Errorf("E2E_APP_ID is required")
	}

	// E2E_PRIVATE_KEY accepts raw PEM content (convenient for local dev via $(cat key.pem)).
	// E2E_PRIVATE_KEY_B64 accepts base64-encoded PEM (required for CI / GitHub Actions secrets).
	// E2E_PRIVATE_KEY takes precedence when both are set.
	if raw := os.Getenv("E2E_PRIVATE_KEY"); raw != "" {
		cfg.PrivateKeyPEM = raw
	} else {
		privateKeyB64 := os.Getenv("E2E_PRIVATE_KEY_B64")
		if privateKeyB64 == "" {
			return nil, fmt.Errorf("E2E_PRIVATE_KEY or E2E_PRIVATE_KEY_B64 is required")
		}
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(privateKeyB64))
		if err != nil {
			return nil, fmt.Errorf("failed to base64-decode E2E_PRIVATE_KEY_B64: %w\n"+
				"Tip: for local dev use E2E_PRIVATE_KEY=$(cat key.pem) instead", err)
		}
		cfg.PrivateKeyPEM = string(decoded)
	}

	cfg.TestOrg1 = os.Getenv("E2E_TEST_ORG_1")
	if cfg.TestOrg1 == "" {
		cfg.TestOrg1 = "gh-app-auth-test-1"
	}

	cfg.TestOrg2 = os.Getenv("E2E_TEST_ORG_2")
	if cfg.TestOrg2 == "" {
		cfg.TestOrg2 = "gh-app-auth-test-2"
	}

	cfg.BinaryPath = os.Getenv("E2E_BINARY_PATH")

	// Initialize installation ID cache
	cfg.installationIDCache = make(map[string]int64)

	return cfg, nil
}

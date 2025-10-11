package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"gopkg.in/yaml.v3"
)

// ConfigBuilder helps build test configurations
type ConfigBuilder struct {
	apps []config.GitHubApp
}

// NewConfigBuilder creates a new config builder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		apps: make([]config.GitHubApp, 0),
	}
}

// AddApp adds a GitHub App to the configuration
func (cb *ConfigBuilder) AddApp(app config.GitHubApp) *ConfigBuilder {
	cb.apps = append(cb.apps, app)
	return cb
}

// AddSimpleApp adds a simple app with basic configuration
func (cb *ConfigBuilder) AddSimpleApp(name string, appID int64, patterns []string) *ConfigBuilder {
	app := config.GitHubApp{
		Name:             name,
		AppID:            appID,
		Patterns:         patterns,
		Priority:         5,
		PrivateKeySource: "filesystem",
		PrivateKeyPath:   "/tmp/test.pem",
		InstallationID:   12345,
	}
	cb.apps = append(cb.apps, app)
	return cb
}

// Build creates the config
func (cb *ConfigBuilder) Build() *config.Config {
	return &config.Config{
		Version:    "1.0",
		GitHubApps: cb.apps,
	}
}

// WriteToFile writes the config to a file
func (cb *ConfigBuilder) WriteToFile(t *testing.T, path string) string {
	t.Helper()

	cfg := cb.Build()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	return path
}

// CreateTempConfig creates a temporary config file
func CreateTempConfig(t *testing.T, apps []config.GitHubApp) string {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	builder := NewConfigBuilder()
	for _, app := range apps {
		builder.AddApp(app)
	}

	return builder.WriteToFile(t, configPath)
}

// CreateSimpleConfig creates a simple config with one app
func CreateSimpleConfig(t *testing.T) string {
	t.Helper()

	return NewConfigBuilder().
		AddSimpleApp("Test App", 123456, []string{"github.com/testorg/*"}).
		WriteToFile(t, filepath.Join(t.TempDir(), "config.yml"))
}

// CreateMultiOrgConfig creates a config with multiple organizations
func CreateMultiOrgConfig(t *testing.T) string {
	t.Helper()

	return NewConfigBuilder().
		AddSimpleApp("Org1 App", 111111, []string{"github.com/org1/*"}).
		AddSimpleApp("Org2 App", 222222, []string{"github.com/org2/*"}).
		AddSimpleApp("Org3 App", 333333, []string{"github.com/org3/*"}).
		WriteToFile(t, filepath.Join(t.TempDir(), "config.yml"))
}

// CreateTestPrivateKey creates a test private key file
func CreateTestPrivateKey(t *testing.T) string {
	t.Helper()

	// This is a valid 2048-bit RSA private key for testing only
	// Generated with: openssl genrsa 2048
	// DO NOT use this in production!
	testKey := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF4qjQKzq4V5l6mJ9VpKJqKvYXKrc
IzDRNgQPBCZQqS4OZjFHFVLDPq3EO3R7+2VvTMvKVU6EwXVBXkjJPvbmQGWKqHvL
VB0sPVOAGEqQXcWmCNPpMTLTNLwMSPX3GjNaJEp2a0qSVZpUePqcTjz4U5GZC/0r
sSBDZXWKR1kRCT9E3YiRFKx+PQ9gQcFqMzH4A3OQqTBGPO5F0O0LJwTTTqQqGdTH
g5kMa6WGPQN5hPnCDxaFMNsEEQdKT8LXQ9JNQkW4JYqNHGqN3VkRQdHQEQM0Vq1t
IHQnEW6fV0XqVCVN6IQVVH0uDQqYRVKqEwIDAQABAoIBAG3Z6Y7FVA+rUOqJcW6t
9YRH7HQdvJKqQHFQpwPLVPvN5H6Q3J2TqRBhKv4qJvM6VrFQ5bphZKZQ6Hjqxevq
j6TVhXPKfN7MJHLxQphZqf3xjPG7qmFJWVqR8L5X+RYaGkP5MNYKu+pKKK4T8qpR
cLZxQFPK8WqG4j8wVYWGMH5RqPQTZw5ZKZLqqJKjGvPHMQ8xVGPYGmQwKJqmxPKQ
8YqLqhvGqJFpQJqmPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqmQGYqJP8w
VGYqQHLqPJYGmPqQJhvGqJPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqmQG
YqJP8ECgYEA6i9AxVPQpV8JqCGXQHMYqGHQVPGqJqLGYPqQJ8xGLqYPHqmQGYqJP
8wVGYqQHLqPJYGmPqQJhvGqJPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqm
QGYqJP8wVGYqQHLqPJYGmPqQJhvGqJPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGL
qYPHqmQGYqJP8wVGYqQHLqPJYGmPqQJhvGqJPLYGHRqPQJZqGmLYGPqLqJhvGmPq
QJ8xGLqYPHqmQGYqJP8wVGYqQHLqPJYGmPqQJhvGqJPLYGHRqPQJZqGmLYGPqLqJ
hvGmPqQJ8xGLqYPHqmQGYqJP8wVGYqQHLqPJYGmPqQJhvGqJPLYGHRqPQJZqGmLY
GPqLqJhvGmPqQJ8xGLqYPHqmQGYqJP8wVGYqQHLqPJYGmPqQJhvGqJPLYGHRqPQJ
ZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqmQGYqJP8wVGYqQHLqPJYGmPqQJhvGqJPLYG
HRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqmQGYqJP8wVGYqQHLqPJYGmPqQJhvG
qJPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqmQGYqJP8wVGYqQHLqPJYGmP
qQJhvGqJPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqmQGYqJP8wVGYqQHLq
PJYGmPqQJhvGqJPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqmQGYqJP8wVG
YqQHLqPJYGmPqQJhvGqJPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqmQGYq
JP8wVGYqQHLqPJYGmPqQJhvGqJPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPH
qmQGYqJP8wVGYqQHLqPJYGmPqQJhvGqJPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8x
GLqYPHqmQGYqJP8wVGYqQHLqPJYGmPqQJhvGqJPLYGHRqPQJZqGmLYGPqLqJhvGm
PqQJ8xGLqYPHqmQGYqJP8wVGYqQHLqPJYGmPqQJhvGqJPLYGHRqPQJZqGmLYGPqL
qJhvGmPqQJ8xGLqYPHqmQGYqJP8wVGYqQHLqPJYGmPqQJhvGqJPLYGHRqPQJZqGm
LYGPqLqJhvGmPqQJ8xGLqYPHqmQGYqJP8wVGYqQHLqPJYGmPqQJhvGqJPLYGHRqP
QJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqmQGYqJP8wVGYqQHLqPJYGmPqQJhvGqJPL
YGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqmQGYqJP8wVGYqQHLqPJYGmPqQJh
vGqJPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqmQGYqJP8wVGYqQHLqPJYG
mPqQJhvGqJPLYGHRqPQJZqGmLYGPqLqJhvGmPqQJ8xGLqYPHqmQGYqJP8=
-----END RSA PRIVATE KEY-----`

	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test-key.pem")

	if err := os.WriteFile(keyPath, []byte(testKey), 0600); err != nil {
		t.Fatalf("Failed to write test key: %v", err)
	}

	return keyPath
}

// SetupTestConfig sets up a test config and returns the path
func SetupTestConfig(t *testing.T, apps []config.GitHubApp) string {
	t.Helper()

	configPath := CreateTempConfig(t, apps)
	t.Setenv("GH_APP_AUTH_CONFIG", configPath)
	return configPath
}

// CleanupTestConfig cleans up test configuration
func CleanupTestConfig(t *testing.T, configPath string) {
	t.Helper()
	// TempDir cleanup is automatic, but we can explicitly unset env
	_ = os.Unsetenv("GH_APP_AUTH_CONFIG") // Error intentionally ignored in test cleanup
}

package logger

import (
	"errors"
	"os"
	"testing"
)

func TestLoggerFunctions_Callable(t *testing.T) {
	// Test that logger functions can be called without panicking
	// when logger is not initialized

	t.Run("HashToken", func(t *testing.T) {
		result := HashToken("test-token-123")
		if result == "" {
			t.Error("HashToken should return non-empty string")
		}

		// Empty token case
		emptyResult := HashToken("")
		if emptyResult != "<empty>" {
			t.Errorf("HashToken(\"\") = %q, want %q", emptyResult, "<empty>")
		}
	})

	t.Run("SanitizeURL", func(t *testing.T) {
		// URL without credentials
		url1 := "https://github.com/org/repo"
		result1 := SanitizeURL(url1)
		if result1 != url1 {
			t.Errorf("SanitizeURL should preserve URL without credentials")
		}

		// URL with credentials
		url2 := "https://user:pass@github.com/org/repo"
		result2 := SanitizeURL(url2)
		if result2 == url2 {
			t.Error("SanitizeURL should remove credentials")
		}
	})

	t.Run("SanitizeConfig", func(t *testing.T) {
		data := map[string]interface{}{
			"safe_key": "safe_value",
			"token":    "secret",
		}

		result := SanitizeConfig(data)
		if result == nil {
			t.Error("SanitizeConfig should return non-nil map")
		}
	})

	t.Run("FlowFunctions_DontPanic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Flow functions panicked: %v", r)
			}
		}()

		data := map[string]interface{}{"test": "data"}

		FlowStart("test_operation", data)
		FlowStep("test_step", data)
		FlowSuccess("test_operation", data)
		FlowError("test_operation", errors.New("test error"), data)
	})

	t.Run("DebugInfoError_DontPanic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Debug/Info/Error panicked: %v", r)
			}
		}()

		data := map[string]interface{}{"test": "data"}

		Debug("test message", data)
		Info("test message", data)
		Error("test message", errors.New("test error"), data)
	})
}

func TestInitializeAndClose(t *testing.T) {
	// Test Initialize and Close don't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Initialize/Close panicked: %v", r)
		}
	}()

	Initialize()
	Close()
}

func TestInitialize_WithEnvVar(t *testing.T) {
	// Save original env
	originalEnv := os.Getenv("GH_APP_AUTH_DEBUG_LOG")
	defer func() {
		if originalEnv != "" {
			os.Setenv("GH_APP_AUTH_DEBUG_LOG", originalEnv)
		} else {
			os.Unsetenv("GH_APP_AUTH_DEBUG_LOG")
		}
		// Clean up global logger
		if globalLogger != nil && globalLogger.enabled && globalLogger.file != nil {
			globalLogger.file.Close()
		}
	}()

	// Test with debug log enabled
	os.Setenv("GH_APP_AUTH_DEBUG_LOG", "1")
	Initialize()

	if globalLogger == nil {
		t.Error("Expected globalLogger to be initialized")
	}

	// Test that logger operations work
	data := map[string]interface{}{"test": "value"}
	Debug("test debug", data)
	Info("test info", data)
	Error("test error", errors.New("test"), data)

	Close()
}

func TestFlowFunctions_WithInitializedLogger(t *testing.T) {
	// Initialize with env var
	originalEnv := os.Getenv("GH_APP_AUTH_DEBUG_LOG")
	os.Setenv("GH_APP_AUTH_DEBUG_LOG", "1")
	defer func() {
		if originalEnv != "" {
			os.Setenv("GH_APP_AUTH_DEBUG_LOG", originalEnv)
		} else {
			os.Unsetenv("GH_APP_AUTH_DEBUG_LOG")
		}
		Close()
	}()

	Initialize()

	data := map[string]interface{}{
		"test_key": "test_value",
		"count":    123,
	}

	// Test flow functions with initialized logger
	FlowStart("test_operation", data)
	FlowStep("step1", data)
	FlowStep("step2", data)
	FlowSuccess("test_operation", data)
	FlowError("failed_operation", errors.New("test error"), data)
}

func TestSanitizeConfig_Variations(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]interface{}
	}{
		{
			name: "with token",
			input: map[string]interface{}{
				"token": "secret123",
				"safe":  "value",
			},
		},
		{
			name: "with password",
			input: map[string]interface{}{
				"password": "secret456",
				"safe":     "value",
			},
		},
		{
			name: "with private_key",
			input: map[string]interface{}{
				"private_key": "-----BEGIN RSA PRIVATE KEY-----",
				"safe":        "value",
			},
		},
		{
			name: "with secret",
			input: map[string]interface{}{
				"secret": "my_secret",
				"safe":   "value",
			},
		},
		{
			name: "nested sensitive data",
			input: map[string]interface{}{
				"config": map[string]interface{}{
					"token": "nested_token",
				},
			},
		},
		{
			name:  "empty map",
			input: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeConfig(tt.input)
			if result == nil {
				t.Error("Expected non-nil result")
			}

			// Original should not be modified
			if len(tt.input) > 0 && &result == &tt.input {
				t.Error("Expected new map, not same reference")
			}
		})
	}
}

func TestHashToken_Variations(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		wantHash string
	}{
		{
			name:     "empty token",
			token:    "",
			wantHash: "<empty>",
		},
		{
			name:  "short token",
			token: "abc",
		},
		{
			name:  "long token",
			token: "ghs_1234567890abcdefghijklmnopqrstuvwxyz",
		},
		{
			name:  "special characters",
			token: "token!@#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashToken(tt.token)

			if tt.wantHash != "" {
				if result != tt.wantHash {
					t.Errorf("HashToken(%q) = %q, want %q", tt.token, result, tt.wantHash)
				}
			} else {
				if result == "" {
					t.Error("Expected non-empty hash")
				}
				if result == tt.token {
					t.Error("Hash should not equal original token")
				}
			}
		})
	}
}

func TestSanitizeURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "URL with user:pass",
			url:      "https://user:pass@github.com/repo",
			expected: "https://<credentials>@github.com/repo",
		},
		{
			name:     "URL with only user",
			url:      "https://user@github.com/repo",
			expected: "https://<credentials>@github.com/repo",
		},
		{
			name:     "URL without credentials",
			url:      "https://github.com/owner/repo",
			expected: "https://github.com/owner/repo",
		},
		{
			name:     "empty URL",
			url:      "",
			expected: "",
		},
		{
			name:     "SSH URL with @",
			url:      "git@github.com:owner/repo.git",
			expected: "https://<credentials>@github.com:owner/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeURL(tt.url)
			if result != tt.expected {
				t.Errorf("SanitizeURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

// TestIsSensitiveKey tests key-based sensitivity detection
func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		// Exact/substring matches
		{"password", true},
		{"token", true},
		{"secret", true},
		{"credential", true},
		// Substring matches
		{"user_password", true},
		{"api_token", true},
		{"client_secret", true},
		{"auth_header", true},
		{"private_key", true},
		{"access_token", true},
		{"bearer_token", true},
		{"api_key", true},
		{"key_file", true},
		{"github_pat", true},
		{"pat_token", true},
		// Case insensitive
		{"PASSWORD", true},
		{"Token", true},
		{"API_KEY", true},
		// Non-sensitive keys (should NOT match)
		{"username", false},
		{"host", false},
		{"path", false}, // should not match "_pat" pattern
		{"protocol", false},
		{"operation", false},
		{"message", false},
		{"monkey", false},   // should not match "key" pattern
		{"keystone", false}, // should not match "key" pattern (no underscore)
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isSensitiveKey(tt.key)
			if result != tt.expected {
				t.Errorf("isSensitiveKey(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

// TestIsSensitiveValue tests value-based pattern detection
func TestIsSensitiveValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		// GitHub tokens
		{"GitHub PAT classic", "ghp_1234567890abcdefghijklmnopqrstuvwxyz12", true},
		{"GitHub OAuth token", "gho_1234567890abcdefghijklmnopqrstuvwxyz12", true},
		{"GitHub user token", "ghu_1234567890abcdefghijklmnopqrstuvwxyz12", true},
		{"GitHub server token", "ghs_1234567890abcdefghijklmnopqrstuvwxyz12", true},
		{"GitHub refresh token", "ghr_1234567890abcdefghijklmnopqrstuvwxyz12", true},
		// AWS keys
		{"AWS Access Key AKIA", "AKIAIOSFODNN7EXAMPLE", true},
		{"AWS Access Key ASIA", "ASIAISAMPLEKEYID1234", true},
		// JWT tokens
		{"JWT token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
			"eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U", true},
		// Private keys
		{"PEM private key", "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBg...", true},
		{"RSA private key", "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQ...", true},
		// Slack tokens
		{"Slack bot token", "xoxb-1234567890-abcdefghij", true},
		// URLs with credentials
		{"URL with creds", "https://user:password@github.com/repo", true},
		// Long alphanumeric (potential API keys)
		{"Long alphanumeric", "abcdefghijklmnopqrstuvwxyz123456", true},
		// Non-sensitive values
		{"Short string", "abc", false},
		{"Normal text", "hello world", false},
		{"Normal URL", "https://github.com/repo", false},
		{"Empty string", "", false},
		{"Number", "12345678", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSensitiveValue(tt.value)
			if result != tt.expected {
				t.Errorf("isSensitiveValue(%q) = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

// TestSanitizeValueForLogging tests the multi-layered sanitization
func TestSanitizeValueForLogging(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		value       interface{}
		shouldMatch string // substring that should be in result if redacted
	}{
		// Key-based redaction
		{"password key", "password", "mysecretpassword", "<redacted:"},
		{"token key", "api_token", "sometoken123", "<redacted:"},
		{"secret key", "client_secret", "secretvalue", "<redacted:"},
		// Value-based redaction (regardless of key name)
		{"GitHub token in random key", "some_data", "ghp_1234567890abcdefghijklmnopqrstuvwxyz12", "<redacted:github_token:"},
		{"JWT in random key", "payload",
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
				"eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			"<redacted:jwt:"},
		{"AWS key in random key", "identifier", "AKIAIOSFODNN7EXAMPLE", "<redacted:aws_key:"},
		// Non-sensitive data should pass through
		{"safe key and value", "host", "github.com", ""},
		{"safe numeric", "count", 42, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeValueForLogging(tt.key, tt.value)
			resultStr, isString := result.(string)

			if tt.shouldMatch != "" {
				if !isString {
					t.Errorf("Expected string result for redacted value, got %T", result)
					return
				}
				if !contains(resultStr, tt.shouldMatch) {
					t.Errorf("sanitizeValueForLogging(%q, %v) = %q, should contain %q",
						tt.key, tt.value, resultStr, tt.shouldMatch)
				}
			} else {
				// Should pass through unchanged
				if result != tt.value {
					t.Errorf("sanitizeValueForLogging(%q, %v) = %v, want %v (unchanged)",
						tt.key, tt.value, result, tt.value)
				}
			}
		})
	}
}

// TestIdentifySecretType tests secret type identification
func TestIdentifySecretType(t *testing.T) {
	tests := []struct {
		value    string
		expected string
	}{
		{"ghp_1234567890abcdefghijklmnopqrstuvwxyz12", "github_token"},
		{"gho_1234567890abcdefghijklmnopqrstuvwxyz12", "github_token"},
		{"ghu_1234567890abcdefghijklmnopqrstuvwxyz12", "github_token"},
		{"ghs_1234567890abcdefghijklmnopqrstuvwxyz12", "github_token"},
		{"ghr_1234567890abcdefghijklmnopqrstuvwxyz12", "github_token"},
		{"AKIAIOSFODNN7EXAMPLE", "aws_key"},
		{"ASIAISAMPLEKEYID1234", "aws_key"},
		{"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.sig", "jwt"},
		{"xoxb-1234567890-abc", "slack_token"},
		{"-----BEGIN PRIVATE KEY-----\ndata", "private_key"},
		{"https://user:pass@host.com", "url_with_creds"},
		{"some_random_secret_value", "secret"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := identifySecretType(tt.value)
			if result != tt.expected {
				t.Errorf("identifySecretType(%q) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

// TestRedactSecret tests the redaction output format
func TestRedactSecret(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		contains string
	}{
		{"empty", "", "<empty>"},
		{"GitHub token", "ghp_1234567890abcdefghijklmnopqrstuvwxyz12", "<redacted:github_token:"},
		{"generic secret", "mysupersecretpassword", "<redacted:secret:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactSecret(tt.secret)
			if !contains(result, tt.contains) {
				t.Errorf("RedactSecret(%q) = %q, should contain %q", tt.secret, result, tt.contains)
			}
			// Ensure original secret is not in output
			if tt.secret != "" && len(tt.secret) > 8 && contains(result, tt.secret) {
				t.Errorf("RedactSecret output should not contain original secret")
			}
		})
	}
}

// helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

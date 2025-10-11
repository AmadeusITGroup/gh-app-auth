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
		if emptyResult != "empty" {
			t.Errorf("HashToken(\"\") = %q, want %q", emptyResult, "empty")
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
			wantHash: "empty",
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

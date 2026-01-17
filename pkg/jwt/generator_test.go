package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// generateTestKey creates a test RSA private key
func generateTestKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

// writeTestKeyFile writes a test private key to a temporary file
func writeTestKeyFile(t *testing.T, key *rsa.PrivateKey, format string) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "jwt-test-key")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	keyPath := filepath.Join(tmpDir, "test-key.pem")

	var keyBytes []byte
	var blockType string

	switch format {
	case "pkcs1":
		keyBytes = x509.MarshalPKCS1PrivateKey(key)
		blockType = "RSA PRIVATE KEY"
	case "pkcs8":
		var err error
		keyBytes, err = x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			t.Fatalf("Failed to marshal PKCS8 key: %v", err)
		}
		blockType = "PRIVATE KEY"
	default:
		t.Fatalf("Unknown key format: %s", format)
	}

	block := &pem.Block{
		Type:  blockType,
		Bytes: keyBytes,
	}

	keyFile, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	defer keyFile.Close()

	if err := pem.Encode(keyFile, block); err != nil {
		t.Fatalf("Failed to encode PEM: %v", err)
	}

	// Set secure permissions (600) on the key file
	if err := os.Chmod(keyPath, 0600); err != nil {
		t.Fatalf("Failed to set key file permissions: %v", err)
	}

	return keyPath
}

func TestGenerator_GenerateToken(t *testing.T) {
	generator := NewGenerator()
	testKey, err := generateTestKey()
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	tests := []struct {
		name      string
		appID     int64
		keyFormat string
		wantErr   bool
	}{
		{
			name:      "valid token generation with PKCS1",
			appID:     12345,
			keyFormat: "pkcs1",
			wantErr:   false,
		},
		{
			name:      "valid token generation with PKCS8",
			appID:     67890,
			keyFormat: "pkcs8",
			wantErr:   false,
		},
		{
			name:      "zero app ID",
			appID:     0,
			keyFormat: "pkcs1",
			wantErr:   false, // JWT generation should work, validation happens elsewhere
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPath := writeTestKeyFile(t, testKey, tt.keyFormat)

			token, err := generator.GenerateToken(tt.appID, keyPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if token == "" {
					t.Error("GenerateToken() returned empty token")
					return
				}

				// Validate token structure
				if err := generator.ValidateToken(token); err != nil {
					t.Errorf("Generated token is invalid: %v", err)
				}

				// Check claims
				claims, err := generator.GetTokenClaims(token)
				if err != nil {
					t.Errorf("Failed to get token claims: %v", err)
					return
				}

				// Verify app ID
				if iss, ok := claims["iss"].(float64); !ok || int64(iss) != tt.appID {
					t.Errorf("Token iss claim = %v, want %v", claims["iss"], tt.appID)
				}

				// Verify timestamps
				now := time.Now().Unix()
				if iat, ok := claims["iat"].(float64); !ok || int64(iat) > now || int64(iat) < now-60 {
					t.Errorf("Token iat claim = %v, should be around %v", claims["iat"], now)
				}

				if exp, ok := claims["exp"].(float64); !ok || int64(exp) <= now || int64(exp) > now+600 {
					t.Errorf("Token exp claim = %v, should be between %v and %v", claims["exp"], now, now+600)
				}
			}
		})
	}
}

func TestGenerator_GenerateToken_Errors(t *testing.T) {
	generator := NewGenerator()

	tests := []struct {
		name            string
		appID           int64
		setupKeyFile    func(t *testing.T) string
		wantErrContains string
	}{
		{
			name:  "world-readable key file",
			appID: 12345,
			setupKeyFile: func(t *testing.T) string {
				t.Helper()
				testKey, err := generateTestKey()
				if err != nil {
					t.Fatalf("Failed to generate test key: %v", err)
				}
				keyPath := writeTestKeyFile(t, testKey, "pkcs1")

				// Make file world-readable (644)
				if err := os.Chmod(keyPath, 0644); err != nil {
					t.Fatalf("Failed to change key file permissions: %v", err)
				}
				return keyPath
			},
			wantErrContains: "overly permissive permissions",
		},
		{
			name:  "nonexistent key file",
			appID: 12345,
			setupKeyFile: func(t *testing.T) string {
				t.Helper()
				return "/nonexistent/path/key.pem"
			},
			wantErrContains: "failed to load private key",
		},
		{
			name:  "invalid PEM file",
			appID: 12345,
			setupKeyFile: func(t *testing.T) string {
				t.Helper()
				tmpDir, err := os.MkdirTemp("", "jwt-test-invalid")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				t.Cleanup(func() { os.RemoveAll(tmpDir) })

				keyPath := filepath.Join(tmpDir, "invalid.pem")
				if err := os.WriteFile(keyPath, []byte("not a pem file"), 0600); err != nil {
					t.Fatalf("Failed to write invalid file: %v", err)
				}
				return keyPath
			},
			wantErrContains: "failed to parse PEM block",
		},
		{
			name:  "unsupported key type",
			appID: 12345,
			setupKeyFile: func(t *testing.T) string {
				t.Helper()
				tmpDir, err := os.MkdirTemp("", "jwt-test-unsupported")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				t.Cleanup(func() { os.RemoveAll(tmpDir) })

				keyPath := filepath.Join(tmpDir, "unsupported.pem")
				block := &pem.Block{
					Type:  "EC PRIVATE KEY",
					Bytes: []byte("fake ec key data"),
				}

				keyFile, err := os.Create(keyPath)
				if err != nil {
					t.Fatalf("Failed to create key file: %v", err)
				}
				defer keyFile.Close()

				if err := pem.Encode(keyFile, block); err != nil {
					t.Fatalf("Failed to encode PEM: %v", err)
				}

				// Set secure permissions so permission check passes
				if err := os.Chmod(keyPath, 0600); err != nil {
					t.Fatalf("Failed to set key file permissions: %v", err)
				}
				return keyPath
			},
			wantErrContains: "unsupported private key type",
		},
		{
			name:  "permission denied",
			appID: 12345,
			setupKeyFile: func(t *testing.T) string {
				t.Helper()
				tmpDir, err := os.MkdirTemp("", "jwt-test-perms")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				t.Cleanup(func() { os.RemoveAll(tmpDir) })

				keyPath := filepath.Join(tmpDir, "no-perms.pem")
				if err := os.WriteFile(keyPath, []byte("test"), 0000); err != nil {
					t.Fatalf("Failed to write no-perms file: %v", err)
				}
				return keyPath
			},
			wantErrContains: "failed to load private key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip permission tests on Windows where file permissions work differently
			if runtime.GOOS == "windows" && tt.name == "world-readable key file" {
				t.Skip("Skipping permission test on Windows")
			}

			keyPath := tt.setupKeyFile(t)

			_, err := generator.GenerateToken(tt.appID, keyPath)

			if err == nil {
				t.Error("GenerateToken() expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Errorf("GenerateToken() error = %v, want error containing %v", err.Error(), tt.wantErrContains)
			}
		})
	}
}

func TestGenerator_ValidateToken(t *testing.T) {
	generator := NewGenerator()

	// Generate a valid token for testing
	testKey, err := generateTestKey()
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}
	keyPath := writeTestKeyFile(t, testKey, "pkcs1")
	validToken, err := generator.GenerateToken(12345, keyPath)
	if err != nil {
		t.Fatalf("Failed to generate valid token: %v", err)
	}

	tests := []struct {
		name    string
		token   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid token",
			token:   validToken,
			wantErr: false,
		},
		{
			name:    "invalid format - too few parts",
			token:   "header.payload",
			wantErr: true,
			errMsg:  "invalid JWT format",
		},
		{
			name:    "invalid format - too many parts",
			token:   "header.payload.signature.extra",
			wantErr: true,
			errMsg:  "invalid JWT format",
		},
		{
			name:    "invalid header encoding",
			token:   "invalid-base64.eyJpc3MiOjEyMzQ1LCJpYXQiOjE2MzA0NDM2MDAsImV4cCI6MTYzMDQ0NDIwMH0.signature",
			wantErr: true,
			errMsg:  "failed to", // Could be "failed to decode header" or "failed to parse header JSON"
		},
		{
			name:    "invalid header JSON",
			token:   "aW52YWxpZCBqc29u.eyJpc3MiOjEyMzQ1LCJpYXQiOjE2MzA0NDM2MDAsImV4cCI6MTYzMDQ0NDIwMH0.signature",
			wantErr: true,
			errMsg:  "failed to parse header JSON",
		},
		{
			name: "invalid algorithm",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
				"eyJpc3MiOjEyMzQ1LCJpYXQiOjE2MzA0NDM2MDAsImV4cCI6MTYzMDQ0NDIwMH0.signature",
			wantErr: true,
			errMsg:  "invalid algorithm",
		},
		{
			name:    "missing signature",
			token:   "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOjEyMzQ1LCJpYXQiOjE2MzA0NDM2MDAsImV4cCI6MTYzMDQ0NDIwMH0.",
			wantErr: true,
			errMsg:  "missing signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.ValidateToken(tt.token)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateToken() error = %v, want error containing %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestGenerator_GetTokenClaims(t *testing.T) {
	generator := NewGenerator()

	// Generate a valid token
	testKey, err := generateTestKey()
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}
	keyPath := writeTestKeyFile(t, testKey, "pkcs1")

	appID := int64(12345)
	token, err := generator.GenerateToken(appID, keyPath)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := generator.GetTokenClaims(token)
	if err != nil {
		t.Fatalf("GetTokenClaims() error = %v", err)
	}

	// Check required claims
	if iss, ok := claims["iss"].(float64); !ok || int64(iss) != appID {
		t.Errorf("claims[iss] = %v, want %v", claims["iss"], appID)
	}

	if _, ok := claims["iat"].(float64); !ok {
		t.Errorf("claims[iat] missing or wrong type")
	}

	if _, ok := claims["exp"].(float64); !ok {
		t.Errorf("claims[exp] missing or wrong type")
	}

	// Test with invalid token
	_, err = generator.GetTokenClaims("invalid.token")
	if err == nil {
		t.Error("GetTokenClaims() with invalid token should return error")
	}
}

func TestLoadPrivateKey_Formats(t *testing.T) {
	generator := NewGenerator()
	testKey, err := generateTestKey()
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	tests := []struct {
		name      string
		keyFormat string
		wantErr   bool
	}{
		{
			name:      "PKCS1 format",
			keyFormat: "pkcs1",
			wantErr:   false,
		},
		{
			name:      "PKCS8 format",
			keyFormat: "pkcs8",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPath := writeTestKeyFile(t, testKey, tt.keyFormat)

			loadedKey, err := generator.loadPrivateKey(keyPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("loadPrivateKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if loadedKey == nil {
					t.Error("loadPrivateKey() returned nil key")
					return
				}

				// Verify key integrity by comparing key parameters
				if loadedKey.N.Cmp(testKey.N) != 0 {
					t.Error("Loaded key N parameter doesn't match original")
				}
				if loadedKey.E != testKey.E {
					t.Error("Loaded key E parameter doesn't match original")
				}
			}
		})
	}
}

func TestTokenExpiration(t *testing.T) {
	generator := NewGenerator()
	testKey, err := generateTestKey()
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}
	keyPath := writeTestKeyFile(t, testKey, "pkcs1")

	token, err := generator.GenerateToken(12345, keyPath)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := generator.GetTokenClaims(token)
	if err != nil {
		t.Fatalf("Failed to get claims: %v", err)
	}

	iat := int64(claims["iat"].(float64))
	exp := int64(claims["exp"].(float64))

	// Verify expiration is exactly 10 minutes (600 seconds) after issued time
	expectedExp := iat + 600
	if exp != expectedExp {
		t.Errorf("Token expiration = %v, want %v (10 minutes after iat)", exp, expectedExp)
	}
}

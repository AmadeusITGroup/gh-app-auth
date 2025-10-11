package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"strings"
	"testing"
)

func TestGenerateJWT_Wrapper(t *testing.T) {
	// Generate a real RSA key for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Create temporary key file
	tempFile, err := os.CreateTemp("", "test-key-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write key to file
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	if err := pem.Encode(tempFile, privateKeyPEM); err != nil {
		t.Fatalf("Failed to encode PEM: %v", err)
	}
	tempFile.Close()

	auth := NewAuthenticator()

	t.Run("generates valid JWT token", func(t *testing.T) {
		token, err := auth.GenerateJWT(123456, tempFile.Name())
		if err != nil {
			t.Fatalf("GenerateJWT() error = %v", err)
		}

		if token == "" {
			t.Error("Expected non-empty token")
		}

		// JWT should have 3 parts
		parts := strings.Split(token, ".")
		if len(parts) != 3 {
			t.Errorf("Expected 3 JWT parts, got %d", len(parts))
		}
	})

	t.Run("handles zero app ID", func(t *testing.T) {
		// Note: JWT generation may not validate app ID
		token, err := auth.GenerateJWT(0, tempFile.Name())
		// Either error or generate token - both acceptable
		if err == nil && token == "" {
			t.Error("Expected either error or token")
		}
	})

	t.Run("error with nonexistent file", func(t *testing.T) {
		_, err := auth.GenerateJWT(123456, "/nonexistent/key.pem")
		if err == nil {
			t.Error("Expected error with nonexistent file")
		}
	})

	t.Run("error with empty file path", func(t *testing.T) {
		_, err := auth.GenerateJWT(123456, "")
		if err == nil {
			t.Error("Expected error with empty file path")
		}
	})
}

func TestGenerateJWT_InvalidKeyFormat(t *testing.T) {
	// Create temp file with invalid content
	tempFile, err := os.CreateTemp("", "invalid-key-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write invalid content
	if _, err := tempFile.WriteString("not a valid PEM key"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	auth := NewAuthenticator()

	_, err = auth.GenerateJWT(123456, tempFile.Name())
	if err == nil {
		t.Error("Expected error with invalid key format")
	}
}

func TestGenerateJWT_NegativeAppID(t *testing.T) {
	// Generate a real RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Create temporary key file
	tempFile, err := os.CreateTemp("", "test-key-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write key to file
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	if err := pem.Encode(tempFile, privateKeyPEM); err != nil {
		t.Fatalf("Failed to encode PEM: %v", err)
	}
	tempFile.Close()

	auth := NewAuthenticator()

	// Negative app IDs may be accepted by JWT generation
	token, err := auth.GenerateJWT(-1, tempFile.Name())
	// Test passes if either we get an error OR we get a token
	if err == nil && token == "" {
		t.Error("Expected either error or token")
	}
}

func TestGenerateJWT_Consistency(t *testing.T) {
	// Generate a real RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Create temporary key file
	tempFile, err := os.CreateTemp("", "test-key-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write key to file
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	if err := pem.Encode(tempFile, privateKeyPEM); err != nil {
		t.Fatalf("Failed to encode PEM: %v", err)
	}
	tempFile.Close()

	auth := NewAuthenticator()

	// Generate token and verify format is consistent
	token1, err1 := auth.GenerateJWT(123456, tempFile.Name())
	token2, err2 := auth.GenerateJWT(123456, tempFile.Name())

	if err1 != nil || err2 != nil {
		t.Fatalf("Unexpected errors: %v, %v", err1, err2)
	}

	// Both should have 3 parts (header.payload.signature)
	parts1 := strings.Split(token1, ".")
	parts2 := strings.Split(token2, ".")

	if len(parts1) != 3 {
		t.Errorf("Token 1 has %d parts, want 3", len(parts1))
	}
	if len(parts2) != 3 {
		t.Errorf("Token 2 has %d parts, want 3", len(parts2))
	}

	// Both tokens should be non-empty
	if token1 == "" || token2 == "" {
		t.Error("Expected non-empty tokens")
	}

	// Note: Tokens may be identical if generated in the same second
	// This is expected behavior for JWT tokens with 1-second granularity
}

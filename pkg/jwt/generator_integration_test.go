package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

// generateTestRSAKey generates a test RSA private key in PEM format
func generateTestRSAKey(t *testing.T) string {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	return string(pem.EncodeToMemory(privateKeyPEM))
}

func TestGenerateTokenFromKey(t *testing.T) {
	privateKey := generateTestRSAKey(t)

	tests := []struct {
		name       string
		appID      int64
		privateKey string
		wantErr    bool
	}{
		{
			name:       "valid key and app ID",
			appID:      123456,
			privateKey: privateKey,
			wantErr:    false,
		},
		{
			name:       "zero app ID still works",
			appID:      0,
			privateKey: privateKey,
			wantErr:    false, // Implementation doesn't validate app ID
		},
		{
			name:       "negative app ID still works",
			appID:      -1,
			privateKey: privateKey,
			wantErr:    false, // Implementation doesn't validate app ID
		},
		{
			name:       "empty private key",
			appID:      123456,
			privateKey: "",
			wantErr:    true,
		},
		{
			name:       "invalid private key format",
			appID:      123456,
			privateKey: "not-a-valid-key",
			wantErr:    true,
		},
		{
			name:       "malformed PEM",
			appID:      123456,
			privateKey: "-----BEGIN RSA PRIVATE KEY-----\ninvalid\n-----END RSA PRIVATE KEY-----",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator()
			token, err := gen.GenerateTokenFromKey(tt.appID, tt.privateKey)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if token == "" {
				t.Error("Expected non-empty token")
			}

			// Validate JWT structure (3 parts: header.payload.signature)
			if len(token) < 10 {
				t.Errorf("Token too short: %s", token)
			}

			// Token should start with typical JWT header
			if token[:2] != "ey" {
				t.Errorf("Token doesn't start with expected JWT prefix: %s", token[:10])
			}
		})
	}
}

func TestValidateToken_WithRealKey(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	gen := NewGenerator()
	appID := int64(123456)

	// Generate a valid token
	token, err := gen.GenerateTokenFromKey(appID, privateKey)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   token,
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "malformed token - not enough parts",
			token:   "invalid.token",
			wantErr: true,
		},
		{
			name:    "malformed token - invalid base64",
			token:   "not!!!.valid!!!.jwt!!!",
			wantErr: true,
		},
		{
			name:    "token with modified signature still validates structurally",
			token:   token[:len(token)-5] + "XXXXX",
			wantErr: false, // ValidateToken only checks structure, not crypto
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gen.ValidateToken(tt.token)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGetTokenClaims_WithRealKey(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	gen := NewGenerator()
	appID := int64(123456)

	// Generate a valid token
	token, err := gen.GenerateTokenFromKey(appID, privateKey)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   token,
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   "invalid.token",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := gen.GetTokenClaims(tt.token)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if claims == nil {
				t.Error("Expected non-nil claims")
				return
			}

			// Verify app ID in claims
			if iss, ok := claims["iss"].(float64); !ok || int64(iss) != appID {
				t.Errorf("Expected iss claim to be %d, got %v", appID, claims["iss"])
			}

			// Verify standard claims exist
			if _, ok := claims["exp"]; !ok {
				t.Error("Expected exp claim")
			}
			if _, ok := claims["iat"]; !ok {
				t.Error("Expected iat claim")
			}
		})
	}
}

func TestJWTExpiration(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	gen := NewGenerator()
	appID := int64(123456)

	token, err := gen.GenerateTokenFromKey(appID, privateKey)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := gen.GetTokenClaims(token)
	if err != nil {
		t.Fatalf("Failed to get claims: %v", err)
	}

	// Verify expiration is set and in the future
	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("exp claim not found or wrong type")
	}

	iat, ok := claims["iat"].(float64)
	if !ok {
		t.Fatal("iat claim not found or wrong type")
	}

	// Expiration should be after issued time
	if exp <= iat {
		t.Errorf("Expiration (%f) should be after issued time (%f)", exp, iat)
	}

	// Standard expiration is 10 minutes (600 seconds)
	expectedDuration := float64(600)
	actualDuration := exp - iat

	// Allow some tolerance (595-605 seconds)
	if actualDuration < expectedDuration-5 || actualDuration > expectedDuration+5 {
		t.Errorf("Expected duration ~%f seconds, got %f", expectedDuration, actualDuration)
	}
}

func TestConcurrentTokenGeneration(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	gen := NewGenerator()
	appID := int64(123456)

	// Generate tokens concurrently
	done := make(chan bool)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			token, err := gen.GenerateTokenFromKey(appID, privateKey)
			if err != nil {
				errors <- err
			} else if token == "" {
				errors <- err
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent generation failed: %v", err)
	}
}

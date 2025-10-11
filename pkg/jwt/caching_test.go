package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func generateTestKeyPEM(t *testing.T) string {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	return string(pem.EncodeToMemory(privateKeyPEM))
}

func TestKeyCache(t *testing.T) {
	gen := NewGenerator()
	key := generateTestKeyPEM(t)
	appID := int64(123456)

	// First call - should parse and cache the key
	token1, err := gen.GenerateTokenFromKey(appID, key)
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}

	// Second call - should use cached key
	token2, err := gen.GenerateTokenFromKey(appID, key)
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	// Both tokens should be valid (though different due to timestamps)
	if token1 == "" || token2 == "" {
		t.Error("Expected non-empty tokens")
	}

	// Validate both tokens
	if err := gen.ValidateToken(token1); err != nil {
		t.Errorf("Token 1 validation failed: %v", err)
	}
	if err := gen.ValidateToken(token2); err != nil {
		t.Errorf("Token 2 validation failed: %v", err)
	}
}

func TestMultipleApps(t *testing.T) {
	gen := NewGenerator()

	// Generate keys for different apps
	key1 := generateTestKeyPEM(t)
	key2 := generateTestKeyPEM(t)

	appID1 := int64(111111)
	appID2 := int64(222222)

	// Generate tokens for different apps
	token1, err := gen.GenerateTokenFromKey(appID1, key1)
	if err != nil {
		t.Fatalf("App 1 token generation failed: %v", err)
	}

	token2, err := gen.GenerateTokenFromKey(appID2, key2)
	if err != nil {
		t.Fatalf("App 2 token generation failed: %v", err)
	}

	// Verify both tokens are valid
	if err := gen.ValidateToken(token1); err != nil {
		t.Errorf("Token 1 validation failed: %v", err)
	}
	if err := gen.ValidateToken(token2); err != nil {
		t.Errorf("Token 2 validation failed: %v", err)
	}

	// Verify tokens are different
	if token1 == token2 {
		t.Error("Expected different tokens for different apps")
	}

	// Extract and verify claims
	claims1, err := gen.GetTokenClaims(token1)
	if err != nil {
		t.Fatalf("Failed to get claims 1: %v", err)
	}
	claims2, err := gen.GetTokenClaims(token2)
	if err != nil {
		t.Fatalf("Failed to get claims 2: %v", err)
	}

	// Verify app IDs in claims
	if iss1, ok := claims1["iss"].(float64); !ok || int64(iss1) != appID1 {
		t.Errorf("Expected iss=%d in token 1, got %v", appID1, claims1["iss"])
	}
	if iss2, ok := claims2["iss"].(float64); !ok || int64(iss2) != appID2 {
		t.Errorf("Expected iss=%d in token 2, got %v", appID2, claims2["iss"])
	}
}

func TestTokenClaimsStructure(t *testing.T) {
	gen := NewGenerator()
	key := generateTestKeyPEM(t)
	appID := int64(123456)

	token, err := gen.GenerateTokenFromKey(appID, key)
	if err != nil {
		t.Fatalf("Token generation failed: %v", err)
	}

	claims, err := gen.GetTokenClaims(token)
	if err != nil {
		t.Fatalf("Failed to get claims: %v", err)
	}

	// Verify required JWT claims exist
	requiredClaims := []string{"iss", "iat", "exp"}
	for _, claim := range requiredClaims {
		if _, ok := claims[claim]; !ok {
			t.Errorf("Missing required claim: %s", claim)
		}
	}

	// Verify claim types
	if _, ok := claims["iss"].(float64); !ok {
		t.Errorf("iss claim should be a number, got %T", claims["iss"])
	}
	if _, ok := claims["iat"].(float64); !ok {
		t.Errorf("iat claim should be a number, got %T", claims["iat"])
	}
	if _, ok := claims["exp"].(float64); !ok {
		t.Errorf("exp claim should be a number, got %T", claims["exp"])
	}
}

func TestValidateTokenEdgeCases(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "token with only two parts",
			token:   "part1.part2",
			wantErr: true,
		},
		{
			name:    "token with four parts",
			token:   "part1.part2.part3.part4",
			wantErr: true,
		},
		{
			name:    "empty parts",
			token:   "..",
			wantErr: true,
		},
		{
			name:    "only dots",
			token:   "...",
			wantErr: true,
		},
		{
			name:    "single dot",
			token:   ".",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gen.ValidateToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetTokenClaimsEdgeCases(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "token with two parts",
			token:   "part1.part2",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "token with invalid payload encoding",
			token:   "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.invalid!!!.signature",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := gen.GetTokenClaims(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTokenClaims() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator()

	if gen == nil {
		t.Fatal("NewGenerator() returned nil")
	}

	// Verify it's functional
	key := generateTestKeyPEM(t)
	_, err := gen.GenerateTokenFromKey(123456, key)
	if err != nil {
		t.Errorf("New generator should be functional: %v", err)
	}
}

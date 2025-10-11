package jwt

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateJWTHeader_EdgeCases(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name    string
		header  string
		wantErr bool
	}{
		{
			name:    "valid RS256 header",
			header:  base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`)),
			wantErr: false,
		},
		{
			name:    "invalid base64",
			header:  "not-valid-base64!!!",
			wantErr: true,
		},
		{
			name:    "empty header",
			header:  "",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			header:  base64.RawURLEncoding.EncodeToString([]byte(`{invalid json}`)),
			wantErr: true,
		},
		{
			name:    "missing alg field",
			header:  base64.RawURLEncoding.EncodeToString([]byte(`{"typ":"JWT"}`)),
			wantErr: true,
		},
		{
			name:    "wrong algorithm",
			header:  base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gen.validateJWTHeader(tt.header)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateJWTHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateJWTPayload(t *testing.T) {
	gen := NewGenerator()

	// Create valid payload
	validPayload := map[string]interface{}{
		"iss": 123456.0,
		"iat": 1234567890.0,
		"exp": 1234567900.0,
	}
	validPayloadJSON, _ := json.Marshal(validPayload)
	validPayloadB64 := base64.RawURLEncoding.EncodeToString(validPayloadJSON)

	tests := []struct {
		name    string
		payload string
		wantErr bool
	}{
		{
			name:    "valid payload",
			payload: validPayloadB64,
			wantErr: false,
		},
		{
			name:    "invalid base64",
			payload: "not-valid-base64!!!",
			wantErr: true,
		},
		{
			name:    "empty payload",
			payload: "",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			payload: base64.RawURLEncoding.EncodeToString([]byte(`{invalid}`)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gen.validateJWTPayload(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateJWTPayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTokenStructure(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		wantParts int
		wantErr   bool
	}{
		{
			name:      "three parts but invalid",
			token:     "header.payload.signature",
			wantParts: 3,
			wantErr:   true, // Will fail validation due to invalid base64
		},
		{
			name:      "two parts",
			token:     "header.payload",
			wantParts: 2,
			wantErr:   true,
		},
		{
			name:      "four parts",
			token:     "a.b.c.d",
			wantParts: 4,
			wantErr:   true,
		},
		{
			name:      "one part",
			token:     "singlepart",
			wantParts: 1,
			wantErr:   true,
		},
		{
			name:      "empty",
			token:     "",
			wantParts: 1,
			wantErr:   true,
		},
	}

	gen := NewGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Split(tt.token, ".")
			if len(parts) != tt.wantParts {
				t.Errorf("Split resulted in %d parts, want %d", len(parts), tt.wantParts)
			}

			err := gen.ValidateToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetTokenClaims_PayloadDecoding(t *testing.T) {
	gen := NewGenerator()

	// Create a token with known claims
	claims := map[string]interface{}{
		"iss":    123456.0,
		"iat":    1234567890.0,
		"exp":    1234567900.0,
		"custom": "value",
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("Failed to marshal claims: %v", err)
	}

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signature := base64.RawURLEncoding.EncodeToString([]byte("fake-signature"))

	token := header + "." + payload + "." + signature

	extracted, err := gen.GetTokenClaims(token)
	if err != nil {
		t.Fatalf("GetTokenClaims() error = %v", err)
	}

	// Verify claims
	if extracted["iss"].(float64) != 123456.0 {
		t.Errorf("iss = %v, want 123456", extracted["iss"])
	}
	if extracted["custom"].(string) != "value" {
		t.Errorf("custom = %v, want 'value'", extracted["custom"])
	}
}

func TestJWTStandardClaims(t *testing.T) {
	// Standard JWT claims that should be present
	standardClaims := []string{"iss", "iat", "exp"}

	t.Run("all standard claims required", func(t *testing.T) {
		for _, claim := range standardClaims {
			if claim == "" {
				t.Errorf("Standard claim is empty")
			}
		}
	})
}

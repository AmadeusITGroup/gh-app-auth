package jwt

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"
)
type Generator struct{}

// NewGenerator creates a new JWT token generator
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateToken generates a GitHub App JWT token
func (g *Generator) GenerateToken(appID int64, privateKeyPath string) (string, error) {
	// Load private key
	privateKey, err := g.loadPrivateKey(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to load private key: %w", err)
	}

	// Create JWT token
	token, err := g.createJWT(appID, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create JWT: %w", err)
	}
	return token, nil
}

// loadPrivateKey loads and parses an RSA private key from a PEM file
func (g *Generator) loadPrivateKey(keyPath string) (*rsa.PrivateKey, error) {
	// Check file permissions before reading
	fileInfo, err := os.Stat(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat private key file: %w", err)
	}
	
	// Ensure file is not world-readable (should be 600 or 400)
	if fileInfo.Mode().Perm() & 0044 != 0 {
		return nil, fmt.Errorf("private key file %s has overly permissive permissions %o", 
			keyPath, fileInfo.Mode().Perm())
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	// Ensure key data is zeroed out when we're done
	defer func() {
		for i := range keyData {
			keyData[i] = 0
		}
	}()

	// Parse PEM block
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	// Parse private key based on type
	var privateKey *rsa.PrivateKey
	switch block.Type {
	case "RSA PRIVATE KEY":
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS1 private key: %w", err)
		}
	case "PRIVATE KEY":
		parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
		}
		var ok bool
		privateKey, ok = parsedKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
	default:
		return nil, fmt.Errorf("unsupported private key type: %s", block.Type)
	}

	return privateKey, nil
}

// createJWT creates a GitHub App JWT token
func (g *Generator) createJWT(appID int64, privateKey *rsa.PrivateKey) (string, error) {
	// JWT Header
	header := map[string]interface{}{
		"alg": "RS256",
		"typ": "JWT",
	}

	// JWT Payload
	now := time.Now()
	payload := map[string]interface{}{
		"iss": appID,
		"iat": now.Unix(),
		"exp": now.Add(10 * time.Minute).Unix(), // GitHub Apps tokens expire in 10 minutes max
	}

	// Encode header and payload
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Base64 URL encode
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Create signing input
	signingInput := headerB64 + "." + payloadB64

	// Sign the token
	signature, err := g.signRS256(signingInput, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	// Create final JWT
	token := signingInput + "." + signature

	return token, nil
}

// signRS256 signs data using RS256 algorithm
func (g *Generator) signRS256(data string, privateKey *rsa.PrivateKey) (string, error) {
	// Create hash
	hasher := sha256.New()
	hasher.Write([]byte(data))
	hash := hasher.Sum(nil)

	// Sign hash
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash)
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}

	// Base64 URL encode the signature
	return base64.RawURLEncoding.EncodeToString(signature), nil
}

// ValidateToken validates a JWT token structure (for testing)
func (g *Generator) ValidateToken(token string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	// Validate header
	headerData, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("failed to decode header: %w", err)
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerData, &header); err != nil {
		return fmt.Errorf("failed to parse header JSON: %w", err)
	}

	if header["alg"] != "RS256" {
		return fmt.Errorf("invalid algorithm: expected RS256, got %v", header["alg"])
	}

	if header["typ"] != "JWT" {
		return fmt.Errorf("invalid type: expected JWT, got %v", header["typ"])
	}

	// Validate payload
	payloadData, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("failed to decode payload: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadData, &payload); err != nil {
		return fmt.Errorf("failed to parse payload JSON: %w", err)
	}

	// Check required fields
	if _, ok := payload["iss"]; !ok {
		return fmt.Errorf("missing 'iss' claim")
	}

	if _, ok := payload["iat"]; !ok {
		return fmt.Errorf("missing 'iat' claim")
	}

	if _, ok := payload["exp"]; !ok {
		return fmt.Errorf("missing 'exp' claim")
	}

	// Check signature is present (we don't validate it here since we'd need the public key)
	if parts[2] == "" {
		return fmt.Errorf("missing signature")
	}

	return nil
}

// GetTokenClaims extracts claims from a JWT token (for testing/debugging)
func (g *Generator) GetTokenClaims(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	payloadData, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadData, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse payload JSON: %w", err)
	}

	return payload, nil
}

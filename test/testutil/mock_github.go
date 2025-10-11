package testutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// MockGitHubServer provides a mock GitHub API server for testing
type MockGitHubServer struct {
	*httptest.Server
	t                     *testing.T
	installationTokens    map[string]string // appID -> token
	jwtValidationBehavior JWTValidationBehavior
	errorMode             ErrorMode
}

// JWTValidationBehavior controls JWT validation behavior
type JWTValidationBehavior int

const (
	JWTAcceptAll JWTValidationBehavior = iota
	JWTRejectAll
	JWTValidateFormat
)

// ErrorMode controls error simulation
type ErrorMode int

const (
	ErrorNone ErrorMode = iota
	Error401Unauthorized
	Error403Forbidden
	Error404NotFound
	Error500ServerError
	ErrorTimeout
)

// NewMockGitHubServer creates a new mock GitHub API server
func NewMockGitHubServer(t *testing.T) *MockGitHubServer {
	t.Helper()

	m := &MockGitHubServer{
		t:                     t,
		installationTokens:    make(map[string]string),
		jwtValidationBehavior: JWTAcceptAll,
		errorMode:             ErrorNone,
	}

	mux := http.NewServeMux()

	// Mock installation token endpoint
	mux.HandleFunc("/app/installations/", m.handleInstallationToken)

	// Mock app endpoint (for validation)
	mux.HandleFunc("/app", m.handleApp)

	m.Server = httptest.NewServer(mux)

	return m
}

// SetInstallationToken configures a token for a specific installation
func (m *MockGitHubServer) SetInstallationToken(installationID, token string) {
	m.installationTokens[installationID] = token
}

// SetJWTValidation sets the JWT validation behavior
func (m *MockGitHubServer) SetJWTValidation(behavior JWTValidationBehavior) {
	m.jwtValidationBehavior = behavior
}

// SetErrorMode sets the error simulation mode
func (m *MockGitHubServer) SetErrorMode(mode ErrorMode) {
	m.errorMode = mode
}

// handleInstallationToken handles POST /app/installations/{id}/access_tokens
func (m *MockGitHubServer) handleInstallationToken(w http.ResponseWriter, r *http.Request) {
	// Apply error mode
	if m.errorMode != ErrorNone {
		m.handleError(w)
		return
	}

	// Validate method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate JWT in Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing authorization", http.StatusUnauthorized)
		return
	}

	if !m.validateJWT(authHeader) {
		http.Error(w, "Invalid JWT", http.StatusUnauthorized)
		return
	}

	// Extract installation ID from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	installationID := parts[3]

	// Get or generate token
	token, ok := m.installationTokens[installationID]
	if !ok {
		// Generate default token
		token = fmt.Sprintf("ghs_mock_token_%s_%d", installationID, time.Now().Unix())
		m.installationTokens[installationID] = token
	}

	// Return token response
	response := map[string]interface{}{
		"token":      token,
		"expires_at": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response) // Error intentionally ignored in test mock
}

// handleApp handles GET /app
func (m *MockGitHubServer) handleApp(w http.ResponseWriter, r *http.Request) {
	// Apply error mode
	if m.errorMode != ErrorNone {
		m.handleError(w)
		return
	}

	// Validate JWT
	authHeader := r.Header.Get("Authorization")
	if !m.validateJWT(authHeader) {
		http.Error(w, "Invalid JWT", http.StatusUnauthorized)
		return
	}

	// Return mock app info
	response := map[string]interface{}{
		"id":   123456,
		"name": "Test App",
		"slug": "test-app",
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response) // Error intentionally ignored in test mock
}

// validateJWT validates the JWT token based on configured behavior
func (m *MockGitHubServer) validateJWT(authHeader string) bool {
	switch m.jwtValidationBehavior {
	case JWTAcceptAll:
		return strings.HasPrefix(authHeader, "Bearer ")
	case JWTRejectAll:
		return false
	case JWTValidateFormat:
		// Basic format validation
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return false
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		// JWT should have 3 parts separated by dots
		return len(strings.Split(token, ".")) == 3
	default:
		return false
	}
}

// handleError handles error simulation
func (m *MockGitHubServer) handleError(w http.ResponseWriter) {
	switch m.errorMode {
	case ErrorNone:
		// No error, this case should not be reached
		return
	case Error401Unauthorized:
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	case Error403Forbidden:
		http.Error(w, "Forbidden", http.StatusForbidden)
	case Error404NotFound:
		http.Error(w, "Not Found", http.StatusNotFound)
	case Error500ServerError:
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	case ErrorTimeout:
		// Simulate timeout by not responding
		time.Sleep(100 * time.Millisecond)
		http.Error(w, "Request Timeout", http.StatusRequestTimeout)
	}
}

// Close shuts down the mock server
func (m *MockGitHubServer) Close() {
	if m.Server != nil {
		m.Server.Close()
	}
}

// GetURL returns the base URL of the mock server
func (m *MockGitHubServer) GetURL() string {
	return m.Server.URL
}

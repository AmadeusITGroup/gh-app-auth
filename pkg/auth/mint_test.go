package auth

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

type mintRoundTripper func(*http.Request) (*http.Response, error)

func (f mintRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestMintInstallationTokenAlwaysRequestsFreshToken(t *testing.T) {
	keyPath := setupTestKeyFile(t)
	app := &config.GitHubApp{
		Name:             "Test App",
		AppID:            123456,
		InstallationID:   789012,
		PrivateKeyPath:   keyPath,
		PrivateKeySource: config.PrivateKeySourceFilesystem,
	}

	authenticator := NewAuthenticator()
	requestCount := 0
	authenticator.httpClient = &http.Client{
		Transport: mintRoundTripper(func(req *http.Request) (*http.Response, error) {
			requestCount++
			if req.Method != http.MethodPost {
				t.Errorf("request method = %s, want POST", req.Method)
			}
			if req.URL.Host != "api.github.com" {
				t.Errorf("request host = %s, want api.github.com", req.URL.Host)
			}
			if req.URL.Path != "/app/installations/789012/access_tokens" {
				t.Errorf("request path = %s, want installation token path", req.URL.Path)
			}
			if !strings.HasPrefix(req.Header.Get("Authorization"), "Bearer ") {
				t.Error("request is missing a Bearer authorization header")
			}

			token := fmt.Sprintf("ghs_test_installation_token_%d", requestCount)
			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(fmt.Sprintf(
					`{"token":%q,"expires_at":"2030-01-01T00:00:00Z"}`,
					token,
				))),
				Request: req,
			}, nil
		}),
	}

	token1, err := authenticator.MintInstallationToken(app, "github.com/myorg/myrepo")
	if err != nil {
		t.Fatalf("first MintInstallationToken() error = %v", err)
	}
	token2, err := authenticator.MintInstallationToken(app, "github.com/myorg/myrepo")
	if err != nil {
		t.Fatalf("second MintInstallationToken() error = %v", err)
	}

	if token1 != "ghs_test_installation_token_1" {
		t.Errorf("first token = %q, want first minted token", token1)
	}
	if token2 != "ghs_test_installation_token_2" {
		t.Errorf("second token = %q, want second minted token", token2)
	}
	if requestCount != 2 {
		t.Errorf("request count = %d, want 2", requestCount)
	}
	if got := authenticator.tokenCache.Size(); got != 0 {
		t.Errorf("token cache size = %d, want 0 for fresh minting", got)
	}
}

func TestHTTPClientRejectsCrossHostRedirect(t *testing.T) {
	client := newHTTPClient()
	from := httptest.NewRequest(http.MethodGet, "https://api.github.com/app", nil)
	to := httptest.NewRequest(http.MethodGet, "https://attacker.example/app", nil)

	err := client.CheckRedirect(to, []*http.Request{from})
	if err == nil || !strings.Contains(err.Error(), "refusing cross-host redirect") {
		t.Fatalf("CheckRedirect() error = %v, want cross-host redirect error", err)
	}
}

func TestHTTPClientAllowsSameHostRedirect(t *testing.T) {
	client := newHTTPClient()
	from := httptest.NewRequest(http.MethodGet, "https://api.github.com/app", nil)
	to := httptest.NewRequest(http.MethodGet, "https://api.github.com/app?page=2", nil)

	if err := client.CheckRedirect(to, []*http.Request{from}); err != nil {
		t.Fatalf("CheckRedirect() error = %v, want nil", err)
	}
}

func TestMintInstallationTokenValidatesAppAndKey(t *testing.T) {
	authenticator := NewAuthenticator()
	if _, err := authenticator.MintInstallationToken(nil, "github.com/myorg/myrepo"); err == nil ||
		!strings.Contains(err.Error(), "configuration is required") {
		t.Fatalf("MintInstallationToken(nil) error = %v, want configuration error", err)
	}

	app := &config.GitHubApp{
		Name:             "Test App",
		AppID:            123456,
		InstallationID:   789012,
		PrivateKeySource: config.PrivateKeySourceFilesystem,
		PrivateKeyPath:   "/nonexistent/test-key.pem",
	}
	if _, err := authenticator.MintInstallationToken(app, "github.com/myorg/myrepo"); err == nil ||
		!strings.Contains(err.Error(), "failed to get private key") {
		t.Fatalf("MintInstallationToken() error = %v, want private-key error", err)
	}
}

func TestMintInstallationTokenRejectsInvalidResponses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
	}{
		{
			name:       "empty token",
			statusCode: http.StatusCreated,
			body:       `{"token":"","expires_at":"2030-01-01T00:00:00Z"}`,
			wantErr:    "empty installation token",
		},
		{
			name:       "newline in token",
			statusCode: http.StatusCreated,
			body:       `{"token":"ghs_bad\ntoken","expires_at":"2030-01-01T00:00:00Z"}`,
			wantErr:    "invalid installation token",
		},
		{
			name:       "API failure",
			statusCode: http.StatusForbidden,
			body:       `{"message":"forbidden"}`,
			wantErr:    "status 403",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPath := setupTestKeyFile(t)
			app := &config.GitHubApp{
				Name:             "Test App",
				AppID:            123456,
				InstallationID:   789012,
				PrivateKeyPath:   keyPath,
				PrivateKeySource: config.PrivateKeySourceFilesystem,
			}

			authenticator := NewAuthenticator()
			authenticator.httpClient = &http.Client{
				Transport: mintRoundTripper(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.statusCode,
						Header:     make(http.Header),
						Body:       io.NopCloser(strings.NewReader(tt.body)),
					}, nil
				}),
			}

			_, err := authenticator.MintInstallationToken(app, "github.com/myorg/myrepo")
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("MintInstallationToken() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

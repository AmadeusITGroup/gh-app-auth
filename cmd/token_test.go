package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

func TestTokenCommandPrintsOnlyToken(t *testing.T) {
	var request execCredentialRequest
	var output bytes.Buffer

	cmd := newTokenCmd(func(got execCredentialRequest) (string, error) {
		request = got
		return "ghs_test_installation_token", nil
	})
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"--app-id", "123456",
		"--repo", "github.com/myorg/myrepo",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if output.String() != "ghs_test_installation_token\n" {
		t.Errorf("stdout = %q, want token followed by one newline", output.String())
	}
	if request.AppID != 123456 {
		t.Errorf("request AppID = %d, want %d", request.AppID, 123456)
	}
	if request.Repository != "github.com/myorg/myrepo" {
		t.Errorf("request repository = %q, want %q", request.Repository, "github.com/myorg/myrepo")
	}
}

func TestTokenCommandSupportsClientID(t *testing.T) {
	var request execCredentialRequest

	cmd := newTokenCmd(func(got execCredentialRequest) (string, error) {
		request = got
		return "ghs_test_installation_token", nil
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"--client-id", "Iv1.TestClientID",
		"--installation-id", "789012",
		"--repo", "github.com/myorg/myrepo",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if request.ClientID != "Iv1.TestClientID" {
		t.Errorf("request ClientID = %q, want %q", request.ClientID, "Iv1.TestClientID")
	}
	if request.InstallationID != 789012 {
		t.Errorf("request installation ID = %d, want %d", request.InstallationID, 789012)
	}
}

func TestTokenCommandRejectsInvalidSelectors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing App selector",
			args: []string{"--repo", "github.com/myorg/myrepo"},
			want: "must specify either --app-id or --client-id",
		},
		{
			name: "both App selectors",
			args: []string{
				"--app-id", "123456",
				"--client-id", "Iv1.TestClientID",
				"--repo", "github.com/myorg/myrepo",
			},
			want: "cannot use both --app-id and --client-id",
		},
		{
			name: "missing repository",
			args: []string{"--app-id", "123456"},
			want: "repository is required",
		},
		{
			name: "negative App ID",
			args: []string{
				"--app-id", "-1",
				"--repo", "github.com/myorg/myrepo",
			},
			want: "app ID must be positive",
		},
		{
			name: "negative installation ID",
			args: []string{
				"--app-id", "123456",
				"--installation-id", "-1",
				"--repo", "github.com/myorg/myrepo",
			},
			want: "installation ID must be positive",
		},
		{
			name: "zero installation ID",
			args: []string{
				"--app-id", "123456",
				"--installation-id", "0",
				"--repo", "github.com/myorg/myrepo",
			},
			want: "installation ID must be positive",
		},
		{
			name: "zero App ID",
			args: []string{
				"--app-id", "0",
				"--repo", "github.com/myorg/myrepo",
			},
			want: "app ID must be positive",
		},
		{
			name: "HTTP repository",
			args: []string{
				"--app-id", "123456",
				"--repo", "http://github.com/myorg/myrepo",
			},
			want: "repository must use HTTPS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolverCalled := false
			cmd := newTokenCmd(func(execCredentialRequest) (string, error) {
				resolverCalled = true
				return "unexpected-token", nil
			})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Execute() error = %v, want containing %q", err, tt.want)
			}
			if resolverCalled {
				t.Fatal("resolver should not be called for invalid selectors")
			}
		})
	}
}

func TestTokenCommandPropagatesResolverError(t *testing.T) {
	wantErr := errors.New("configured App route not found")
	cmd := newTokenCmd(func(execCredentialRequest) (string, error) {
		return "", wantErr
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"--app-id", "123456",
		"--repo", "github.com/myorg/myrepo",
	})

	if err := cmd.Execute(); !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestCanonicalTokenRepository(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{
			name:  "host owner repository",
			input: "github.com/myorg/myrepo",
			want:  "github.com/myorg/myrepo",
		},
		{
			name:  "HTTPS repository",
			input: "https://github.com/myorg/myrepo",
			want:  "github.com/myorg/myrepo",
		},
		{
			name:    "owner repository without host",
			input:   "myorg/myrepo",
			wantErr: "host/owner/repository",
		},
		{
			name:    "embedded credentials",
			input:   "https://user:password@github.com/myorg/myrepo",
			wantErr: "embedded credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := canonicalTokenRepository(tt.input)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("canonicalTokenRepository() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("canonicalTokenRepository() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("canonicalTokenRepository() = %q, want %q", got, tt.want)
			}
		})
	}
}

type tokenErrorWriter struct{}

func (tokenErrorWriter) Write([]byte) (int, error) {
	return 0, errors.New("stdout unavailable")
}

func TestTokenCommandRejectsEmptyResolvedToken(t *testing.T) {
	cmd := newTokenCmd(func(execCredentialRequest) (string, error) {
		return "", nil
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"--app-id", "123456",
		"--repo", "github.com/myorg/myrepo",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "empty token") {
		t.Fatalf("Execute() error = %v, want empty-token error", err)
	}
}

func TestTokenCommandPropagatesOutputError(t *testing.T) {
	cmd := newTokenCmd(func(execCredentialRequest) (string, error) {
		return "ghs_test_installation_token", nil
	})
	cmd.SetOut(tokenErrorWriter{})
	cmd.SetArgs([]string{
		"--app-id", "123456",
		"--repo", "github.com/myorg/myrepo",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "stdout unavailable") {
		t.Fatalf("Execute() error = %v, want output error", err)
	}
}

func TestResolveTokenCredentialWithUsesConfiguredApp(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yml")
	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	cfg := &config.Config{
		Version: "1.0",
		GitHubApps: []config.GitHubApp{
			{
				Name:             "Test App",
				AppID:            123456,
				InstallationID:   789012,
				PrivateKeySource: config.PrivateKeySourceKeyring,
				Patterns:         []string{"github.com/myorg/*"},
			},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	request, err := newTokenCredentialRequest("github.com/myorg/myrepo", 123456, "", 789012)
	if err != nil {
		t.Fatalf("newTokenCredentialRequest() error = %v", err)
	}

	mintCalled := false
	token, err := resolveTokenCredentialWith(request, func(app *config.GitHubApp, repoURL string) (string, error) {
		mintCalled = true
		if app.AppID != 123456 || app.InstallationID != 789012 {
			t.Errorf("selected App = (%d, %d), want (123456, 789012)", app.AppID, app.InstallationID)
		}
		if repoURL != "github.com/myorg/myrepo" {
			t.Errorf("mint repository = %q, want %q", repoURL, "github.com/myorg/myrepo")
		}
		return "ghs_test_installation_token", nil
	})
	if err != nil {
		t.Fatalf("resolveTokenCredentialWith() error = %v", err)
	}
	if !mintCalled {
		t.Fatal("mint function was not called")
	}
	if token != "ghs_test_installation_token" {
		t.Errorf("token = %q, want %q", token, "ghs_test_installation_token")
	}
}

func TestResolveTokenCredentialWithPropagatesMintError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yml")
	t.Setenv("GH_APP_AUTH_CONFIG", configPath)

	cfg := &config.Config{
		Version: "1.0",
		GitHubApps: []config.GitHubApp{
			{
				Name:             "Test App",
				AppID:            123456,
				InstallationID:   789012,
				PrivateKeySource: config.PrivateKeySourceKeyring,
				Patterns:         []string{"github.com/myorg/*"},
			},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	request, err := newTokenCredentialRequest("github.com/myorg/myrepo", 123456, "", 0)
	if err != nil {
		t.Fatalf("newTokenCredentialRequest() error = %v", err)
	}

	wantErr := errors.New("mint unavailable")
	_, err = resolveTokenCredentialWith(request, func(*config.GitHubApp, string) (string, error) {
		return "", wantErr
	})
	if err == nil || !strings.Contains(err.Error(), wantErr.Error()) {
		t.Fatalf("resolveTokenCredentialWith() error = %v, want containing %q", err, wantErr)
	}
}

func TestResolveTokenCredentialRejectsUnconfiguredRoutes(t *testing.T) {
	tests := []struct {
		name           string
		installationID int64
		repository     string
		wantErr        string
	}{
		{
			name:       "repository outside configured route",
			repository: "attacker.example/owner/repo",
			wantErr:    "matches repository",
		},
		{
			name:           "installation outside configuration",
			installationID: 999,
			repository:     "github.com/myorg/myrepo",
			wantErr:        "matches app ID 123456 and installation ID 999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.yml")
			t.Setenv("GH_APP_AUTH_CONFIG", configPath)

			cfg := &config.Config{
				Version: "1.0",
				GitHubApps: []config.GitHubApp{
					{
						Name:             "Test App",
						AppID:            123456,
						InstallationID:   789012,
						PrivateKeySource: config.PrivateKeySourceKeyring,
						Patterns:         []string{"github.com/myorg/*"},
					},
				},
			}
			if err := cfg.Save(); err != nil {
				t.Fatalf("Save() error = %v", err)
			}
			if _, err := os.Stat(configPath); err != nil {
				t.Fatalf("config file was not created: %v", err)
			}

			request, err := newTokenCredentialRequest(
				tt.repository, 123456, "", tt.installationID,
			)
			if err != nil {
				t.Fatalf("newTokenCredentialRequest() error = %v", err)
			}

			_, err = resolveTokenCredential(request)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("resolveTokenCredential() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

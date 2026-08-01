package cmd

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

func TestExecCommand(t *testing.T) {
	t.Run("repository routing", func(t *testing.T) {
		var (
			resolvedRequest execCredentialRequest
			runName         string
			runArgs         []string
			runEnv          []string
		)

		cmd := newExecCmd(
			func(request execCredentialRequest) (execCredential, error) {
				resolvedRequest = request
				return execCredential{
					Token:      "secret-token",
					Host:       gitHubAPIHost,
					Repository: request.Repository,
				}, nil
			},
			func(
				_ context.Context,
				name string,
				args []string,
				env []string,
				_ io.Reader,
				_ io.Writer,
				_ io.Writer,
			) error {
				runName = name
				runArgs = args
				runEnv = env
				return nil
			},
		)
		cmd.SetArgs([]string{"--repo", "github.com/myorg/myrepo", "--", "gh", "api", "repos/{owner}/{repo}"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if resolvedRequest.Repository != "github.com/myorg/myrepo" {
			t.Errorf("resolved repository = %q, want %q", resolvedRequest.Repository, "github.com/myorg/myrepo")
		}
		if runName != "gh" {
			t.Errorf("command name = %q, want %q", runName, "gh")
		}
		if got := strings.Join(runArgs, " "); got != "api repos/{owner}/{repo}" {
			t.Errorf("command args = %q, want %q", got, "api repos/{owner}/{repo}")
		}
		assertEnvironmentValue(t, runEnv, "GH_TOKEN", "secret-token")
		assertEnvironmentValue(t, runEnv, "GH_HOST", gitHubAPIHost)
		assertEnvironmentValue(t, runEnv, "GH_REPO", "github.com/myorg/myrepo")
	})

	t.Run("repository-independent App routing", func(t *testing.T) {
		var resolvedRequest execCredentialRequest
		var runEnv []string

		cmd := newExecCmd(
			func(request execCredentialRequest) (execCredential, error) {
				resolvedRequest = request
				return execCredential{Token: "app-token", Host: gitHubAPIHost}, nil
			},
			func(
				_ context.Context,
				_ string,
				_ []string,
				env []string,
				_ io.Reader,
				_ io.Writer,
				_ io.Writer,
			) error {
				runEnv = env
				return nil
			},
		)
		cmd.SetArgs([]string{
			"--app-id", "123456",
			"--installation-id", "789012",
			"--", "gh", "api", "/installation/repositories",
		})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if resolvedRequest.Repository != "" {
			t.Errorf("resolved repository = %q, want empty", resolvedRequest.Repository)
		}
		if resolvedRequest.AppID != 123456 || resolvedRequest.InstallationID != 789012 {
			t.Errorf("resolved IDs = (%d, %d), want (123456, 789012)", resolvedRequest.AppID, resolvedRequest.InstallationID)
		}
		assertEnvironmentValue(t, runEnv, "GH_TOKEN", "app-token")
		assertEnvironmentValue(t, runEnv, "GH_HOST", gitHubAPIHost)
		assertEnvironmentMissing(t, runEnv, "GH_REPO")
	})
}

func TestExecCommandRejectsNonPositiveSelectors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "zero App ID", args: []string{"--app-id", "0", "--", "gh", "api", "user"}},
		{name: "negative App ID", args: []string{"--app-id", "-1", "--", "gh", "api", "user"}},
		{name: "zero installation ID", args: []string{"--installation-id", "0", "--", "gh", "api", "user"}},
		{name: "negative installation ID", args: []string{"--installation-id", "-1", "--", "gh", "api", "user"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newExecCmd(
				func(execCredentialRequest) (execCredential, error) {
					t.Fatal("resolver should not be called")
					return execCredential{}, nil
				},
				func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error {
					t.Fatal("runner should not be called")
					return nil
				},
			)
			cmd.SetArgs(tt.args)

			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "must be positive") {
				t.Fatalf("Execute() error = %v, want positive-value error", err)
			}
		})
	}
}

func TestExecCommandPropagatesErrors(t *testing.T) {
	t.Run("credential resolution", func(t *testing.T) {
		wantErr := errors.New("token unavailable")
		cmd := newExecCmd(
			func(execCredentialRequest) (execCredential, error) { return execCredential{}, wantErr },
			func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error {
				t.Fatal("runner should not be called")
				return nil
			},
		)
		cmd.SetArgs([]string{"--repo", "github.com/myorg/myrepo", "--", "gh", "api", "user"})

		if err := cmd.Execute(); !errors.Is(err, wantErr) {
			t.Fatalf("Execute() error = %v, want %v", err, wantErr)
		}
	})

	t.Run("child process", func(t *testing.T) {
		wantErr := errors.New("child failed")
		cmd := newExecCmd(
			func(request execCredentialRequest) (execCredential, error) {
				return execCredential{Token: "secret-token", Host: gitHubAPIHost, Repository: request.Repository}, nil
			},
			func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error {
				return wantErr
			},
		)
		cmd.SetArgs([]string{"--repo", "github.com/myorg/myrepo", "--", "gh", "api", "user"})

		if err := cmd.Execute(); !errors.Is(err, wantErr) {
			t.Fatalf("Execute() error = %v, want %v", err, wantErr)
		}
		if !cmd.SilenceErrors || !cmd.SilenceUsage {
			t.Fatal("child failures should not print an error or command usage")
		}
	})
}

func TestSelectExecApp(t *testing.T) {
	cfg := &config.Config{GitHubApps: []config.GitHubApp{
		{Name: "org-a", AppID: 100, InstallationID: 200, Patterns: []string{"github.com/org-a/*"}},
		{Name: "org-b", AppID: 100, InstallationID: 300, Patterns: []string{"github.com/org-b/*"}},
		{Name: "enterprise", AppID: 400, InstallationID: 500, Patterns: []string{"github.example.com/org/*"}},
	}}

	tests := []struct {
		name        string
		request     execCredentialRequest
		wantName    string
		wantErrText string
	}{
		{
			name:     "installation ID",
			request:  execCredentialRequest{InstallationID: 300},
			wantName: "org-b",
		},
		{
			name:     "App and installation ID",
			request:  execCredentialRequest{AppID: 100, InstallationID: 200},
			wantName: "org-a",
		},
		{
			name:     "repository disambiguates App ID",
			request:  execCredentialRequest{Repository: "github.com/org-b/repo", AppID: 100},
			wantName: "org-b",
		},
		{
			name:        "ambiguous App ID",
			request:     execCredentialRequest{AppID: 100},
			wantErrText: "multiple GitHub App configurations match",
		},
		{
			name:        "unknown installation",
			request:     execCredentialRequest{InstallationID: 999},
			wantErrText: "no configured GitHub App matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := selectExecApp(cfg, tt.request)
			if tt.wantErrText != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrText) {
					t.Fatalf("selectExecApp() error = %v, want containing %q", err, tt.wantErrText)
				}
				return
			}
			if err != nil {
				t.Fatalf("selectExecApp() error = %v", err)
			}
			if app.Name != tt.wantName {
				t.Errorf("selected app = %q, want %q", app.Name, tt.wantName)
			}
		})
	}
}

func TestInferExecHost(t *testing.T) {
	tests := []struct {
		name        string
		patterns    []string
		want        string
		wantErrText string
	}{
		{
			name:     "same host",
			patterns: []string{"github.com/org-a/*", "https://github.com/org-b/*"},
			want:     gitHubAPIHost,
		},
		{
			name:        "multiple hosts",
			patterns:    []string{"github.com/org/*", "github.example.com/org/*"},
			wantErrText: "spans multiple hosts",
		},
		{
			name:        "no patterns",
			wantErrText: "has no host pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := inferExecHost(tt.patterns)
			if tt.wantErrText != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrText) {
					t.Fatalf("inferExecHost() error = %v, want containing %q", err, tt.wantErrText)
				}
				return
			}
			if err != nil {
				t.Fatalf("inferExecHost() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("inferExecHost() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExecEnvironment(t *testing.T) {
	t.Run("github.com repository", func(t *testing.T) {
		env := execEnvironment(
			[]string{
				"PATH=/usr/bin",
				"GH_TOKEN=old-gh-token",
				"GITHUB_TOKEN=old-github-token",
				"GH_ENTERPRISE_TOKEN=old-enterprise-token",
				"GITHUB_ENTERPRISE_TOKEN=old-github-enterprise-token",
				"GH_HOST=old.example.com",
				"GH_REPO=old/repo",
			},
			execCredential{
				Token:      "new-token",
				Host:       gitHubAPIHost,
				Repository: "github.com/myorg/myrepo",
			},
		)

		assertEnvironmentValue(t, env, "PATH", "/usr/bin")
		assertEnvironmentValue(t, env, "GH_TOKEN", "new-token")
		assertEnvironmentValue(t, env, "GH_HOST", gitHubAPIHost)
		assertEnvironmentValue(t, env, "GH_REPO", "github.com/myorg/myrepo")
		assertEnvironmentMissing(t, env, "GITHUB_TOKEN")
		assertEnvironmentMissing(t, env, "GH_ENTERPRISE_TOKEN")
		assertEnvironmentMissing(t, env, "GITHUB_ENTERPRISE_TOKEN")
	})

	t.Run("GitHub Enterprise Server without repository", func(t *testing.T) {
		env := execEnvironment(
			nil,
			execCredential{Token: "enterprise-token", Host: "github.example.com"},
		)

		assertEnvironmentValue(t, env, "GH_ENTERPRISE_TOKEN", "enterprise-token")
		assertEnvironmentValue(t, env, "GH_HOST", "github.example.com")
		assertEnvironmentMissing(t, env, "GH_TOKEN")
		assertEnvironmentMissing(t, env, "GH_REPO")
	})

	t.Run("GitHub Enterprise Cloud data residency", func(t *testing.T) {
		env := execEnvironment(
			nil,
			execCredential{Token: "cloud-token", Host: "octocorp.ghe.com"},
		)

		assertEnvironmentValue(t, env, "GH_TOKEN", "cloud-token")
		assertEnvironmentMissing(t, env, "GH_ENTERPRISE_TOKEN")
	})
}

func assertEnvironmentValue(t *testing.T, env []string, key, want string) {
	t.Helper()
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			if got := strings.TrimPrefix(entry, prefix); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
			return
		}
	}
	t.Errorf("%s is missing from environment", key)
}

func assertEnvironmentMissing(t *testing.T, env []string, key string) {
	t.Helper()
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			t.Errorf("%s unexpectedly present in environment", key)
			return
		}
	}
}

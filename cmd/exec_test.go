package cmd

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

func TestExecCommand(t *testing.T) {
	var (
		resolvedRepo string
		runName      string
		runArgs      []string
		runEnv       []string
	)

	cmd := newExecCmd(
		func(repo string) (string, error) {
			resolvedRepo = repo
			return "secret-token", nil
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
	if resolvedRepo != "github.com/myorg/myrepo" {
		t.Errorf("resolved repository = %q, want %q", resolvedRepo, "github.com/myorg/myrepo")
	}
	if runName != "gh" {
		t.Errorf("command name = %q, want %q", runName, "gh")
	}
	if got := strings.Join(runArgs, " "); got != "api repos/{owner}/{repo}" {
		t.Errorf("command args = %q, want %q", got, "api repos/{owner}/{repo}")
	}
	assertEnvironmentValue(t, runEnv, "GH_TOKEN", "secret-token")
	assertEnvironmentValue(t, runEnv, "GH_HOST", "github.com")
	assertEnvironmentValue(t, runEnv, "GH_REPO", "github.com/myorg/myrepo")
}

func TestExecCommandPropagatesErrors(t *testing.T) {
	t.Run("token resolution", func(t *testing.T) {
		wantErr := errors.New("token unavailable")
		cmd := newExecCmd(
			func(string) (string, error) { return "", wantErr },
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
			func(string) (string, error) { return "secret-token", nil },
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

func TestExecEnvironment(t *testing.T) {
	t.Run("github.com", func(t *testing.T) {
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
			repository.Repository{Host: "github.com", Owner: "myorg", Name: "myrepo"},
			"new-token",
		)

		assertEnvironmentValue(t, env, "PATH", "/usr/bin")
		assertEnvironmentValue(t, env, "GH_TOKEN", "new-token")
		assertEnvironmentValue(t, env, "GH_HOST", "github.com")
		assertEnvironmentValue(t, env, "GH_REPO", "github.com/myorg/myrepo")
		assertEnvironmentMissing(t, env, "GITHUB_TOKEN")
		assertEnvironmentMissing(t, env, "GH_ENTERPRISE_TOKEN")
		assertEnvironmentMissing(t, env, "GITHUB_ENTERPRISE_TOKEN")
	})

	t.Run("GitHub Enterprise Server", func(t *testing.T) {
		env := execEnvironment(
			nil,
			repository.Repository{Host: "github.example.com", Owner: "myorg", Name: "myrepo"},
			"enterprise-token",
		)

		assertEnvironmentValue(t, env, "GH_ENTERPRISE_TOKEN", "enterprise-token")
		assertEnvironmentValue(t, env, "GH_HOST", "github.example.com")
		assertEnvironmentMissing(t, env, "GH_TOKEN")
	})

	t.Run("GitHub Enterprise Cloud data residency", func(t *testing.T) {
		env := execEnvironment(
			nil,
			repository.Repository{Host: "octocorp.ghe.com", Owner: "myorg", Name: "myrepo"},
			"cloud-token",
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

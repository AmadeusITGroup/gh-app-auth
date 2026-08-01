package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/spf13/cobra"
)

type execTokenResolver func(string) (string, error)

type execCommandRunner func(
	context.Context,
	string,
	[]string,
	[]string,
	io.Reader,
	io.Writer,
	io.Writer,
) error

func NewExecCmd() *cobra.Command {
	return newExecCmd(resolveExecToken, runExecCommand)
}

func newExecCmd(resolveToken execTokenResolver, runCommand execCommandRunner) *cobra.Command {
	var repoFlag string

	cmd := &cobra.Command{
		Use:   "exec [--repo <repository>] -- <command> [args...]",
		Short: "Run a command with GitHub App authentication",
		Long: `Run a command with a short-lived token from the GitHub App or PAT
configured for a repository. The token is exposed only to the child process
through the environment and is never printed by gh-app-auth.`,
		Example: `  # Call the GitHub API for the current repository
  gh app-auth exec -- gh api repos/{owner}/{repo}

  # Run a gh command outside a repository
  gh app-auth exec --repo github.com/myorg/myrepo -- gh pr list`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoURL, err := determineRepositoryURL(repoFlag)
			if err != nil {
				return err
			}

			repo, err := repository.Parse(repoURL)
			if err != nil {
				return fmt.Errorf("failed to parse repository %q: %w", repoURL, err)
			}
			canonicalRepo := fmt.Sprintf("%s/%s/%s", repo.Host, repo.Owner, repo.Name)

			token, err := resolveToken(canonicalRepo)
			if err != nil {
				return err
			}

			env := execEnvironment(os.Environ(), repo, token)
			err = runCommand(
				cmd.Context(),
				args[0],
				args[1:],
				env,
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				cmd.ErrOrStderr(),
			)
			if err != nil {
				cmd.Root().SilenceErrors = true
				cmd.Root().SilenceUsage = true
			}
			return err
		},
	}

	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository to authenticate for (default: current repository)")

	return cmd
}

func resolveExecToken(repoURL string) (string, error) {
	cfg, err := loadCredentialConfig()
	if err != nil {
		return "", err
	}

	matchedApp, matchedPAT, err := findMatchingCredential(cfg, repoURL)
	if err != nil {
		return "", err
	}
	if matchedApp == nil && matchedPAT == nil {
		return "", fmt.Errorf("no credential configured for %s; run 'gh app-auth setup' first", repoURL)
	}

	if matchedPAT != nil {
		secretManager, err := newDefaultSecretsManager()
		if err != nil {
			return "", err
		}
		token, err := matchedPAT.GetPAT(secretManager)
		if err != nil {
			return "", fmt.Errorf("failed to get PAT: %w", err)
		}
		return token, nil
	}

	token, _, err := auth.NewAuthenticator().GetCredentials(matchedApp, repoURL)
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub App credentials: %w", err)
	}
	return token, nil
}

func execEnvironment(current []string, repo repository.Repository, token string) []string {
	blocked := map[string]struct{}{
		"GH_TOKEN":                {},
		"GITHUB_TOKEN":            {},
		"GH_ENTERPRISE_TOKEN":     {},
		"GITHUB_ENTERPRISE_TOKEN": {},
		"GH_HOST":                 {},
		"GH_REPO":                 {},
	}

	env := make([]string, 0, len(current)+3)
	for _, entry := range current {
		key, _, _ := strings.Cut(entry, "=")
		if _, found := blocked[key]; !found {
			env = append(env, entry)
		}
	}

	tokenVariable := "GH_ENTERPRISE_TOKEN"
	if repo.Host == "github.com" || strings.HasSuffix(repo.Host, ".ghe.com") {
		tokenVariable = "GH_TOKEN"
	}

	return append(
		env,
		tokenVariable+"="+token,
		"GH_HOST="+repo.Host,
		fmt.Sprintf("GH_REPO=%s/%s/%s", repo.Host, repo.Owner, repo.Name),
	)
}

func runExecCommand(
	ctx context.Context,
	name string,
	args []string,
	env []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) error {
	// #nosec G204 -- The user explicitly selects the command, which is executed directly without a shell.
	child := exec.CommandContext(ctx, name, args...)
	child.Env = env
	child.Stdin = stdin
	child.Stdout = stdout
	child.Stderr = stderr
	return child.Run()
}

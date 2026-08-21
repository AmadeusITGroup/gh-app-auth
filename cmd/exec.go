package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/auth"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/matcher"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/spf13/cobra"
)

type execCredentialRequest struct {
	Repository     string
	AppID          int64
	ClientID       string
	InstallationID int64
}

type execCredential struct {
	Token      string
	Host       string
	Repository string
}

type execCredentialResolver func(execCredentialRequest) (execCredential, error)

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
	return newExecCmd(resolveExecCredential, runExecCommand)
}

func newExecCmd(resolveCredential execCredentialResolver, runCommand execCommandRunner) *cobra.Command {
	var (
		repoFlag           string
		appIDFlag          int64
		installationIDFlag int64
	)

	cmd := &cobra.Command{
		Use:   "exec [flags] -- <command> [args...]",
		Short: "Run a command with managed GitHub authentication",
		Long: `Run a command with a short-lived token from a configured GitHub App
or PAT. Select credentials by repository, App ID, or installation ID. Explicit
App selectors must match configured entries, and --repo must match the selected
App route. The token is exposed only to the child process through the environment
and is never printed by gh-app-auth.`,
		Example: `  # Call the GitHub API for the current repository
  gh app-auth exec -- gh api repos/{owner}/{repo}

  # Run a gh command outside a repository
  gh app-auth exec --repo github.com/myorg/myrepo -- gh pr list

  # Run a repository-independent API command as a configured App installation
  gh app-auth exec --app-id 123456 --installation-id 789012 -- gh api /installation/repositories`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("app-id") && appIDFlag <= 0 {
				return fmt.Errorf("app ID must be positive")
			}
			if cmd.Flags().Changed("installation-id") && installationIDFlag <= 0 {
				return fmt.Errorf("installation ID must be positive")
			}

			request, err := newExecCredentialRequest(repoFlag, appIDFlag, installationIDFlag)
			if err != nil {
				return err
			}

			credential, err := resolveCredential(request)
			if err != nil {
				return err
			}

			env := execEnvironment(os.Environ(), credential)
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

	cmd.Flags().StringVarP(
		&repoFlag,
		"repo",
		"R",
		"",
		"Repository to authenticate for (default: current repository unless an ID selector is used)",
	)
	cmd.Flags().Int64Var(&appIDFlag, "app-id", 0, "Configured GitHub App ID to authenticate with")
	cmd.Flags().Int64Var(&installationIDFlag, "installation-id", 0, "GitHub App installation ID to authenticate with")

	return cmd
}

func newExecCredentialRequest(repoURL string, appID, installationID int64) (execCredentialRequest, error) {
	if appID < 0 {
		return execCredentialRequest{}, fmt.Errorf("app ID must be positive")
	}
	if installationID < 0 {
		return execCredentialRequest{}, fmt.Errorf("installation ID must be positive")
	}

	if repoURL == "" && appID == 0 && installationID == 0 {
		var err error
		repoURL, err = determineRepositoryURL("")
		if err != nil {
			return execCredentialRequest{}, err
		}
	}

	if repoURL != "" {
		repo, err := repository.Parse(repoURL)
		if err != nil {
			return execCredentialRequest{}, fmt.Errorf("failed to parse repository %q: %w", repoURL, err)
		}
		repoURL = fmt.Sprintf("%s/%s/%s", repo.Host, repo.Owner, repo.Name)
	}

	return execCredentialRequest{
		Repository:     repoURL,
		AppID:          appID,
		InstallationID: installationID,
	}, nil
}

func resolveExecCredential(request execCredentialRequest) (execCredential, error) {
	cfg, err := loadCredentialConfig()
	if err != nil {
		return execCredential{}, err
	}

	if request.AppID == 0 && request.ClientID == "" && request.InstallationID == 0 {
		return resolveRepositoryCredential(cfg, request.Repository)
	}

	selectedApp, err := selectExecApp(cfg, request)
	if err != nil {
		return execCredential{}, err
	}

	app := *selectedApp

	host, tokenTarget, err := execCredentialTarget(app, request.Repository)
	if err != nil {
		return execCredential{}, err
	}

	token, _, err := auth.NewAuthenticator().GetCredentials(&app, tokenTarget)
	if err != nil {
		return execCredential{}, fmt.Errorf("failed to get GitHub App credentials: %w", err)
	}

	return execCredential{Token: token, Host: host, Repository: request.Repository}, nil
}

func execCredentialTarget(app config.GitHubApp, repoURL string) (string, string, error) {
	if repoURL != "" {
		repo, err := repository.Parse(repoURL)
		if err != nil {
			return "", "", fmt.Errorf("failed to parse repository %q: %w", repoURL, err)
		}
		return repo.Host, repoURL, nil
	}

	if app.InstallationID == 0 {
		return "", "", fmt.Errorf("selected GitHub App has no installation ID; use --installation-id or --repo")
	}
	host, err := inferExecHost(app.Patterns)
	if err != nil {
		return "", "", err
	}
	return host, host, nil
}

func resolveRepositoryCredential(cfg *config.Config, repoURL string) (execCredential, error) {
	matchedApp, matchedPAT, err := findMatchingCredential(cfg, repoURL)
	if err != nil {
		return execCredential{}, err
	}
	if matchedApp == nil && matchedPAT == nil {
		return execCredential{}, fmt.Errorf("no credential configured for %s; run 'gh app-auth setup' first", repoURL)
	}

	repo, err := repository.Parse(repoURL)
	if err != nil {
		return execCredential{}, fmt.Errorf("failed to parse repository %q: %w", repoURL, err)
	}

	if matchedPAT != nil {
		secretManager, managerErr := newDefaultSecretsManager()
		if managerErr != nil {
			return execCredential{}, managerErr
		}
		token, tokenErr := matchedPAT.GetPAT(secretManager)
		if tokenErr != nil {
			return execCredential{}, fmt.Errorf("failed to get PAT: %w", tokenErr)
		}
		return execCredential{Token: token, Host: repo.Host, Repository: repoURL}, nil
	}

	token, _, err := auth.NewAuthenticator().GetCredentials(matchedApp, repoURL)
	if err != nil {
		return execCredential{}, fmt.Errorf("failed to get GitHub App credentials: %w", err)
	}
	return execCredential{Token: token, Host: repo.Host, Repository: repoURL}, nil
}

func selectExecApp(cfg *config.Config, request execCredentialRequest) (*config.GitHubApp, error) {
	candidates := execAppCandidates(cfg.GitHubApps, request)
	if len(candidates) == 0 {
		return nil, execAppNotFoundError(request)
	}

	if request.Repository != "" {
		matchedApp, err := matcher.NewMatcher(candidates).Match(request.Repository)
		if err != nil {
			return nil, fmt.Errorf("failed to match selected GitHub App to repository: %w", err)
		}
		if matchedApp == nil {
			return nil, execAppRepositoryNotFoundError(request)
		}
		return matchedApp, nil
	}

	if len(candidates) == 1 {
		return &candidates[0], nil
	}

	return nil, ambiguousExecAppError(request)
}

func execAppCandidates(apps []config.GitHubApp, request execCredentialRequest) []config.GitHubApp {
	return matchingExecApps(apps, request)
}

func matchingExecApps(apps []config.GitHubApp, request execCredentialRequest) []config.GitHubApp {
	candidates := make([]config.GitHubApp, 0, len(apps))
	for _, app := range apps {
		if request.AppID != 0 && app.AppID != request.AppID {
			continue
		}
		if request.ClientID != "" && app.ClientID != request.ClientID {
			continue
		}
		if request.InstallationID != 0 && app.InstallationID != request.InstallationID {
			continue
		}
		candidates = append(candidates, app)
	}
	return candidates
}

func execAppNotFoundError(request execCredentialRequest) error {
	switch {
	case request.ClientID != "" && request.InstallationID != 0:
		return fmt.Errorf(
			"no configured GitHub App matches client ID %s and installation ID %d",
			request.ClientID,
			request.InstallationID,
		)
	case request.AppID != 0 && request.InstallationID != 0:
		return fmt.Errorf(
			"no configured GitHub App matches app ID %d and installation ID %d",
			request.AppID,
			request.InstallationID,
		)
	case request.ClientID != "":
		return fmt.Errorf("no configured GitHub App matches client ID %s", request.ClientID)
	case request.AppID != 0:
		return fmt.Errorf("no configured GitHub App matches app ID %d", request.AppID)
	default:
		return fmt.Errorf("no configured GitHub App matches installation ID %d", request.InstallationID)
	}
}

func execAppRepositoryNotFoundError(request execCredentialRequest) error {
	selector := fmt.Sprintf("app ID %d", request.AppID)
	if request.ClientID != "" {
		selector = fmt.Sprintf("client ID %s", request.ClientID)
	}
	return fmt.Errorf("no configured GitHub App with %s matches repository %s", selector, request.Repository)
}

func ambiguousExecAppError(request execCredentialRequest) error {
	switch {
	case request.ClientID != "" && request.InstallationID != 0:
		return fmt.Errorf("multiple GitHub App configurations match; use --repo to disambiguate the host")
	case request.ClientID != "":
		return fmt.Errorf("multiple GitHub App configurations match; use --installation-id or --repo")
	case request.AppID != 0 && request.InstallationID != 0:
		return fmt.Errorf("multiple GitHub App configurations match; use --repo to disambiguate the host")
	case request.AppID != 0:
		return fmt.Errorf("multiple GitHub App configurations match; use --installation-id or --repo")
	default:
		return fmt.Errorf("multiple GitHub App configurations match; use --app-id or --repo")
	}
}

func inferExecHost(patterns []string) (string, error) {
	host := ""
	for _, pattern := range patterns {
		candidate := strings.TrimSpace(pattern)
		if candidate == "" {
			continue
		}

		info, err := matcher.GetRepositoryInfo(candidate)
		if err == nil {
			candidate = info.Host
		} else {
			candidate = strings.TrimPrefix(candidate, "https://")
			candidate = strings.TrimPrefix(candidate, "http://")
			candidate = strings.TrimSuffix(candidate, "/")
			if strings.Contains(candidate, "/") || strings.ContainsAny(candidate, "*") {
				return "", fmt.Errorf("cannot infer host from GitHub App pattern %q; use --repo", pattern)
			}
		}

		if host != "" && candidate != host {
			return "", fmt.Errorf("selected GitHub App spans multiple hosts; use --repo")
		}
		host = candidate
	}

	if host == "" {
		return "", fmt.Errorf("selected GitHub App has no host pattern; use --repo")
	}
	return host, nil
}

func execEnvironment(current []string, credential execCredential) []string {
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
	if credential.Host == gitHubAPIHost || strings.HasSuffix(credential.Host, ".ghe.com") {
		tokenVariable = "GH_TOKEN"
	}

	env = append(env, tokenVariable+"="+credential.Token, "GH_HOST="+credential.Host)
	if credential.Repository != "" {
		env = append(env, "GH_REPO="+credential.Repository)
	}
	return env
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

package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/spf13/cobra"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/auth"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

// tokenCredentialResolver resolves a configured GitHub App to a fresh token.
type tokenCredentialResolver func(execCredentialRequest) (string, error)

// NewTokenCmd creates the command that prints a fresh GitHub App installation token.
func NewTokenCmd() *cobra.Command {
	return newTokenCmd(resolveTokenCredential)
}

func newTokenCmd(resolveCredential tokenCredentialResolver) *cobra.Command {
	var (
		repoFlag           string
		appIDFlag          int64
		clientIDFlag       string
		installationIDFlag int64
	)

	cmd := &cobra.Command{
		Use:          "token",
		Short:        "Print a fresh GitHub App installation token",
		SilenceUsage: true,
		Long: `Print a fresh installation token for a configured GitHub App.

This command does not use the active GitHub CLI account. Select the configured App
explicitly with --app-id or --client-id and provide the repository that must match
its configured route. On success, stdout contains only the token followed by a newline.`,
		Example: `  # Print a fresh token for a configured App and repository
  gh app-auth token --app-id 123456 --repo github.com/myorg/myrepo

  # Use a Client ID and disambiguate a configured installation
  gh app-auth token \
    --client-id Iv1.AbCdEfGhIjKlMn \
    --installation-id 789012 \
    --repo github.com/myorg/myrepo`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("app-id") && appIDFlag <= 0 {
				return fmt.Errorf("app ID must be positive")
			}
			if cmd.Flags().Changed("installation-id") && installationIDFlag <= 0 {
				return fmt.Errorf("installation ID must be positive")
			}

			request, err := newTokenCredentialRequest(
				repoFlag, appIDFlag, clientIDFlag, installationIDFlag,
			)
			if err != nil {
				return err
			}

			token, err := resolveCredential(request)
			if err != nil {
				return err
			}
			if token == "" {
				return fmt.Errorf("token resolver returned an empty token")
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), token)
			return err
		},
	}

	cmd.Flags().StringVarP(
		&repoFlag,
		"repo",
		"R",
		"",
		"Repository to authenticate for (required; must match the configured App route)",
	)
	cmd.Flags().Int64Var(&appIDFlag, "app-id", 0, "Configured GitHub App ID to authenticate with")
	cmd.Flags().StringVar(&clientIDFlag, "client-id", "", "Configured GitHub App Client ID to authenticate with")
	cmd.Flags().Int64Var(
		&installationIDFlag,
		"installation-id",
		0,
		"Configured GitHub App installation ID to authenticate with",
	)

	return cmd
}

func newTokenCredentialRequest(
	repoURL string,
	appID int64,
	clientID string,
	installationID int64,
) (execCredentialRequest, error) {
	if appID < 0 {
		return execCredentialRequest{}, fmt.Errorf("app ID must be positive")
	}
	if installationID < 0 {
		return execCredentialRequest{}, fmt.Errorf("installation ID must be positive")
	}

	clientID = strings.TrimSpace(clientID)
	if appID == 0 && clientID == "" {
		return execCredentialRequest{}, fmt.Errorf("must specify either --app-id or --client-id")
	}
	if appID != 0 && clientID != "" {
		return execCredentialRequest{}, fmt.Errorf("cannot use both --app-id and --client-id")
	}

	canonicalRepo, err := canonicalTokenRepository(repoURL)
	if err != nil {
		return execCredentialRequest{}, err
	}

	return execCredentialRequest{
		Repository:     canonicalRepo,
		AppID:          appID,
		ClientID:       clientID,
		InstallationID: installationID,
	}, nil
}

func canonicalTokenRepository(repoURL string) (string, error) {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return "", fmt.Errorf("repository is required; use --repo")
	}
	if strings.HasPrefix(repoURL, "http://") {
		return "", fmt.Errorf("repository must use HTTPS or host/owner/repository format")
	}
	if strings.HasPrefix(repoURL, "git@") {
		return "", fmt.Errorf("SSH repository syntax is not supported; use host/owner/repository format")
	}
	if strings.ContainsAny(repoURL, "?#") {
		return "", fmt.Errorf("repository must not contain a query or fragment")
	}
	if strings.HasPrefix(repoURL, "https://") {
		parsedURL, err := url.Parse(repoURL)
		if err != nil {
			return "", fmt.Errorf("failed to parse repository %q: %w", repoURL, err)
		}
		if parsedURL.User != nil {
			return "", fmt.Errorf("repository must not contain embedded credentials")
		}
	} else {
		parts := strings.Split(strings.TrimSuffix(repoURL, ".git"), "/")
		if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
			return "", fmt.Errorf("repository must use host/owner/repository format")
		}
	}

	repo, err := repository.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse repository %q: %w", repoURL, err)
	}
	if repo.Host == "" || repo.Owner == "" || repo.Name == "" {
		return "", fmt.Errorf("repository must include host, owner, and repository")
	}

	return fmt.Sprintf("%s/%s/%s", repo.Host, repo.Owner, repo.Name), nil
}

type tokenMinter func(*config.GitHubApp, string) (string, error)

func resolveTokenCredential(request execCredentialRequest) (string, error) {
	return resolveTokenCredentialWith(request, func(app *config.GitHubApp, repoURL string) (string, error) {
		return auth.NewAuthenticator().MintInstallationToken(app, repoURL)
	})
}

func resolveTokenCredentialWith(request execCredentialRequest, mint tokenMinter) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load configuration: %w", err)
	}

	selectedApp, err := selectExecApp(cfg, request)
	if err != nil {
		return "", err
	}

	_, tokenTarget, err := execCredentialTarget(*selectedApp, request.Repository)
	if err != nil {
		return "", err
	}

	token, err := mint(selectedApp, tokenTarget)
	if err != nil {
		return "", fmt.Errorf("failed to mint GitHub App installation token: %w", err)
	}
	if strings.ContainsAny(token, "\r\n") {
		return "", fmt.Errorf("GitHub returned an invalid installation token")
	}

	return token, nil
}

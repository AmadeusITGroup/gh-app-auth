package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/auth"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/logger"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/matcher"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/secrets"
	"github.com/spf13/cobra"
)

var gitCredentialPattern string

func NewGitCredentialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "git-credential",
		Short: "Git credential helper for GitHub App authentication",
		Long: `Git credential helper that provides GitHub App authentication.

This command implements the git credential helper protocol and should not
be called directly. Instead, configure git to use it:

  git config credential.helper "app-auth git-credential"

Or for specific URL prefixes:

  git config credential."https://github.com/myorg".helper "app-auth git-credential --pattern 'https://github.com/myorg'"

The --pattern flag specifies a URL prefix to match against the repository URL.
This aligns with git's native credential helper URL scoping behavior.`,
		Hidden: true, // Hide from general help as it's internal
		Args:   cobra.ExactArgs(1),
		RunE:   gitCredentialRun,
	}

	cmd.Flags().StringVar(&gitCredentialPattern, "pattern", "", "URL prefix to match against repository URLs (e.g., 'https://github.com/myorg')")

	return cmd
}

func gitCredentialRun(cmd *cobra.Command, args []string) error {
	operation := args[0]

	logger.FlowStart("git_credential", map[string]interface{}{
		"operation": operation,
		"pattern":   gitCredentialPattern,
	})

	// Log all the input arguments - not just the operation
	logger.FlowStep("git_credential", map[string]interface{}{
		"args":    args,
		"pattern": gitCredentialPattern,
	})

	var err error
	switch operation {
	case "get":
		err = handleCredentialGet()
	case "store":
		err = handleCredentialStore()
	case "erase":
		err = handleCredentialErase()
	default:
		err = fmt.Errorf("unsupported git credential operation: %s", operation)
	}

	if err != nil {
		logger.FlowError("git_credential", err, map[string]interface{}{
			"operation": operation,
			"pattern":   gitCredentialPattern,
		})
	} else {
		logger.FlowSuccess("git_credential", map[string]interface{}{
			"operation": operation,
			"pattern":   gitCredentialPattern,
		})
	}

	return err
}

func handleCredentialGet() error {
	// Read and process input
	input, repoURL, err := processCredentialInput()
	if err != nil {
		return err
	}

	// Git credential protocol: When git queries with just host (no path),
	// we should exit silently. Git will call again with the full path.
	if input["path"] == "" {
		logger.FlowStep("host_only_query", map[string]interface{}{
			"host": input["host"],
			"note": "Exiting silently, git will query again with full path",
		})
		return nil
	}

	// Load configuration
	cfg, err := loadCredentialConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		return nil // Exit silently if no config
	}

	// Find matching credential provider (PAT or GitHub App)
	matchedApp, matchedPAT, err := findMatchingCredential(cfg, repoURL)
	if err != nil {
		return err
	}
	if matchedApp == nil && matchedPAT == nil {
		return nil // Exit silently if no match
	}

	// Generate and output credentials based on what matched
	if matchedPAT != nil {
		return generateAndOutputPATCredentials(matchedPAT)
	}
	return generateAndOutputCredentials(matchedApp, repoURL)
}

// processCredentialInput reads and processes git credential input
func processCredentialInput() (map[string]string, string, error) {
	logger.FlowStep("read_input", map[string]interface{}{})
	logger.FlowStep("git_credential_pattern", map[string]interface{}{
		"pattern": gitCredentialPattern,
	})

	// Read input from git
	input, err := readCredentialInput(os.Stdin)
	if err != nil {
		logger.FlowError("read_input", err, map[string]interface{}{})
		return nil, "", fmt.Errorf("failed to read credential input: %w", err)
	}

	// Sanitize input for logging (remove sensitive data)
	sanitizedInput := logger.SanitizeConfig(map[string]interface{}{
		"protocol": input["protocol"],
		"host":     input["host"],
		"path":     input["path"],
	})
	logger.FlowStep("parse_input", sanitizedInput)

	// Build repository URL from input
	repoURL := buildRepositoryURL(input)
	if repoURL == "" {
		logger.FlowError("build_url", fmt.Errorf("empty URL"), map[string]interface{}{
			"input": sanitizedInput,
		})
		return nil, "", fmt.Errorf("unable to determine repository URL from input")
	}

	logger.FlowStep("build_url", map[string]interface{}{
		"url": logger.SanitizeURL(repoURL),
	})

	return input, repoURL, nil
}

// loadCredentialConfig loads the GitHub App configuration
func loadCredentialConfig() (*config.Config, error) {
	cfg, err := config.LoadOrCreate()
	if err != nil {
		logger.FlowError("load_config", err, map[string]interface{}{})
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return cfg, nil
}

// findMatchingCredential finds the best matching credential provider (PAT or GitHub App) based on priority
func findMatchingCredential(
	cfg *config.Config, repoURL string,
) (*config.GitHubApp, *config.PersonalAccessToken, error) {
	// Find matching apps and PATs
	var matchedApps []*config.GitHubApp
	var matchedPATs []*config.PersonalAccessToken

	// Match GitHub Apps
	app := findAppByPattern(cfg, repoURL)
	if app != nil {
		matchedApps = append(matchedApps, app)
	} else {
		app, err := findAppByURL(cfg, repoURL)
		if err != nil {
			return nil, nil, err
		}
		if app != nil {
			matchedApps = append(matchedApps, app)
		} else {
			app, err = doAutomaticSetup(repoURL)
			if err != nil {
				return nil, nil, err
			}
			if app != nil {
				matchedApps = append(matchedApps, app)
			}
		}
	}

	// Match PATs
	for i := range cfg.PATs {
		pat := &cfg.PATs[i]
		for _, pattern := range pat.Patterns {
			if matchesPatternForPAT(pattern, repoURL) {
				matchedPATs = append(matchedPATs, pat)
				break
			}
		}
	}

	// No matches found
	if len(matchedApps) == 0 && len(matchedPATs) == 0 {
		return nil, nil, nil
	}

	// Determine best match by priority
	var bestApp *config.GitHubApp
	var bestPAT *config.PersonalAccessToken
	highestPriority := -1

	for _, app := range matchedApps {
		if app.Priority > highestPriority {
			highestPriority = app.Priority
			bestApp = app
			bestPAT = nil
		}
	}

	for _, pat := range matchedPATs {
		if pat.Priority > highestPriority {
			highestPriority = pat.Priority
			bestPAT = pat
			bestApp = nil
		}
	}

	return bestApp, bestPAT, nil
}

// matchesPatternForPAT checks if a PAT pattern matches the repository URL
func matchesPatternForPAT(pattern, repoURL string) bool {
	// Normalize both strings for comparison
	normalizedPattern := strings.TrimPrefix(strings.TrimSpace(pattern), "https://")
	normalizedURL := strings.TrimPrefix(strings.TrimSpace(repoURL), "https://")

	// Check if URL starts with pattern
	return strings.HasPrefix(normalizedURL, normalizedPattern)
}

// findAppByPattern finds an app using the --pattern flag
func findAppByPattern(cfg *config.Config, repoURL string) *config.GitHubApp {
	logger.FlowStep("match_by_pattern", map[string]interface{}{
		"pattern":  gitCredentialPattern,
		"repo_url": logger.SanitizeURL(repoURL),
	})

	// Normalize both pattern and URL for comparison (remove protocol)
	normalizedPattern := strings.TrimPrefix(strings.TrimPrefix(gitCredentialPattern, "https://"), "http://")
	normalizedURL := strings.TrimPrefix(strings.TrimPrefix(repoURL, "https://"), "http://")

	// Check if the pattern matches the repository URL
	patternMatches := strings.HasPrefix(normalizedURL, normalizedPattern) ||
		strings.HasPrefix(normalizedPattern, normalizedURL)
	if !patternMatches {
		logger.FlowStep("no_pattern_match", map[string]interface{}{
			"pattern":  gitCredentialPattern,
			"repo_url": logger.SanitizeURL(repoURL),
			"reason":   "URL prefix mismatch",
		})
		return nil
	}

	// Sort apps by pattern length in descending order
	sort.Slice(cfg.GitHubApps, func(i, j int) bool {
		return len(cfg.GitHubApps[i].Patterns) > len(cfg.GitHubApps[j].Patterns)
	})

	for i := range cfg.GitHubApps {
		app := &cfg.GitHubApps[i]
		logger.FlowStep("match_by_pattern", map[string]interface{}{
			"app_id":               app.AppID,
			"app_name":             app.Name,
			"pattern":              gitCredentialPattern,
			"repo_url":             logger.SanitizeURL(repoURL),
			"gitCredentialPattern": gitCredentialPattern,
		})

		for _, pattern := range app.Patterns {
			if matchesPattern(pattern, gitCredentialPattern) {
				logger.FlowStep("app_matched_by_pattern", map[string]interface{}{
					"app_id":   app.AppID,
					"app_name": app.Name,
					"pattern":  pattern,
					"repo_url": logger.SanitizeURL(repoURL),
				})
				return app
			}
		}
	}

	logger.FlowStep("no_pattern_match", map[string]interface{}{
		"pattern":  gitCredentialPattern,
		"repo_url": logger.SanitizeURL(repoURL),
		"reason":   "pattern not found",
	})
	return nil
}

// findAppByURL finds an app using URL-based matching
func findAppByURL(cfg *config.Config, repoURL string) (*config.GitHubApp, error) {
	logger.FlowStep("match_app", map[string]interface{}{
		"url": logger.SanitizeURL(repoURL),
	})

	m := matcher.NewMatcher(cfg.GitHubApps)
	matchedApp, err := m.Match(repoURL)

	if err != nil {
		// If URL doesn't have a path (e.g., just host), exit silently
		if strings.Contains(err.Error(), "no path found") {
			logger.FlowStep("no_path_exit", map[string]interface{}{
				"url":   logger.SanitizeURL(repoURL),
				"error": err.Error(),
			})
			return nil, nil
		}
		// Other errors should be reported
		logger.FlowError("match_app", err, map[string]interface{}{
			"url": logger.SanitizeURL(repoURL),
		})
		return nil, err
	}

	if matchedApp == nil {
		logger.FlowStep("no_match_exit", map[string]interface{}{
			"url": logger.SanitizeURL(repoURL),
		})
		return nil, nil
	}

	logger.FlowStep("app_matched", map[string]interface{}{
		"app_id":   matchedApp.AppID,
		"app_name": matchedApp.Name,
		"patterns": matchedApp.Patterns,
	})

	return matchedApp, nil
}

// matchesPattern checks if a pattern matches the git credential pattern
func matchesPattern(appPattern, gitCredPattern string) bool {
	logger.FlowStep("match_by_pattern", map[string]interface{}{
		"pattern":                    appPattern,
		"gitCredentialPattern":       gitCredPattern,
		"pattern_len":                len(appPattern),
		"gitCredentialPattern_len":   len(gitCredPattern),
		"pattern_bytes":              []byte(appPattern),
		"gitCredentialPattern_bytes": []byte(gitCredPattern),
		"exact_match":                appPattern == gitCredPattern,
		"prefix_match":               strings.HasPrefix(gitCredPattern, appPattern),
		"trimmed_pattern":            strings.TrimSpace(appPattern),
		"trimmed_gitCredPattern":     strings.TrimSpace(gitCredPattern),
		"trimmed_match":              strings.TrimSpace(appPattern) == strings.TrimSpace(gitCredPattern),
	})

	// Normalize both strings for comparison - remove protocol prefix
	normalizedAppPattern := strings.TrimPrefix(appPattern, "https://")
	normalizedPattern := strings.TrimPrefix(gitCredPattern, "https://")

	return strings.HasPrefix(normalizedPattern, normalizedAppPattern) || normalizedAppPattern == normalizedPattern
}

// doAutomaticSetup will automatically configure GitHub App if GH_APP_PRIVATE_KEY_PATH and GH_APP_ID are set
func doAutomaticSetup(repoURL string) (*config.GitHubApp, error) {
	if os.Getenv("GH_APP_PRIVATE_KEY_PATH") != "" && os.Getenv("GH_APP_ID") != "" {
		cfg, err := config.LoadOrCreate()
		if err != nil {
			return nil, err
		}
		appId, err := strconv.ParseInt(os.Getenv("GH_APP_ID"), 10, 64)
		if err != nil {
			return nil, err
		}
		keyFile := os.Getenv("GH_APP_PRIVATE_KEY_PATH")
		useFileSystem := true
		useKeyring := true
		silent := true
		priority := 5
		patterns := []string{repoURL}
		logger.FlowStep("automatic_setup", map[string]interface{}{
			"app_key":       keyFile,
			"app_id":        appId,
			"useFileSystem": useFileSystem,
			"keyFile":       keyFile,
			"useKeyring":    useKeyring,
			"patterns":      patterns,
		})
		return setupGitHubApp(
			cfg, appId, keyFile, "Auto setup", 0,
			patterns, priority, useKeyring, useFileSystem, silent,
		)
	}
	return nil, nil
}

// generateAndOutputPATCredentials generates PAT credentials and outputs them
func generateAndOutputPATCredentials(matchedPAT *config.PersonalAccessToken) error {
	logger.FlowStep("generate_pat_credentials", map[string]interface{}{
		"pat_name": matchedPAT.Name,
	})

	// Initialize secrets manager
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth")
	secretMgr := secrets.NewManager(configDir)

	// Retrieve PAT from secure storage
	token, err := matchedPAT.GetPAT(secretMgr)
	if err != nil {
		logger.FlowError("get_pat", err, map[string]interface{}{
			"pat_name": matchedPAT.Name,
		})
		return fmt.Errorf("failed to get PAT: %w", err)
	}

	logger.FlowStep("pat_retrieved", map[string]interface{}{
		"pat_name":     matchedPAT.Name,
		"token_hash":   logger.HashToken(token),
		"token_length": len(token),
	})

	// Determine username for HTTP basic auth
	// Default to "x-access-token" for GitHub, but allow custom username for other services (e.g., Bitbucket)
	username := matchedPAT.Username
	if username == "" {
		username = "x-access-token"
	}

	// Output credentials in git credential format
	fmt.Printf("username=%s\n", username)
	fmt.Printf("password=%s\n", token)

	logger.FlowStep("output_pat_credentials", map[string]interface{}{
		"pat_name":   matchedPAT.Name,
		"username":   username,
		"token_hash": logger.HashToken(token),
	})

	return nil
}

// generateAndOutputCredentials generates authentication credentials and outputs them
func generateAndOutputCredentials(matchedApp *config.GitHubApp, repoURL string) error {
	logger.FlowStep("generate_credentials", map[string]interface{}{
		"app_id": matchedApp.AppID,
	})

	authenticator := auth.NewAuthenticator()
	token, username, err := authenticator.GetCredentials(matchedApp, repoURL)
	if err != nil {
		logger.FlowError("generate_credentials", err, map[string]interface{}{
			"app_id": matchedApp.AppID,
		})
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	logger.FlowStep("credentials_generated", map[string]interface{}{
		"app_id":       matchedApp.AppID,
		"username":     username,
		"token_hash":   logger.HashToken(token),
		"token_length": len(token),
	})

	// Output credentials in git credential format
	fmt.Printf("username=%s\n", username)
	fmt.Printf("password=%s\n", token)

	logger.FlowStep("output_credentials", map[string]interface{}{
		"username":   username,
		"token_hash": logger.HashToken(token),
	})

	return nil
}

func handleCredentialStore() error {
	logger.FlowStep("store_read_input", map[string]interface{}{})

	// Read and ignore input - we don't need to store anything
	// since we generate tokens dynamically
	input, err := readCredentialInput(os.Stdin)
	if err != nil {
		logger.FlowError("store_read_input", err, map[string]interface{}{})
		return fmt.Errorf("failed to read credential input: %w", err)
	}

	// Sanitize input for logging
	sanitizedInput := logger.SanitizeConfig(map[string]interface{}{
		"protocol": input["protocol"],
		"host":     input["host"],
		"path":     input["path"],
		"username": input["username"],
	})
	logger.FlowStep("store_input_received", sanitizedInput)

	// Nothing to store for GitHub App authentication
	logger.FlowStep("store_noop", map[string]interface{}{
		"reason": "dynamic token generation",
	})
	return nil
}

func handleCredentialErase() error {
	logger.FlowStep("erase_read_input", map[string]interface{}{})

	// Read input to get the repository information
	input, err := readCredentialInput(os.Stdin)
	if err != nil {
		logger.FlowError("erase_read_input", err, map[string]interface{}{})
		return fmt.Errorf("failed to read credential input: %w", err)
	}

	// Sanitize input for logging
	sanitizedInput := logger.SanitizeConfig(map[string]interface{}{
		"protocol": input["protocol"],
		"host":     input["host"],
		"path":     input["path"],
	})
	logger.FlowStep("erase_input_received", sanitizedInput)

	// Build repository URL from input
	repoURL := buildRepositoryURL(input)
	if repoURL == "" {
		logger.FlowStep("erase_no_url", map[string]interface{}{
			"input": sanitizedInput,
		})
		return nil // Nothing to erase if we can't determine the repo
	}

	logger.FlowStep("erase_url_built", map[string]interface{}{
		"url": logger.SanitizeURL(repoURL),
	})

	// Clear any cached tokens for this repository
	// This is a placeholder - actual implementation would clear cache
	logger.FlowStep("erase_cache_clear", map[string]interface{}{
		"url":    logger.SanitizeURL(repoURL),
		"status": "placeholder",
	})
	return nil
}

func readCredentialInput(reader io.Reader) (map[string]string, error) {
	input := make(map[string]string)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break // Empty line marks end of input
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) < 2 {
			continue
		}

		key, value := parts[0], parts[1]

		// Handle URL format: git can send "url=https://github.com/owner/repo"
		// instead of separate protocol/host/path fields
		if key == "url" {
			u, err := url.Parse(value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse URL %q: %w", value, err)
			}
			input["protocol"] = u.Scheme
			input["host"] = u.Host
			input["path"] = strings.TrimPrefix(u.Path, "/")
			if u.User != nil {
				if username := u.User.Username(); username != "" {
					input["username"] = username
				}
				if password, ok := u.User.Password(); ok {
					input["password"] = password
				}
			}
		} else {
			input[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return input, nil
}

func buildRepositoryURL(input map[string]string) string {
	host := input["host"]
	path := input["path"]

	if host == "" {
		return ""
	}

	// Git sometimes queries without path first
	if path == "" {
		// Return host-only format for matching
		// This allows "github.com/org/*" patterns to match "github.com" queries
		return host
	}

	// Clean up path
	path = strings.Trim(path, "/")
	path = strings.TrimSuffix(path, ".git")

	// Return format: github.com/owner/repo
	return fmt.Sprintf("%s/%s", host, path)
}

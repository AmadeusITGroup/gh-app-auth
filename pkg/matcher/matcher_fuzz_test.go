package matcher

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

const maxFuzzSegmentInput = 128

func FuzzParseRepositoryURL(f *testing.F) {
	seeds := []string{
		"https://github.com/owner/repo",
		"http://github.com/owner/repo",
		"git@github.com:owner/repo.git",
		"github.com/owner/repo",
		"https://ghe.example.com/owner/repo.git",
		"https://gitlab.example.com/group/subgroup/repo",
		"https://github.com/owner/repo.git?ref=main#readme",
		"https://github.com/owner/%zz",
		"https://github.com/owner%2Frepo/name",
		"https://github.com/%2e%2e/repo",
		"",
		"github.com",
		"https://github.com",
		"https://github.com/owner/\xff\xfe",
		"https://github.com/owner/re\x00po",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		info, err := parseRepositoryURL(input)
		if err != nil {
			return
		}
		if info == nil {
			t.Fatalf("parseRepositoryURL(%q) returned nil info without error", input)
		}
		if info.Host == "" {
			t.Fatalf("parseRepositoryURL(%q) returned empty host", input)
		}
		wantFullPath := info.Host + "/" + info.Owner + "/" + info.Repository
		if info.FullPath != wantFullPath {
			t.Fatalf("parseRepositoryURL(%q) FullPath = %q, want %q", input, info.FullPath, wantFullPath)
		}
		if want := normalizeURL(input); info.URL != want {
			t.Fatalf("parseRepositoryURL(%q) URL = %q, want %q", input, info.URL, want)
		}

		again, err := parseRepositoryURL(info.URL)
		if err != nil {
			t.Fatalf("reparsing normalized URL %q failed: %v", info.URL, err)
		}
		if *again != *info {
			t.Fatalf("reparsing normalized URL %q = %+v, want %+v", info.URL, *again, *info)
		}
	})
}

func FuzzPrefixMatchesPath(f *testing.F) {
	seeds := [][2]string{
		{"github.com/owner/repo", "github.com/owner/repo"},
		{"github.com/owner/repo", "github.com/owner"},
		{"github.com/owner/repo", "github.com"},
		{"github.com/owner/repo", "github.com/own"},
		{"github.com/owner/repository", "github.com/owner/repo"},
		{"github.com/owner/repo/", "github.com/owner/repo"},
		{"github.com/owner/repo", "github.com/owner/repo/"},
		{"github.com/owner/repo", ""},
		{"", ""},
		{"", "github.com"},
		{"github.com/öwner/répo", "github.com/öwner"},
		{"github.com/\xff/repo", "github.com/\xff"},
		{"github.com/owner/repo", "github.com/\xff"},
	}
	for _, seed := range seeds {
		f.Add(seed[0], seed[1])
	}

	f.Fuzz(func(t *testing.T, path, prefix string) {
		want := path == prefix || strings.HasPrefix(path, prefix+"/")
		if got := prefixMatchesPath(path, prefix); got != want {
			t.Fatalf("prefixMatchesPath(%q, %q) = %v, want %v", path, prefix, got, want)
		}
	})
}

func FuzzMatcherLongestPrefix(f *testing.F) {
	addSegmentSeeds(f)

	f.Fuzz(func(t *testing.T, rawOwner, rawRepo string) {
		owner, repo := safeSegments(t, rawOwner, rawRepo)

		orgPatterns := []string{
			"github.com/" + owner,
			"github.com/" + owner + "/",
			"github.com/" + owner + "/*",
		}
		for _, orgPattern := range orgPatterns {
			apps := []config.GitHubApp{
				{Name: "fallback", Patterns: []string{"github.com"}},
				{Name: "org", Patterns: []string{orgPattern}},
				{Name: "repo", Patterns: []string{"github.com/" + owner + "/" + repo}},
			}
			orderings := map[string][]config.GitHubApp{
				"forward":  apps,
				"reversed": {apps[2], apps[1], apps[0]},
			}
			for ordering, ordered := range orderings {
				m := NewMatcher(ordered)
				cases := []struct {
					url  string
					want string
				}{
					{"https://github.com/" + owner + "/" + repo, "repo"},
					{"https://github.com/" + owner + "/" + repo + "-other", "org"},
					{"https://github.com/" + owner + "-other/" + repo, "fallback"},
				}
				for _, tc := range cases {
					app, err := m.Match(tc.url)
					if err != nil {
						t.Fatalf("%s %q: Match(%q) error: %v", ordering, orgPattern, tc.url, err)
					}
					if app == nil || app.Name != tc.want {
						t.Fatalf("%s %q: Match(%q) = %v, want app %q", ordering, orgPattern, tc.url, app, tc.want)
					}
				}
			}
		}
	})
}

func FuzzMatcherScope(f *testing.F) {
	addSegmentSeeds(f)

	f.Fuzz(func(t *testing.T, rawOwner, rawRepo string) {
		owner, repo := safeSegments(t, rawOwner, rawRepo)
		allowedURL := "https://github.com/" + owner + "/" + repo

		allScope := NewMatcher([]config.GitHubApp{{
			Name:     "all",
			Patterns: []string{"github.com"},
			Scope: &config.InstallationScope{
				RepositorySelection: "all",
				AccountLogin:        owner,
			},
		}})
		selectedScope := NewMatcher([]config.GitHubApp{{
			Name:     "selected",
			Patterns: []string{"github.com"},
			Scope: &config.InstallationScope{
				RepositorySelection: "selected",
				Repositories:        []config.RepositoryInfo{{FullName: owner + "/" + repo}},
			},
		}})

		cases := []struct {
			name    string
			matcher *Matcher
			url     string
			allowed bool
		}{
			{"all own repo", allScope, allowedURL, true},
			{"all other owner", allScope, "https://github.com/" + owner + "-other/" + repo, false},
			{"selected listed repo", selectedScope, allowedURL, true},
			{"selected other repo", selectedScope, "https://github.com/" + owner + "/" + repo + "-other", false},
			{"selected other owner", selectedScope, "https://github.com/" + owner + "-other/" + repo, false},
		}
		for _, tc := range cases {
			app, err := tc.matcher.Match(tc.url)
			if err != nil {
				t.Fatalf("%s: Match(%q) error: %v", tc.name, tc.url, err)
			}
			if tc.allowed && app == nil {
				t.Fatalf("%s: Match(%q) = nil, want a match", tc.name, tc.url)
			}
			if !tc.allowed && app != nil {
				t.Fatalf("%s: Match(%q) = %q, want no match", tc.name, tc.url, app.Name)
			}
		}
	})
}

func addSegmentSeeds(f *testing.F) {
	f.Helper()
	seeds := [][2]string{
		{"", ""},
		{"owner", "repo"},
		{"my-org", "my.repo_name"},
		{"owner/with/slash", "repo?query#frag"},
		{"%2e%2e", "re po"},
		{"öwner", "répo-日本"},
		{"\xff\xfe", "\x00\x01"},
	}
	for _, seed := range seeds {
		f.Add(seed[0], seed[1])
	}
}

func safeSegments(t *testing.T, rawOwner, rawRepo string) (string, string) {
	t.Helper()
	if len(rawOwner) > maxFuzzSegmentInput || len(rawRepo) > maxFuzzSegmentInput {
		t.Skip("input segment too large")
	}
	return "org-" + hex.EncodeToString([]byte(rawOwner)), "repo-" + hex.EncodeToString([]byte(rawRepo))
}

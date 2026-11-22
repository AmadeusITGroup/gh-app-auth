package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/AmadeusITGroup/gh-app-auth/pkg/config"
)

type sample struct {
	line    int
	body    string
	invalid bool
	marker  string
	indent  int
}

var fencePattern = regexp.MustCompile("^( {0,3})(`{3,}|~{3,})(.*)$")

const unknownFieldFixture = `version: "1.0"
pats:
  - name: Example
    token_source: keyring
    patterns:
      - github.com/example-org/
`

func fenceStart(line string, number int) *sample {
	match := fencePattern.FindStringSubmatch(line)
	if match == nil || (match[2][0] == '`' && strings.Contains(match[3], "`")) {
		return nil
	}
	current := &sample{line: -1, marker: match[2], indent: len(match[1])}
	info := strings.Fields(match[3])
	if len(info) == 0 || (info[0] != "yaml" && info[0] != "yml") {
		return current
	}
	current.line = number + 1
	current.invalid = len(info) > 1 && info[1] == "invalid"
	return current
}

func closesFence(line string, open *sample) bool {
	match := fencePattern.FindStringSubmatch(line)
	if match == nil {
		return false
	}
	return match[2][0] == open.marker[0] && len(match[2]) >= len(open.marker) && strings.Trim(match[3], " \t") == ""
}

func stripIndent(line string, width int) string {
	trimmed := 0
	for trimmed < width && trimmed < len(line) && line[trimmed] == ' ' {
		trimmed++
	}
	return line[trimmed:]
}

func extractSamples(markdown string) ([]sample, error) {
	var samples []sample
	var current *sample
	for number, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSuffix(line, "\r")
		switch {
		case current == nil:
			current = fenceStart(line, number)
		case closesFence(line, current):
			if current.line > 0 {
				samples = append(samples, *current)
			}
			current = nil
		default:
			current.body += stripIndent(line, current.indent) + "\n"
		}
	}
	if err := unterminated(current); err != nil {
		return nil, err
	}
	return samples, nil
}

func unterminated(current *sample) error {
	if current == nil {
		return nil
	}
	if current.line > 0 {
		return fmt.Errorf("line %d: unterminated yaml code block", current.line)
	}
	return errors.New("unterminated code block")
}

func validate(body string) (*config.Config, error) {
	decoder := yaml.NewDecoder(bytes.NewReader([]byte(body)))
	decoder.KnownFields(true)
	var cfg config.Config
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("decode: sample contains more than one YAML document")
		}
		return nil, fmt.Errorf("decode: trailing content after the first document: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	return &cfg, nil
}

type tally struct {
	appOnly, patOnly, mixed int
}

func (t *tally) count(cfg *config.Config) {
	switch {
	case len(cfg.GitHubApps) > 0 && len(cfg.PATs) > 0:
		t.mixed++
	case len(cfg.GitHubApps) > 0:
		t.appOnly++
	default:
		t.patOnly++
	}
}

func checkSamples(samples []sample) (tally, []string) {
	var counts tally
	var problems []string
	for _, s := range samples {
		cfg, err := validate(s.body)
		switch {
		case s.invalid && err == nil:
			problems = append(problems, fmt.Sprintf("line %d: sample marked invalid was accepted", s.line))
		case !s.invalid && err != nil:
			problems = append(problems, fmt.Sprintf("line %d: %v", s.line, err))
		case !s.invalid:
			counts.count(cfg)
		}
	}
	if counts.appOnly == 0 || counts.patOnly == 0 || counts.mixed == 0 {
		problems = append(problems, fmt.Sprintf("need valid App-only, PAT-only and mixed samples, found %+v", counts))
	}
	return counts, problems
}

func checkUnknownFieldFixture() []string {
	_, err := validate(unknownFieldFixture)
	if err == nil {
		return []string{"unknown token_source field fixture was accepted"}
	}
	if !strings.Contains(err.Error(), "token_source") {
		return []string{fmt.Sprintf("unknown field fixture failed for another reason: %v", err)}
	}
	return nil
}

func check(markdown string) (tally, []string) {
	samples, err := extractSamples(markdown)
	if err != nil {
		return tally{}, []string{err.Error()}
	}
	if len(samples) == 0 {
		return tally{}, []string{"no configuration samples found"}
	}
	counts, problems := checkSamples(samples)
	return counts, append(problems, checkUnknownFieldFixture()...)
}

const configurationReference = "docs/reference/configuration.md"

func run() error {
	content, err := os.ReadFile(configurationReference)
	if err != nil {
		return fmt.Errorf("read %s (run from the repository root): %w", configurationReference, err)
	}
	counts, problems := check(string(content))
	if len(problems) > 0 {
		return fmt.Errorf("configuration samples are invalid:\n- %s", strings.Join(problems, "\n- "))
	}
	fmt.Printf("Validated configuration samples: %d App, %d PAT, %d mixed\n", counts.appOnly, counts.patOnly, counts.mixed)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

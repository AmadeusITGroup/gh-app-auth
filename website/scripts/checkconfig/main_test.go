package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validDocument = "```yaml\nversion: \"1.0\"\ngithub_apps:\n  - name: App\n    client_id: Iv1.example\n" +
	"    private_key_source: keyring\n    patterns:\n      - github.com/example-org/\n```\n\n" +
	"```yaml\nversion: \"1.0\"\npats:\n  - name: PAT\n    private_key_source: keyring\n    patterns:\n" +
	"      - github.com/example-org/\n```\n\n" +
	"```yaml\nversion: \"1.0\"\ngithub_apps:\n  - name: App\n    app_id: 1\n    private_key_source: keyring\n" +
	"    patterns:\n      - github.com/a/\npats:\n  - name: PAT\n    patterns:\n      - github.com/b/\n```\n"

func TestExtractSamples(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     int
		wantErr  string
	}{
		{"no blocks", "text", 0, ""},
		{"fragment without version is still a sample", "```yaml\nname: x\n```\n", 1, ""},
		{"config block", "```yaml\nversion: \"1\"\n```\n", 1, ""},
		{"other languages ignored", "```json\nversion: 1\n```\n", 0, ""},
		{"text fences ignored", "```text\nversion: 1\n```\n", 0, ""},
		{"unlabelled fences ignored", "```\nplain\n```\n", 0, ""},
		{"three blocks", validDocument, 3, ""},
		{"unterminated yaml block", "```yaml\nversion: \"1\"\n", 0, "unterminated yaml"},
		{"unterminated other block", "```text\nx\n", 0, "unterminated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractSamples(tt.markdown)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("extractSamples() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("extractSamples() error = %v", err)
			}
			if len(got) != tt.want {
				t.Fatalf("extractSamples() = %d samples, want %d", len(got), tt.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			"valid app",
			"version: \"1.0\"\ngithub_apps:\n  - name: A\n    client_id: Iv1.x\n" +
				"    private_key_source: keyring\n    patterns: [github.com/o/]\n",
			"",
		},
		{"unknown field", unknownFieldFixture, "token_source"},
		{
			"filesystem source without path",
			"version: \"1.0\"\ngithub_apps:\n  - name: A\n    app_id: 1\n" +
				"    private_key_source: filesystem\n    patterns: [github.com/o/]\n",
			"private_key_path",
		},
		{"no credentials", "version: \"1.0\"\n", "at least one"},
		{"missing version", "pats:\n  - name: P\n    patterns: [x]\n", "version"},
		{
			"second document",
			"version: \"1.0\"\npats:\n  - name: P\n    patterns: [x]\n---\nversion: \"1.0\"\n",
			"more than one YAML document",
		},
		{"trailing garbage", "version: \"1.0\"\npats:\n  - name: P\n    patterns: [x]\n---\n: : :\n", "trailing content"},
		{"empty", "", "decode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validate(tt.body)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validate() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestCheck(t *testing.T) {
	if _, problems := check(validDocument); len(problems) != 0 {
		t.Fatalf("check(valid) problems = %v", problems)
	}
	if _, problems := check("nothing here"); len(problems) == 0 {
		t.Fatal("check(empty) should fail when no samples exist")
	}
	invalidMarked := validDocument + "\n```yaml invalid\nversion: \"1.0\"\ngithub_apps:\n  - name: A\n    app_id: 1\n" +
		"    private_key_source: keyring\n    patterns: [x]\n```\n"
	if _, problems := check(invalidMarked); len(problems) == 0 {
		t.Fatal("check should flag a sample marked invalid that validates")
	}
	missingVersion := validDocument + "\n```yaml\npats:\n  - name: P\n    patterns: [x]\n```\n"
	if _, problems := check(missingVersion); len(problems) == 0 {
		t.Fatal("check should flag a yaml block without a version")
	}
	if _, problems := check(validDocument + "\n```yaml\nversion: \"1.0\"\n"); len(problems) == 0 {
		t.Fatal("check should flag an unterminated yaml block")
	}
	rejected := validDocument + "\n```yaml invalid\nversion: \"1.0\"\npats:\n  - name: P\n" +
		"    token_source: keyring\n    patterns: [x]\n```\n"
	if _, problems := check(rejected); len(problems) != 0 {
		t.Fatalf("check(rejected) problems = %v", problems)
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		write    bool
		wantErr  string
	}{
		{"missing reference", "", false, "read " + configurationReference},
		{"valid reference", validDocument, true, ""},
		{"invalid reference", "nothing here", true, "configuration samples are invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if tt.write {
				if err := os.MkdirAll(filepath.Dir(configurationReference), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(configurationReference, []byte(tt.markdown), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			err := run()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("run() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("run() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestFenceStyles(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     int
		body     string
		wantErr  string
	}{
		{"tilde yaml", "~~~yaml\nversion: \"1\"\n~~~\n", 1, "version: \"1\"\n", ""},
		{"tilde yml", "~~~yml\nname: x\n~~~\n", 1, "name: x\n", ""},
		{
			"four backticks keep inner fence",
			"````yaml\nversion: \"1\"\n```\ninner: 1\n````\n",
			1,
			"version: \"1\"\n```\ninner: 1\n",
			"",
		},
		{"shorter closing does not close", "````yaml\na: 1\n```\n", 0, "", "unterminated yaml"},
		{"wrong character does not close", "```yaml\na: 1\n~~~\n", 0, "", "unterminated yaml"},
		{"closing with info does not close", "```yaml\na: 1\n```text\n", 0, "", "unterminated yaml"},
		{"longer closing is valid", "```yaml\na: 1\n`````\n", 1, "a: 1\n", ""},
		{"closing may have trailing whitespace", "```yaml\na: 1\n```  \t\n", 1, "a: 1\n", ""},
		{"indented body is deducted", "  ```yaml\n  a: 1\n    b: 2\n  ```\n", 1, "a: 1\n  b: 2\n", ""},
		{"ignored backtick block does not leak", "````text\n```yaml\na: 1\n```\n````\n", 0, "", ""},
		{"ignored tilde block does not leak", "~~~text\n```yaml\na: 1\n```\n~~~\n", 0, "", ""},
		{"backtick info with backtick is not a fence", "```yaml`x\na: 1\n", 0, "", ""},
		{"unterminated ignored long fence", "````text\n```\n", 0, "", "unterminated"},
		{"non-breaking space tail does not close", "```yaml\na: 1\n```\u00a0\n", 0, "", "unterminated yaml"},
		{"crlf backtick fence", "```yaml\r\na: 1\r\n```\r\n", 1, "a: 1\n", ""},
		{"crlf tilde fence", "~~~yaml\r\na: 1\r\n~~~\r\n", 1, "a: 1\n", ""},
		{"crlf four backticks keep inner fence", "````yaml\r\na: 1\r\n```\r\nb: 2\r\n````\r\n", 1, "a: 1\n```\nb: 2\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractSamples(tt.markdown)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("extractSamples() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("extractSamples() error = %v", err)
			}
			if len(got) != tt.want {
				t.Fatalf("extractSamples() = %d samples, want %d", len(got), tt.want)
			}
			if tt.want == 1 && got[0].body != tt.body {
				t.Fatalf("body = %q, want %q", got[0].body, tt.body)
			}
		})
	}
}

func TestCheckAcrossFenceStyles(t *testing.T) {
	for _, fence := range []string{"~~~", "````"} {
		bad := validDocument + "\n" + fence + "yaml\n" + unknownFieldFixture + fence + "\n"
		if _, problems := check(bad); len(problems) == 0 {
			t.Fatalf("check should flag an unknown field inside a %s fence", fence)
		}
		flagged := validDocument + "\n" + fence + "yaml invalid\nversion: \"1.0\"\ngithub_apps:\n  - name: A\n" +
			"    app_id: 1\n    private_key_source: keyring\n    patterns: [x]\n" + fence + "\n"
		if _, problems := check(flagged); len(problems) == 0 {
			t.Fatalf("check should flag a valid sample marked invalid inside a %s fence", fence)
		}
		rejected := validDocument + "\n" + fence + "yaml invalid\n" + unknownFieldFixture + fence + "\n"
		if _, problems := check(rejected); len(problems) != 0 {
			t.Fatalf("check(%s rejected) problems = %v", fence, problems)
		}
	}
}

func TestCheckWithCRLF(t *testing.T) {
	crlf := func(text string) string { return strings.ReplaceAll(text, "\n", "\r\n") }
	if _, problems := check(crlf(validDocument)); len(problems) != 0 {
		t.Fatalf("check(CRLF valid) problems = %v", problems)
	}
	for _, fence := range []string{"~~~", "````"} {
		bad := crlf(validDocument + "\n" + fence + "yaml\n" + unknownFieldFixture + fence + "\n")
		_, problems := check(bad)
		if len(problems) == 0 || !strings.Contains(strings.Join(problems, "\n"), "token_source") {
			t.Fatalf("check(CRLF %s unknown field) problems = %v, want token_source", fence, problems)
		}
	}
}

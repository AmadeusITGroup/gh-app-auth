package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const securityWorkflowPath = ".github/workflows/security.yml"

type securityWorkflow struct {
	On   map[string]yaml.Node       `yaml:"on"`
	Jobs map[string]securityScanJob `yaml:"jobs"`
}

type securityScanJob struct {
	Steps []securityScanStep `yaml:"steps"`
}

type securityScanStep struct {
	Name string            `yaml:"name"`
	Env  map[string]string `yaml:"env"`
	Run  string            `yaml:"run"`
}

func TestSecurityWorkflowSupportsManualDispatch(t *testing.T) {
	workflow := loadSecurityWorkflow(t)

	manualDispatch, ok := workflow.On["workflow_dispatch"]
	if !ok {
		t.Fatal("security workflow does not define the workflow_dispatch trigger")
	}
	isEmptyMapping := manualDispatch.Kind == yaml.MappingNode && len(manualDispatch.Content) == 0
	isNullValue := manualDispatch.Kind == yaml.ScalarNode && manualDispatch.Tag == "!!null"
	if !isEmptyMapping && !isNullValue {
		t.Fatalf("workflow_dispatch should not require inputs, got %#v", manualDispatch)
	}

	for _, existingTrigger := range []string{"push", "pull_request", "schedule"} {
		if _, ok := workflow.On[existingTrigger]; !ok {
			t.Errorf("security workflow lost its %q trigger", existingTrigger)
		}
	}
}

func TestSecurityWorkflowConfiguresNancyCredentials(t *testing.T) {
	step := loadNancyStep(t)

	wantEnv := map[string]string{
		"OSSI_TOKEN":    "${{ secrets.SONATATYPE_OSSI_TOKEN }}",
		"OSSI_USERNAME": "${{ vars.SONATATYPE_OSSI_USERNAME }}",
	}
	for variable, want := range wantEnv {
		if got := step.Env[variable]; got != want {
			t.Errorf("Run Nancy %s value = %q, want %q", variable, got, want)
		}
	}
}

func TestSecurityWorkflowNancyCredentialHandling(t *testing.T) {
	step := loadNancyStep(t)

	tests := []struct {
		name             string
		token            string
		username         string
		wantDiagnostics  []string
		rejectDiagnostic []string
	}{
		{
			name:            "both credentials missing",
			wantDiagnostics: []string{"OSSI_TOKEN is EMPTY", "OSSI_USERNAME is EMPTY"},
		},
		{
			name:             "token missing",
			username:         "test-user",
			wantDiagnostics:  []string{"OSSI_TOKEN is EMPTY"},
			rejectDiagnostic: []string{"OSSI_USERNAME is EMPTY"},
		},
		{
			name:             "username missing",
			token:            "test-token",
			wantDiagnostics:  []string{"OSSI_USERNAME is EMPTY"},
			rejectDiagnostic: []string{"OSSI_TOKEN is EMPTY"},
		},
		{
			name:             "both credentials present",
			token:            "test-token",
			username:         "test-user",
			rejectDiagnostic: []string{"OSSI_TOKEN is EMPTY", "OSSI_USERNAME is EMPTY"},
		},
		{
			name:             "credential values are shell safe",
			token:            `test token;$(echo should-not-run)`,
			username:         `test user "quoted"`,
			rejectDiagnostic: []string{"OSSI_TOKEN is EMPTY", "OSSI_USERNAME is EMPTY", "should-not-run\n"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, arguments, stdin := runNancyStep(t, step.Run, test.token, test.username)

			for _, diagnostic := range test.wantDiagnostics {
				if !strings.Contains(stdout, diagnostic) {
					t.Errorf("output %q does not contain %q", stdout, diagnostic)
				}
			}
			for _, diagnostic := range test.rejectDiagnostic {
				if strings.Contains(stdout, diagnostic) {
					t.Errorf("output %q unexpectedly contains %q", stdout, diagnostic)
				}
			}

			wantArguments := []string{"sleuth", "--token", test.token, "--username", test.username}
			if !reflect.DeepEqual(arguments, wantArguments) {
				t.Errorf("nancy arguments = %#v, want %#v", arguments, wantArguments)
			}
			if want := "{\"Path\":\"example.invalid/dependency\"}\n"; stdin != want {
				t.Errorf("nancy stdin = %q, want %q", stdin, want)
			}
			if test.token != "" && strings.Contains(stdout, test.token) {
				t.Error("workflow output exposed the token")
			}
			if test.username != "" && strings.Contains(stdout, test.username) {
				t.Error("workflow output exposed the username")
			}
		})
	}
}

func loadSecurityWorkflow(t *testing.T) securityWorkflow {
	t.Helper()

	contents, err := os.ReadFile(securityWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", securityWorkflowPath, err)
	}

	var workflow securityWorkflow
	if err := yaml.Unmarshal(contents, &workflow); err != nil {
		t.Fatalf("parse %s: %v", securityWorkflowPath, err)
	}
	return workflow
}

func loadNancyStep(t *testing.T) securityScanStep {
	t.Helper()

	workflow := loadSecurityWorkflow(t)
	nancyJob, ok := workflow.Jobs["nancy"]
	if !ok {
		t.Fatal("security workflow does not define the nancy job")
	}
	for _, step := range nancyJob.Steps {
		if step.Name == "Run Nancy" {
			return step
		}
	}
	t.Fatal("nancy job does not define a Run Nancy step")
	return securityScanStep{}
}

func runNancyStep(t *testing.T, script, token, username string) (string, []string, string) {
	t.Helper()

	tempDir := t.TempDir()
	argumentsPath := filepath.Join(tempDir, "nancy-arguments")
	stdinPath := filepath.Join(tempDir, "nancy-stdin")
	writeExecutable(t, filepath.Join(tempDir, "go"), `#!/bin/sh
if [ "$1" != "list" ] || [ "$2" != "-json" ] || [ "$3" != "-deps" ] || [ "$4" != "./..." ]; then
  echo "unexpected go arguments: $*" >&2
  exit 2
fi
printf '{"Path":"example.invalid/dependency"}\n'
`)
	writeExecutable(t, filepath.Join(tempDir, "nancy"), `#!/bin/sh
printf '%s\0' "$@" > "$NANCY_ARGUMENTS_PATH"
cat > "$NANCY_STDIN_PATH"
`)

	command := exec.Command("bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", script)
	command.Env = []string{
		"PATH=" + tempDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"OSSI_TOKEN=" + token,
		"OSSI_USERNAME=" + username,
		"NANCY_ARGUMENTS_PATH=" + argumentsPath,
		"NANCY_STDIN_PATH=" + stdinPath,
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run Nancy workflow step: %v\n%s", err, output)
	}

	argumentBytes, err := os.ReadFile(argumentsPath)
	if err != nil {
		t.Fatalf("read captured Nancy arguments: %v", err)
	}
	stdinBytes, err := os.ReadFile(stdinPath)
	if err != nil {
		t.Fatalf("read captured Nancy stdin: %v", err)
	}

	arguments := bytes.Split(bytes.TrimSuffix(argumentBytes, []byte{0}), []byte{0})
	argumentStrings := make([]string, len(arguments))
	for index, argument := range arguments {
		argumentStrings[index] = string(argument)
	}
	return string(output), argumentStrings, string(stdinBytes)
}

func writeExecutable(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

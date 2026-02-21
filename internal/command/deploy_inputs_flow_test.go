// Where: cli/internal/command/deploy_inputs_flow_test.go
// What: Flow-oriented unit tests for deploy input resolution helpers.
// Why: Keep stack-first env resolution behavior stable for interactive deploy UX.
package command

import (
	"path/filepath"
	"testing"

	"github.com/poruru-code/esb-cli/internal/constants"
	runtimeinfra "github.com/poruru-code/esb-cli/internal/infra/runtime"
)

func TestResolveDeployEnvFromStackNoPromptSources(t *testing.T) {
	tests := []struct {
		name       string
		flagEnv    string
		stack      deployTargetStack
		project    string
		wantValue  string
		wantSource string
	}{
		{
			name:       "prefers flag",
			flagEnv:    "prod",
			stack:      deployTargetStack{Name: "esb-dev", Env: "dev"},
			project:    "esb3",
			wantValue:  "prod",
			wantSource: "flag",
		},
		{
			name:       "uses stack env",
			stack:      deployTargetStack{Name: "esb-dev", Env: "dev"},
			wantValue:  "dev",
			wantSource: "stack",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, prompter := mustResolveDeployEnvFromStack(
				t,
				tc.flagEnv,
				tc.stack,
				tc.project,
				nil,
				"ignored",
			)
			if got.Value != tc.wantValue {
				t.Fatalf("expected env %s, got %q", tc.wantValue, got.Value)
			}
			if got.Source != tc.wantSource {
				t.Fatalf("expected source %s, got %q", tc.wantSource, got.Source)
			}
			assertNoPromptCall(t, prompter)
		})
	}
}

func TestResolveDeployEnvFromStackFallsBackToPrompt(t *testing.T) {
	got, prompter := mustResolveDeployEnvFromStack(
		t,
		"",
		deployTargetStack{},
		"",
		nil,
		"staging",
	)
	if got.Value != "staging" {
		t.Fatalf("expected env staging, got %q", got.Value)
	}
	if got.Source != "prompt" {
		t.Fatalf("expected source prompt, got %q", got.Source)
	}
	if prompter.inputCalls != 1 {
		t.Fatalf("expected one prompt call, got %d", prompter.inputCalls)
	}
}

func TestResolveDeployEnvFromStackUsesRuntimeResolver(t *testing.T) {
	got, prompter := mustResolveDeployEnvFromStack(
		t,
		"",
		deployTargetStack{},
		"esb-dev",
		fixedEnvResolver{
			inferred: runtimeinfra.EnvInference{Env: "qa", Source: "container label"},
		},
		"ignored",
	)
	if got.Value != "qa" {
		t.Fatalf("expected env qa, got %q", got.Value)
	}
	if got.Source != "container label" {
		t.Fatalf("expected source container label, got %q", got.Source)
	}
	assertNoPromptCall(t, prompter)
}

func TestResolveDeployArtifactRootInteractiveUsesPreviousDefault(t *testing.T) {
	repoRoot := t.TempDir()
	prompter := &recordingPrompter{inputValue: ""}
	got, err := resolveDeployArtifactRoot(
		"",
		true,
		prompter,
		filepath.Join(repoRoot, "artifacts", "custom"),
		repoRoot,
		"esb-dev",
		"dev",
	)
	if err != nil {
		t.Fatalf("resolve deploy artifact root: %v", err)
	}
	want := filepath.Join(repoRoot, "artifacts", "custom")
	if got != want {
		t.Fatalf("expected previous artifact root %q, got %q", want, got)
	}
	if prompter.inputCalls != 1 {
		t.Fatalf("expected one prompt call, got %d", prompter.inputCalls)
	}
}

func TestResolveDeployArtifactRootInteractiveUsesDefaultLayout(t *testing.T) {
	repoRoot := t.TempDir()
	prompter := &recordingPrompter{inputValue: ""}
	got, err := resolveDeployArtifactRoot(
		"",
		true,
		prompter,
		"",
		repoRoot,
		"esb-dev",
		"dev",
	)
	if err != nil {
		t.Fatalf("resolve deploy artifact root: %v", err)
	}
	want := filepath.Join(repoRoot, "artifacts", "esb-dev")
	if got != want {
		t.Fatalf("expected default artifact root %q, got %q", want, got)
	}
	if prompter.inputCalls != 1 {
		t.Fatalf("expected one prompt call, got %d", prompter.inputCalls)
	}
}

func TestResolveDeployArtifactRootWithoutTTYUsesDefault(t *testing.T) {
	repoRoot := t.TempDir()
	got, err := resolveDeployArtifactRoot(
		"",
		false,
		nil,
		"",
		repoRoot,
		"esb-dev",
		"dev",
	)
	if err != nil {
		t.Fatalf("resolve deploy artifact root: %v", err)
	}
	want := filepath.Join(repoRoot, "artifacts", "esb-dev")
	if got != want {
		t.Fatalf("expected default artifact root %q, got %q", want, got)
	}
}

func TestResolveDeployArtifactRootWithoutTTYUsesProjectEnvScope(t *testing.T) {
	repoRoot := t.TempDir()
	got, err := resolveDeployArtifactRoot(
		"",
		false,
		nil,
		"",
		repoRoot,
		"esb",
		"dev",
	)
	if err != nil {
		t.Fatalf("resolve deploy artifact root: %v", err)
	}
	want := filepath.Join(repoRoot, "artifacts", "esb-dev")
	if got != want {
		t.Fatalf("expected default artifact root %q, got %q", want, got)
	}
}

func TestResolveDeployProjectInteractiveUsesDefault(t *testing.T) {
	prompter := &recordingPrompter{inputValue: ""}
	got, source, err := resolveDeployProject("esb-dev", true, prompter, "", nil)
	if err != nil {
		t.Fatalf("resolve deploy project: %v", err)
	}
	if got != "esb-dev" || source != "default" {
		t.Fatalf("unexpected project/source: got=(%q,%q)", got, source)
	}
	if prompter.inputCalls != 1 {
		t.Fatalf("expected one prompt call, got %d", prompter.inputCalls)
	}
}

func TestResolveDeployProjectInteractiveUsesPreviousDefault(t *testing.T) {
	prompter := &recordingPrompter{inputValue: ""}
	got, source, err := resolveDeployProject("esb-dev", true, prompter, "esb-prev", nil)
	if err != nil {
		t.Fatalf("resolve deploy project: %v", err)
	}
	if got != "esb-prev" || source != "previous" {
		t.Fatalf("unexpected project/source: got=(%q,%q)", got, source)
	}
	if prompter.inputCalls != 1 {
		t.Fatalf("expected one prompt call, got %d", prompter.inputCalls)
	}
}

func TestResolveDeployComposeFilesDefaultsToAutoWithoutPrompt(t *testing.T) {
	prompter := &recordingPrompter{inputValue: ""}
	got, err := resolveDeployComposeFiles(nil, true, prompter, nil, "/tmp")
	if err != nil {
		t.Fatalf("resolve compose files: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil compose files for auto, got %v", got)
	}
	if prompter.inputCalls != 0 {
		t.Fatalf("expected no prompt calls, got %d", prompter.inputCalls)
	}
}

func TestResolveDeployComposeFilesUsesPreviousWithoutPrompt(t *testing.T) {
	baseDir := t.TempDir()
	previous := []string{"docker-compose.yml", "./docker-compose.override.yml"}
	prompter := &recordingPrompter{inputValue: ""}
	got, err := resolveDeployComposeFiles(nil, true, prompter, previous, baseDir)
	if err != nil {
		t.Fatalf("resolve compose files: %v", err)
	}
	want := normalizeComposeFiles(previous, baseDir)
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected compose files: got=%v want=%v", got, want)
	}
	if prompter.inputCalls != 0 {
		t.Fatalf("expected no prompt calls, got %d", prompter.inputCalls)
	}
}

func TestResolveDeployComposeFilesUsesExplicitValuesWithoutPrompt(t *testing.T) {
	baseDir := t.TempDir()
	prompter := &recordingPrompter{inputValue: ""}
	got, err := resolveDeployComposeFiles(
		[]string{"docker-compose.yml", "docker-compose.prod.yml"},
		true,
		prompter,
		nil,
		baseDir,
	)
	if err != nil {
		t.Fatalf("resolve compose files: %v", err)
	}
	want := normalizeComposeFiles(
		[]string{"docker-compose.yml", "docker-compose.prod.yml"},
		baseDir,
	)
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected compose files: got=%v want=%v", got, want)
	}
	if prompter.inputCalls != 0 {
		t.Fatalf("expected no prompt calls, got %d", prompter.inputCalls)
	}
}

func TestResolveProjectValue(t *testing.T) {
	tests := []struct {
		name         string
		flagProject  string
		envProject   string
		hostPrefix   string
		hostProject  string
		wantValue    string
		wantSource   string
		wantExplicit bool
	}{
		{
			name:         "flag has highest priority",
			flagProject:  "from-flag",
			envProject:   "from-env",
			hostPrefix:   "ESB",
			hostProject:  "from-host",
			wantValue:    "from-flag",
			wantSource:   "flag",
			wantExplicit: true,
		},
		{
			name:         "env is used when flag is empty",
			envProject:   "from-env",
			hostPrefix:   "ESB",
			hostProject:  "from-host",
			wantValue:    "from-env",
			wantSource:   "env",
			wantExplicit: true,
		},
		{
			name:         "host env fallback",
			hostPrefix:   "ESB",
			hostProject:  "from-host",
			wantValue:    "from-host",
			wantSource:   "host",
			wantExplicit: true,
		},
		{
			name:         "returns non-explicit when nothing is set",
			wantValue:    "",
			wantSource:   "",
			wantExplicit: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(constants.EnvProjectName, tc.envProject)
			t.Setenv("ENV_PREFIX", tc.hostPrefix)
			if tc.hostPrefix != "" {
				t.Setenv(tc.hostPrefix+"_"+constants.HostSuffixProject, tc.hostProject)
			}
			value, source, explicit := resolveProjectValue(tc.flagProject)
			if value != tc.wantValue || source != tc.wantSource || explicit != tc.wantExplicit {
				t.Fatalf(
					"unexpected result: got=(%q,%q,%v) want=(%q,%q,%v)",
					value,
					source,
					explicit,
					tc.wantValue,
					tc.wantSource,
					tc.wantExplicit,
				)
			}
		})
	}
}

func TestResolveComposeProjectValue(t *testing.T) {
	tests := []struct {
		name          string
		projectValue  string
		projectSource string
		stack         deployTargetStack
		flagEnv       string
		prevEnv       string
		wantProject   string
		wantSource    string
	}{
		{
			name:          "explicit project",
			projectValue:  "esb-explicit",
			projectSource: "flag",
			stack:         deployTargetStack{Project: "esb-stack", Env: "dev"},
			flagEnv:       "prod",
			prevEnv:       "staging",
			wantProject:   "esb-explicit",
			wantSource:    "flag",
		},
		{
			name:        "stack project",
			stack:       deployTargetStack{Project: "esb-stack", Env: "dev"},
			flagEnv:     "prod",
			prevEnv:     "staging",
			wantProject: "esb-stack",
			wantSource:  "stack",
		},
		{
			name:        "default from flag env",
			stack:       deployTargetStack{},
			flagEnv:     "prod",
			prevEnv:     "staging",
			wantProject: "esb-prod",
			wantSource:  "default",
		},
		{
			name:        "default from previous env",
			stack:       deployTargetStack{},
			flagEnv:     "",
			prevEnv:     "staging",
			wantProject: "esb-staging",
			wantSource:  "default",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotProject, gotSource := resolveComposeProjectValue(
				tc.projectValue,
				tc.projectSource,
				tc.stack,
				tc.flagEnv,
				tc.prevEnv,
			)
			if gotProject != tc.wantProject || gotSource != tc.wantSource {
				t.Fatalf(
					"unexpected project/source: got=(%q,%q) want=(%q,%q)",
					gotProject,
					gotSource,
					tc.wantProject,
					tc.wantSource,
				)
			}
		})
	}
}

func mustResolveDeployEnvFromStack(
	t *testing.T,
	flagEnv string,
	stack deployTargetStack,
	project string,
	resolver runtimeinfra.EnvResolver,
	inputValue string,
) (envChoice, *recordingPrompter) {
	t.Helper()
	prompter := &recordingPrompter{inputValue: inputValue}
	got, err := resolveDeployEnvFromStack(
		flagEnv,
		stack,
		project,
		true,
		prompter,
		resolver,
		"default",
	)
	if err != nil {
		t.Fatalf("resolve env from stack: %v", err)
	}
	return got, prompter
}

func assertNoPromptCall(t *testing.T, prompter *recordingPrompter) {
	t.Helper()
	if prompter.inputCalls != 0 {
		t.Fatalf("expected no prompt call, got %d", prompter.inputCalls)
	}
}

// Where: cli/internal/command/deploy_inputs_flow_test.go
// What: Flow-oriented unit tests for deploy input resolution helpers.
// Why: Keep stack-first env resolution behavior stable for interactive deploy UX.
package command

import (
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

func TestResolveDeployOutputInteractiveUsesPreviousWithoutPrompt(t *testing.T) {
	assertResolveDeployOutputNoPrompt(t, ".esb/custom", ".esb/custom")
}

func TestResolveDeployOutputInteractiveUsesDefaultWithoutPrompt(t *testing.T) {
	assertResolveDeployOutputNoPrompt(t, "", "")
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

func assertResolveDeployOutputNoPrompt(t *testing.T, previous, want string) {
	t.Helper()
	prompter := &recordingPrompter{inputValue: "should-not-be-used"}
	got, err := resolveDeployOutput(
		"",
		true,
		prompter,
		previous,
	)
	if err != nil {
		t.Fatalf("resolve deploy output: %v", err)
	}
	if got != want {
		t.Fatalf("expected output %q, got %q", want, got)
	}
	assertNoPromptCall(t, prompter)
}

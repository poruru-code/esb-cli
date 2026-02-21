// Where: cli/internal/command/deploy_inputs_project_test.go
// What: Tests for deploy project prompt resolution branches.
// Why: Cover non-TTY, prompt failure, and manual selection behaviors.
package command

import (
	"errors"
	"testing"

	"github.com/poruru-code/esb-cli/internal/infra/interaction"
)

type projectInputPrompter struct {
	inputValue string
	inputErr   error
	inputCalls int
}

func (p *projectInputPrompter) Input(_ string, _ []string) (string, error) {
	p.inputCalls++
	if p.inputErr != nil {
		return "", p.inputErr
	}
	return p.inputValue, nil
}

func (*projectInputPrompter) Select(_ string, _ []string) (string, error) {
	return "", nil
}

func (*projectInputPrompter) SelectValue(_ string, _ []interaction.SelectOption) (string, error) {
	return "", nil
}

func TestResolveDeployProjectRequiresDefaultProject(t *testing.T) {
	_, _, err := resolveDeployProject("", true, &projectInputPrompter{}, "", nil)
	if !errors.Is(err, errComposeProjectRequired) {
		t.Fatalf("expected errComposeProjectRequired, got %v", err)
	}
}

func TestResolveDeployProjectNonTTYUsesDefault(t *testing.T) {
	got, source, err := resolveDeployProject("esb-dev", false, nil, "", nil)
	if err != nil {
		t.Fatalf("resolve deploy project: %v", err)
	}
	if got != "esb-dev" || source != "default" {
		t.Fatalf("unexpected project/source: got=(%q,%q)", got, source)
	}
}

func TestResolveDeployProjectInteractiveReturnsPromptSourceForManualInput(t *testing.T) {
	prompter := &projectInputPrompter{inputValue: "esb-manual"}
	got, source, err := resolveDeployProject("esb-dev", true, prompter, "", nil)
	if err != nil {
		t.Fatalf("resolve deploy project: %v", err)
	}
	if got != "esb-manual" || source != "prompt" {
		t.Fatalf("unexpected project/source: got=(%q,%q)", got, source)
	}
	if prompter.inputCalls != 1 {
		t.Fatalf("expected one input call, got %d", prompter.inputCalls)
	}
}

func TestResolveDeployProjectInteractiveReturnsPromptError(t *testing.T) {
	prompter := &projectInputPrompter{inputErr: errors.New("boom")}
	_, _, err := resolveDeployProject("esb-dev", true, prompter, "", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "prompt compose project: boom" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReconcileDeployProjectWithEnv(t *testing.T) {
	tests := []struct {
		name        string
		current     string
		source      string
		env         string
		wantProject string
		wantSource  string
	}{
		{
			name:        "default source follows env",
			current:     "esb-default",
			source:      "default",
			env:         "dev",
			wantProject: "esb-dev",
			wantSource:  "default",
		},
		{
			name:        "prompt source keeps manual project",
			current:     "custom-project",
			source:      "prompt",
			env:         "dev",
			wantProject: "custom-project",
			wantSource:  "prompt",
		},
		{
			name:        "previous source keeps previous project",
			current:     "esb-prev",
			source:      "previous",
			env:         "prod",
			wantProject: "esb-prev",
			wantSource:  "previous",
		},
		{
			name:        "empty env keeps current default project",
			current:     "esb-default",
			source:      "default",
			env:         "",
			wantProject: "esb-default",
			wantSource:  "default",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotProject, gotSource := reconcileDeployProjectWithEnv(tc.current, tc.source, tc.env)
			if gotProject != tc.wantProject || gotSource != tc.wantSource {
				t.Fatalf(
					"unexpected result: got=(%q,%q) want=(%q,%q)",
					gotProject,
					gotSource,
					tc.wantProject,
					tc.wantSource,
				)
			}
		})
	}
}

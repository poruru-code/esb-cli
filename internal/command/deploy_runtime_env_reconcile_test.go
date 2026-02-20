// Where: cli/internal/command/deploy_runtime_env_reconcile_test.go
// What: Unit tests for deploy env reconciliation with runtime inference.
// Why: Freeze mismatch resolution behavior before further refactors.
package command

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/poruru-code/esb-cli/internal/infra/interaction"
	runtimeinfra "github.com/poruru-code/esb-cli/internal/infra/runtime"
)

type fixedEnvResolver struct {
	inferred runtimeinfra.EnvInference
	err      error
}

func (r fixedEnvResolver) InferEnvFromProject(_, _ string) (runtimeinfra.EnvInference, error) {
	return r.inferred, r.err
}

type selectOnlyPrompter struct {
	selected string
}

func (p selectOnlyPrompter) Input(_ string, _ []string) (string, error) {
	return "", nil
}

func (p selectOnlyPrompter) Select(_ string, _ []string) (string, error) {
	return "", nil
}

func (p selectOnlyPrompter) SelectValue(_ string, _ []interaction.SelectOption) (string, error) {
	return p.selected, nil
}

func TestReconcileEnvWithRuntimeKeepsExplicitWhenForceEnabled(t *testing.T) {
	choice := envChoice{Value: "prod", Source: "flag", Explicit: true}
	assertReconcileKeepsChoice(
		t,
		choice,
		"esb-dev",
		runtimeinfra.EnvInference{Env: "dev", Source: "container label"},
		true,
	)
}

func TestReconcileEnvWithRuntimeAlignsImplicitChoice(t *testing.T) {
	got, err := reconcileEnvWithRuntime(
		envChoice{Value: "prod", Source: "default", Explicit: false},
		"esb-dev",
		"template.yaml",
		false,
		nil,
		fixedEnvResolver{inferred: runtimeinfra.EnvInference{Env: "dev", Source: "gateway env"}},
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("reconcile env: %v", err)
	}
	if got.Value != "dev" {
		t.Fatalf("expected aligned env 'dev', got %q", got.Value)
	}
	if got.Source != "gateway env" {
		t.Fatalf("expected source 'gateway env', got %q", got.Source)
	}
	if got.Explicit {
		t.Fatalf("expected implicit choice, got explicit=true")
	}
}

func TestReconcileEnvWithRuntimeErrorsForExplicitMismatchWithoutTTY(t *testing.T) {
	_, err := reconcileEnvWithRuntime(
		envChoice{Value: "prod", Source: "flag", Explicit: true},
		"esb-dev",
		"template.yaml",
		false,
		nil,
		fixedEnvResolver{inferred: runtimeinfra.EnvInference{Env: "dev", Source: "staging"}},
		false,
		nil,
	)
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
	if !errors.Is(err, errEnvMismatch) {
		t.Fatalf("expected errEnvMismatch, got %v", err)
	}
	if !strings.Contains(err.Error(), "running env uses") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestReconcileEnvWithRuntimeUsesPromptSelection(t *testing.T) {
	got, err := reconcileEnvWithRuntime(
		envChoice{Value: "prod", Source: "flag", Explicit: true},
		"esb-dev",
		"template.yaml",
		true,
		selectOnlyPrompter{selected: "dev"},
		fixedEnvResolver{inferred: runtimeinfra.EnvInference{Env: "dev", Source: "container label"}},
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("reconcile env: %v", err)
	}
	if got.Value != "dev" {
		t.Fatalf("expected selected inferred env, got %q", got.Value)
	}
	if got.Source != "container label" {
		t.Fatalf("expected inferred source, got %q", got.Source)
	}
	if !got.Explicit {
		t.Fatalf("expected explicit=true after prompt selection")
	}
}

func TestApplyEnvSelection(t *testing.T) {
	inferred := runtimeinfra.EnvInference{Env: "dev", Source: "container label"}
	tests := []struct {
		name     string
		current  envChoice
		selected string
		want     envChoice
	}{
		{
			name:     "keeps current and marks prompt source",
			current:  envChoice{Value: "prod", Source: "default", Explicit: false},
			selected: "prod",
			want:     envChoice{Value: "prod", Source: "prompt", Explicit: true},
		},
		{
			name:     "uses inferred when selected",
			current:  envChoice{Value: "prod", Source: "flag", Explicit: true},
			selected: "dev",
			want:     envChoice{Value: "dev", Source: "container label", Explicit: true},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := applyEnvSelection(tc.current, inferred, tc.selected)
			if got != tc.want {
				t.Fatalf("applyEnvSelection() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestReconcileEnvWithRuntimeSkipsWhenProjectEmpty(t *testing.T) {
	choice := envChoice{Value: "dev", Source: "flag", Explicit: true}
	assertReconcileKeepsChoice(
		t,
		choice,
		"",
		runtimeinfra.EnvInference{Env: "prod", Source: "runtime"},
		false,
	)
}

func TestReconcileEnvWithRuntimeResolverErrorWritesWarning(t *testing.T) {
	choice := envChoice{Value: "dev", Source: "flag", Explicit: true}
	var errOut bytes.Buffer
	got, err := reconcileEnvWithRuntime(
		choice,
		"esb-dev",
		"template.yaml",
		false,
		nil,
		fixedEnvResolver{err: errors.New("boom")},
		false,
		&errOut,
	)
	if err != nil {
		t.Fatalf("reconcile env: %v", err)
	}
	if got != choice {
		t.Fatalf("expected choice unchanged, got %#v", got)
	}
	if !strings.Contains(errOut.String(), "failed to infer running env") {
		t.Fatalf("expected warning output, got %q", errOut.String())
	}
}

func assertReconcileKeepsChoice(
	t *testing.T,
	choice envChoice,
	project string,
	inferred runtimeinfra.EnvInference,
	force bool,
) {
	t.Helper()
	got, err := reconcileEnvWithRuntime(
		choice,
		project,
		"template.yaml",
		false,
		nil,
		fixedEnvResolver{inferred: inferred},
		force,
		nil,
	)
	if err != nil {
		t.Fatalf("reconcile env: %v", err)
	}
	if got != choice {
		t.Fatalf("expected choice unchanged, got %#v", got)
	}
}

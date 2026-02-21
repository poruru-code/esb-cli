// Where: cli/internal/command/deploy_inputs_mode_test.go
// What: Tests for runtime mode prompt behavior.
// Why: Ensure validation warnings are routed to the injected error writer.
package command

import (
	"bytes"
	"errors"
	"testing"

	"github.com/poruru-code/esb-cli/internal/infra/interaction"
)

type modeSelectPrompter struct {
	selectValues         []string
	selectIndex          int
	selectValueValues    []string
	selectValueIndex     int
	selectValueCalls     int
	lastSelectValueTitle string
	lastSelectValueOpts  []interaction.SelectOption
	selectValueErr       error
}

func (p *modeSelectPrompter) Input(_ string, _ []string) (string, error) {
	return "", nil
}

func (p *modeSelectPrompter) Select(_ string, _ []string) (string, error) {
	if p.selectIndex >= len(p.selectValues) {
		return "", nil
	}
	value := p.selectValues[p.selectIndex]
	p.selectIndex++
	return value, nil
}

func (p *modeSelectPrompter) SelectValue(
	title string,
	options []interaction.SelectOption,
) (string, error) {
	p.selectValueCalls++
	p.lastSelectValueTitle = title
	p.lastSelectValueOpts = append([]interaction.SelectOption{}, options...)
	if p.selectValueErr != nil {
		return "", p.selectValueErr
	}
	if p.selectValueIndex >= len(p.selectValueValues) {
		return "", nil
	}
	value := p.selectValueValues[p.selectValueIndex]
	p.selectValueIndex++
	return value, nil
}

func TestResolveDeployModeWritesWarningToConfiguredErrWriter(t *testing.T) {
	prompter := &modeSelectPrompter{selectValues: []string{"", "containerd"}}
	var errOut bytes.Buffer

	got, err := resolveDeployMode(true, prompter, "docker", &errOut)
	if err != nil {
		t.Fatalf("resolve deploy mode: %v", err)
	}
	if got != "containerd" {
		t.Fatalf("expected containerd, got %q", got)
	}
	if errOut.String() == "" {
		t.Fatalf("expected warning output when first selection is empty")
	}
}

func TestResolveDeployModeConflictInteractiveUsesSelectedMode(t *testing.T) {
	prompter := &modeSelectPrompter{selectValueValues: []string{"containerd"}}

	got, err := resolveDeployModeConflict(
		"docker",
		"compose labels",
		"containerd",
		true,
		prompter,
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("resolve deploy mode conflict: %v", err)
	}
	if got != "containerd" {
		t.Fatalf("expected containerd, got %q", got)
	}
	if prompter.selectValueCalls != 1 {
		t.Fatalf("expected one select-value call, got %d", prompter.selectValueCalls)
	}
	if prompter.lastSelectValueOpts[0].Value != "docker" {
		t.Fatalf("expected inferred mode to be default option, got %q", prompter.lastSelectValueOpts[0].Value)
	}
}

func TestResolveDeployModeConflictInteractiveUsesPreviousSelectionAsDefault(t *testing.T) {
	prompter := &modeSelectPrompter{selectValueValues: []string{"containerd"}}

	got, err := resolveDeployModeConflict(
		"docker",
		"compose labels",
		"containerd",
		true,
		prompter,
		"containerd",
		nil,
	)
	if err != nil {
		t.Fatalf("resolve deploy mode conflict: %v", err)
	}
	if got != "containerd" {
		t.Fatalf("expected containerd, got %q", got)
	}
	if len(prompter.lastSelectValueOpts) == 0 || prompter.lastSelectValueOpts[0].Value != "containerd" {
		t.Fatalf("expected previous selection to be first option, got %#v", prompter.lastSelectValueOpts)
	}
}

func TestResolveDeployModeConflictNonTTYUsesInferredAndWarns(t *testing.T) {
	var errOut bytes.Buffer

	got, err := resolveDeployModeConflict(
		"docker",
		"compose labels",
		"containerd",
		false,
		nil,
		"",
		&errOut,
	)
	if err != nil {
		t.Fatalf("resolve deploy mode conflict: %v", err)
	}
	if got != "docker" {
		t.Fatalf("expected docker, got %q", got)
	}
	if !bytes.Contains(errOut.Bytes(), []byte("ignoring --mode containerd")) {
		t.Fatalf("expected non-tty warning output, got %q", errOut.String())
	}
}

func TestDeployInputsResolverResolveModePromptsOnInferenceConflict(t *testing.T) {
	prompter := &modeSelectPrompter{selectValueValues: []string{"docker"}}
	var errOut bytes.Buffer
	resolver := deployInputsResolver{
		isTTY:    true,
		prompter: prompter,
		errOut:   &errOut,
	}
	cli := CLI{Deploy: DeployCmd{Mode: "docker"}}
	ctx := deployRuntimeContext{
		inferredMode:    "containerd",
		inferredModeSrc: "compose labels",
	}
	last := deployInputs{Mode: "docker"}

	got, err := resolver.resolveMode(cli, ctx, storedDeployDefaults{}, last)
	if err != nil {
		t.Fatalf("resolve mode: %v", err)
	}
	if got != "docker" {
		t.Fatalf("expected docker, got %q", got)
	}
	if prompter.selectValueCalls != 1 {
		t.Fatalf("expected one select-value call, got %d", prompter.selectValueCalls)
	}
	if len(prompter.lastSelectValueOpts) == 0 || prompter.lastSelectValueOpts[0].Value != "docker" {
		t.Fatalf("expected previous selected mode to be first option, got %#v", prompter.lastSelectValueOpts)
	}
}

func TestResolveDeployModeConflictInteractiveRetriesOnEmptySelection(t *testing.T) {
	prompter := &modeSelectPrompter{selectValueValues: []string{"", "docker"}}
	var errOut bytes.Buffer

	got, err := resolveDeployModeConflict(
		"docker",
		"compose labels",
		"containerd",
		true,
		prompter,
		"",
		&errOut,
	)
	if err != nil {
		t.Fatalf("resolve deploy mode conflict: %v", err)
	}
	if got != "docker" {
		t.Fatalf("expected docker, got %q", got)
	}
	if prompter.selectValueCalls != 2 {
		t.Fatalf("expected retry on empty selection, got %d calls", prompter.selectValueCalls)
	}
	if !bytes.Contains(errOut.Bytes(), []byte("Runtime mode selection is required.")) {
		t.Fatalf("expected retry warning, got %q", errOut.String())
	}
}

func TestResolveDeployModeConflictInteractiveReturnsPromptError(t *testing.T) {
	prompter := &modeSelectPrompter{selectValueErr: errors.New("boom")}
	_, err := resolveDeployModeConflict(
		"docker",
		"compose labels",
		"containerd",
		true,
		prompter,
		"",
		nil,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "prompt runtime mode conflict: boom" {
		t.Fatalf("unexpected error: %v", err)
	}
}

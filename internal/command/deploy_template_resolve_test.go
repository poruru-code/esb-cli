// Where: cli/internal/command/deploy_template_resolve_test.go
// What: Tests for template resolution and output derivation helpers.
// Why: Increase coverage for prompt/error branches and deterministic output naming.
package command

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poruru-code/esb-cli/internal/infra/interaction"
)

type templateResolvePrompter struct {
	inputs  []string
	selects []string
}

func (p *templateResolvePrompter) Input(_ string, _ []string) (string, error) {
	if len(p.inputs) == 0 {
		return "", nil
	}
	value := p.inputs[0]
	p.inputs = p.inputs[1:]
	return value, nil
}

func (p *templateResolvePrompter) Select(_ string, _ []string) (string, error) {
	if len(p.selects) == 0 {
		return "", nil
	}
	value := p.selects[0]
	p.selects = p.selects[1:]
	return value, nil
}

func (p *templateResolvePrompter) SelectValue(_ string, _ []interaction.SelectOption) (string, error) {
	return "", nil
}

func TestResolveDeployTemplateRequiresValueWithoutTTY(t *testing.T) {
	_, err := resolveDeployTemplate("", false, nil, "", &bytes.Buffer{})
	if !errors.Is(err, errTemplatePathRequired) {
		t.Fatalf("expected errTemplatePathRequired, got %v", err)
	}
}

func TestResolveDeployTemplateRetriesOnInvalidPromptInput(t *testing.T) {
	tmp := t.TempDir()
	valid := filepath.Join(tmp, "template.yaml")
	if err := os.WriteFile(valid, []byte("Resources: {}"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	prompter := &templateResolvePrompter{
		inputs: []string{"missing.yaml", "template.yaml"},
	}
	var errOut bytes.Buffer

	got, err := resolveDeployTemplate("", true, prompter, "", &errOut)
	if err != nil {
		t.Fatalf("resolve template: %v", err)
	}
	if got != valid {
		t.Fatalf("expected %q, got %q", valid, got)
	}
	if !bytes.Contains(errOut.Bytes(), []byte("Invalid template path")) {
		t.Fatalf("expected invalid-path warning, got %q", errOut.String())
	}
}

func TestResolveDeployTemplateManualSelectionFallback(t *testing.T) {
	tmp := t.TempDir()
	valid := filepath.Join(tmp, "template.yaml")
	if err := os.WriteFile(valid, []byte("Resources: {}"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	prompter := &templateResolvePrompter{
		selects: []string{templateManualOption},
		inputs:  []string{""},
	}
	var errOut bytes.Buffer

	got, err := resolveDeployTemplate("", true, prompter, "", &errOut)
	if err != nil {
		t.Fatalf("resolve template: %v", err)
	}
	if got != valid {
		t.Fatalf("expected %q, got %q", valid, got)
	}
}

func TestResolveDeployTemplateSelectSuggestion(t *testing.T) {
	tmp := t.TempDir()
	serviceDir := filepath.Join(tmp, "svc")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("mkdir svc: %v", err)
	}
	expected := filepath.Join(serviceDir, "template.yaml")
	if err := os.WriteFile(expected, []byte("Resources: {}"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	prompter := &templateResolvePrompter{
		selects: []string{"svc"},
	}
	got, err := resolveDeployTemplate("", true, prompter, "", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("resolve template: %v", err)
	}
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestResolveDeployTemplateManualSelectionWithSuggestions(t *testing.T) {
	tmp := t.TempDir()
	serviceDir := filepath.Join(tmp, "svc")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("mkdir svc: %v", err)
	}
	expected := filepath.Join(serviceDir, "template.yaml")
	if err := os.WriteFile(expected, []byte("Resources: {}"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	prompter := &templateResolvePrompter{
		selects: []string{templateManualOption},
		inputs:  []string{"svc"},
	}
	got, err := resolveDeployTemplate("", true, prompter, "", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("resolve template: %v", err)
	}
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestResolveDeployTemplateRetriesOnInvalidSelection(t *testing.T) {
	tmp := t.TempDir()
	serviceDir := filepath.Join(tmp, "svc")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("mkdir svc: %v", err)
	}
	expected := filepath.Join(serviceDir, "template.yaml")
	if err := os.WriteFile(expected, []byte("Resources: {}"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	prompter := &templateResolvePrompter{
		selects: []string{"missing", "svc"},
	}
	var errOut bytes.Buffer

	got, err := resolveDeployTemplate("", true, prompter, "", &errOut)
	if err != nil {
		t.Fatalf("resolve template: %v", err)
	}
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
	if !strings.Contains(errOut.String(), "Invalid template path") {
		t.Fatalf("expected invalid path warning, got %q", errOut.String())
	}
}

func TestResolveTemplatePromptInputReturnsRequiredWhenFallbackFails(t *testing.T) {
	tmp := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	_, err = resolveTemplatePromptInput("", "", "", nil)
	if !errors.Is(err, errTemplatePathRequired) {
		t.Fatalf("expected errTemplatePathRequired, got %v", err)
	}
}

func TestResolveTemplatePromptInputInvalidValue(t *testing.T) {
	_, err := resolveTemplatePromptInput("missing.yaml", "", "", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "stat template path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveDeployTemplatesNormalizesMultipleValues(t *testing.T) {
	tmp := t.TempDir()
	a := filepath.Join(tmp, "a.yaml")
	b := filepath.Join(tmp, "b.yaml")
	if err := os.WriteFile(a, []byte("Resources: {}"), 0o600); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.WriteFile(b, []byte("Resources: {}"), 0o600); err != nil {
		t.Fatalf("write b: %v", err)
	}

	got, err := resolveDeployTemplates([]string{a, "  " + b + "  "}, false, nil, "", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("resolve templates: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(got))
	}
	if got[0] != a || got[1] != b {
		t.Fatalf("unexpected templates: %#v", got)
	}
}

func TestDeriveTemplateArtifactOutputDirUsesID(t *testing.T) {
	root := t.TempDir()
	got := deriveTemplateArtifactOutputDir(root, "template-e2e-1234abcd")
	want := filepath.Join(root, "entries", "template-e2e-1234abcd")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestDeriveTemplateArtifactIDStable(t *testing.T) {
	repoRoot := t.TempDir()
	templatePath := filepath.Join(repoRoot, "template.yaml")
	if err := os.WriteFile(templatePath, []byte("Resources: {}\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	params := map[string]string{"B": "2", "A": "1"}

	first, err := deriveTemplateArtifactID(repoRoot, templatePath, params)
	if err != nil {
		t.Fatalf("deriveTemplateArtifactID() first error = %v", err)
	}
	second, err := deriveTemplateArtifactID(repoRoot, templatePath, map[string]string{"A": "1", "B": "2"})
	if err != nil {
		t.Fatalf("deriveTemplateArtifactID() second error = %v", err)
	}
	if first != second {
		t.Fatalf("expected stable artifact id, got %q != %q", first, second)
	}
}

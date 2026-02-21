// Where: cli/internal/command/deploy_inputs_output.go
// What: Artifact root and per-template output resolution helpers for deploy inputs.
// Why: Keep output path behavior isolated and deterministic in single-root mode.
package command

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/poruru-code/esb-cli/internal/infra/interaction"
	"github.com/poruru-code/esb/pkg/artifactcore"
)

func resolveDeployArtifactRoot(
	value string,
	isTTY bool,
	prompter interaction.Prompter,
	previous string,
	projectDir string,
	project string,
	env string,
) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return normalizeArtifactRootPath(trimmed, projectDir), nil
	}
	defaultPath := strings.TrimSpace(previous)
	if defaultPath == "" {
		defaultPath = defaultDeployArtifactRoot(projectDir, project, env)
	}
	if !isTTY || prompter == nil {
		return normalizeArtifactRootPath(defaultPath, projectDir), nil
	}
	input, err := prompter.Input(
		fmt.Sprintf("Artifact root directory (default: %s)", defaultPath),
		[]string{defaultPath},
	)
	if err != nil {
		return "", fmt.Errorf("prompt artifact root directory: %w", err)
	}
	selected := strings.TrimSpace(input)
	if selected == "" {
		return normalizeArtifactRootPath(defaultPath, projectDir), nil
	}
	return normalizeArtifactRootPath(selected, projectDir), nil
}

func defaultDeployArtifactRoot(projectDir, project, env string) string {
	root := strings.TrimSpace(projectDir)
	if root == "" {
		root = "."
	}
	projectSegment := sanitizePathSegment(project)
	envSegment := sanitizePathSegment(env)
	scope := projectSegment
	if !strings.HasSuffix(scope, "-"+envSegment) {
		scope = scope + "-" + envSegment
	}
	return filepath.Join(
		root,
		"artifacts",
		scope,
	)
}

func normalizeArtifactRootPath(pathValue, projectDir string) string {
	candidate := filepath.Clean(strings.TrimSpace(pathValue))
	if !filepath.IsAbs(candidate) {
		base := strings.TrimSpace(projectDir)
		if base == "" {
			base = "."
		}
		candidate = filepath.Join(base, candidate)
	}
	return filepath.Clean(candidate)
}

func deriveTemplateArtifactID(
	projectDir string,
	templatePath string,
	parameters map[string]string,
) (string, error) {
	templateSHA, err := fileSHA256(templatePath)
	if err != nil {
		return "", fmt.Errorf("hash template %s: %w", templatePath, err)
	}
	sourcePath := normalizeSourceTemplatePath(projectDir, templatePath)
	return artifactcore.ComputeArtifactID(sourcePath, parameters, templateSHA), nil
}

func deriveTemplateArtifactOutputDir(artifactRoot, artifactID string) string {
	trimmedRoot := strings.TrimSpace(artifactRoot)
	if trimmedRoot == "" {
		return ""
	}
	trimmedID := strings.TrimSpace(artifactID)
	if trimmedID == "" {
		trimmedID = "artifact"
	}
	return filepath.Join(trimmedRoot, "entries", trimmedID)
}

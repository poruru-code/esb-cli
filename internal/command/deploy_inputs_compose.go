// Where: cli/internal/command/deploy_inputs_compose.go
// What: Compose-file normalization helpers for deploy inputs.
// Why: Keep path normalization concerns separate from prompt flow logic.
package command

import (
	"github.com/poruru-code/esb-cli/internal/infra/compose"
	"github.com/poruru-code/esb-cli/internal/infra/interaction"
)

func normalizeComposeFiles(files []string, baseDir string) []string {
	if len(files) == 0 {
		return nil
	}
	return compose.NormalizeComposeFilePaths(files, baseDir)
}

func resolveDeployComposeFiles(
	values []string,
	isTTY bool,
	prompter interaction.Prompter,
	previous []string,
	baseDir string,
) ([]string, error) {
	_ = isTTY
	_ = prompter

	// Compose files are not prompted interactively.
	// Explicit --compose-file wins; otherwise we keep previous explicit value.
	if normalized := normalizeComposeFiles(values, baseDir); len(normalized) > 0 {
		return normalized, nil
	}
	if defaultFiles := normalizeComposeFiles(previous, baseDir); len(defaultFiles) > 0 {
		return append([]string{}, defaultFiles...), nil
	}
	return nil, nil
}

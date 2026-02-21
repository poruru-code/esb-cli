// Where: cli/internal/command/deploy_inputs_compose.go
// What: Compose-file normalization helpers for deploy inputs.
// Why: Keep path normalization concerns separate from prompt flow logic.
package command

import (
	"fmt"
	"strings"

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
	if normalized := normalizeComposeFiles(values, baseDir); len(normalized) > 0 {
		return normalized, nil
	}
	if !isTTY || prompter == nil {
		return nil, nil
	}

	defaultFiles := normalizeComposeFiles(previous, baseDir)
	defaultValue := "auto"
	suggestions := []string{"auto"}
	if len(defaultFiles) > 0 {
		defaultValue = strings.Join(defaultFiles, ",")
		suggestions = []string{defaultValue, "auto"}
	}
	input, err := prompter.Input(
		fmt.Sprintf("Compose file(s) (comma-separated, default: %s)", defaultValue),
		suggestions,
	)
	if err != nil {
		return nil, fmt.Errorf("prompt compose files: %w", err)
	}
	selected := strings.TrimSpace(input)
	switch {
	case selected == "":
		if len(defaultFiles) == 0 {
			return nil, nil
		}
		return append([]string{}, defaultFiles...), nil
	case strings.EqualFold(selected, "auto"):
		return nil, nil
	default:
		return normalizeComposeFiles(strings.Split(selected, ","), baseDir), nil
	}
}

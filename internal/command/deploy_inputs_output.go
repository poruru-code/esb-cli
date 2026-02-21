// Where: cli/internal/command/deploy_inputs_output.go
// What: Output directory resolution helpers for deploy inputs.
// Why: Keep output path behavior isolated and easier to reason about.
package command

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/poruru-code/esb-cli/internal/infra/interaction"
	"github.com/poruru-code/esb-cli/internal/meta"
)

func resolveDeployOutput(
	value string,
	isTTY bool,
	prompter interaction.Prompter,
	previous string,
) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed, nil
	}
	if !isTTY || prompter == nil {
		return "", nil
	}
	if prev := strings.TrimSpace(previous); prev != "" {
		input, err := prompter.Input(
			fmt.Sprintf("Output directory (default: %s)", prev),
			[]string{prev},
		)
		if err != nil {
			return "", fmt.Errorf("prompt output directory: %w", err)
		}
		if selected := strings.TrimSpace(input); selected != "" {
			return selected, nil
		}
		return prev, nil
	}
	input, err := prompter.Input(
		"Output directory (default: auto)",
		[]string{"auto"},
	)
	if err != nil {
		return "", fmt.Errorf("prompt output directory: %w", err)
	}
	selected := strings.TrimSpace(input)
	if selected == "" || strings.EqualFold(selected, "auto") {
		return "", nil
	}
	return selected, nil
}

func deriveMultiTemplateOutputDir(templatePath string, counts map[string]int) string {
	base := strings.TrimSpace(filepath.Base(templatePath))
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	stem = strings.TrimSpace(stem)
	if stem == "" {
		stem = "template"
	}
	count := counts[stem]
	counts[stem] = count + 1
	if count > 0 {
		stem = fmt.Sprintf("%s-%d", stem, count+1)
	}
	return filepath.Join(meta.OutputDir, stem)
}

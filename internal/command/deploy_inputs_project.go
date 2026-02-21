// Where: cli/internal/command/deploy_inputs_project.go
// What: Compose project resolution helpers for deploy inputs.
// Why: Keep interactive project prompt behavior isolated from main resolve flow.
package command

import (
	"fmt"
	"io"
	"strings"

	"github.com/poruru-code/esb-cli/internal/infra/interaction"
)

func resolveDeployProject(
	defaultProject string,
	isTTY bool,
	prompter interaction.Prompter,
	previous string,
	errOut io.Writer,
) (string, string, error) {
	trimmedDefault := strings.TrimSpace(defaultProject)
	if trimmedDefault == "" {
		return "", "", errComposeProjectRequired
	}
	if !isTTY || prompter == nil {
		return trimmedDefault, "default", nil
	}

	previousProject := strings.TrimSpace(previous)
	title := fmt.Sprintf("Compose project (default: %s)", trimmedDefault)
	promptDefault := trimmedDefault
	promptSource := "default"
	suggestions := []string{trimmedDefault}
	if previousProject != "" && previousProject != trimmedDefault {
		title = fmt.Sprintf(
			"Compose project (default: %s, inferred: %s)",
			previousProject,
			trimmedDefault,
		)
		promptDefault = previousProject
		promptSource = "previous"
		suggestions = []string{previousProject, trimmedDefault}
	}

	for {
		input, err := prompter.Input(title, suggestions)
		if err != nil {
			return "", "", fmt.Errorf("prompt compose project: %w", err)
		}
		selected := strings.TrimSpace(input)
		if selected == "" {
			if promptDefault == "" {
				writeWarningf(errOut, "Compose project is required.\n")
				continue
			}
			return promptDefault, promptSource, nil
		}
		if selected == previousProject {
			return previousProject, "previous", nil
		}
		if selected == trimmedDefault {
			return trimmedDefault, "default", nil
		}
		return selected, "prompt", nil
	}
}

func reconcileDeployProjectWithEnv(
	currentProject string,
	projectSource string,
	selectedEnv string,
) (string, string) {
	if strings.TrimSpace(projectSource) != "default" {
		return currentProject, projectSource
	}
	env := strings.TrimSpace(selectedEnv)
	if env == "" {
		return currentProject, projectSource
	}
	return defaultDeployProject(env), "default"
}

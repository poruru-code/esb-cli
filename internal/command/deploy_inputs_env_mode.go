// Where: cli/internal/command/deploy_inputs_env_mode.go
// What: Environment and runtime mode resolution helpers for deploy inputs.
// Why: Keep prompt-driven env/mode logic separate from the main resolve flow.
package command

import (
	"fmt"
	"io"
	"strings"

	runtimecfg "github.com/poruru-code/esb-cli/internal/domain/runtime"
	"github.com/poruru-code/esb-cli/internal/infra/interaction"
	runtimeinfra "github.com/poruru-code/esb-cli/internal/infra/runtime"
)

func resolveDeployEnv(
	value string,
	isTTY bool,
	prompter interaction.Prompter,
	previous string,
) (envChoice, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return envChoice{Value: trimmed, Source: "flag", Explicit: true}, nil
	}
	if !isTTY || prompter == nil {
		return envChoice{}, errEnvironmentRequired
	}
	defaultValue := strings.TrimSpace(previous)
	if defaultValue == "" {
		defaultValue = "default"
	}
	title := fmt.Sprintf("Environment name (default: %s)", defaultValue)
	input, err := prompter.Input(title, []string{defaultValue})
	if err != nil {
		return envChoice{}, fmt.Errorf("prompt environment: %w", err)
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return envChoice{Value: defaultValue, Source: "default", Explicit: false}, nil
	}
	return envChoice{Value: input, Source: "prompt", Explicit: true}, nil
}

func resolveDeployEnvFromStack(
	envValue string,
	stack deployTargetStack,
	composeProject string,
	isTTY bool,
	prompter interaction.Prompter,
	resolver runtimeinfra.EnvResolver,
	previous string,
) (envChoice, error) {
	if trimmed := strings.TrimSpace(envValue); trimmed != "" {
		return envChoice{Value: trimmed, Source: "flag", Explicit: true}, nil
	}
	if env := strings.TrimSpace(stack.Env); env != "" {
		return envChoice{Value: env, Source: "stack", Explicit: false}, nil
	}
	if project := strings.TrimSpace(composeProject); project != "" && resolver != nil {
		if inferred, err := resolver.InferEnvFromProject(project, ""); err == nil && strings.TrimSpace(inferred.Env) != "" {
			return envChoice{Value: inferred.Env, Source: inferred.Source, Explicit: false}, nil
		}
	}
	return resolveDeployEnv("", isTTY, prompter, previous)
}

func resolveDeployMode(
	isTTY bool,
	prompter interaction.Prompter,
	previous string,
	errOut io.Writer,
) (string, error) {
	if !isTTY || prompter == nil {
		return "", errModeRequired
	}
	defaultValue := strings.TrimSpace(strings.ToLower(previous))
	if defaultValue == "" {
		defaultValue = "docker"
	}
	for {
		options := []string{defaultValue}
		for _, opt := range []string{"docker", "containerd"} {
			if opt == defaultValue {
				continue
			}
			options = append(options, opt)
		}
		title := fmt.Sprintf("Runtime mode (default: %s)", defaultValue)
		selected, err := prompter.Select(title, options)
		if err != nil {
			return "", fmt.Errorf("prompt runtime mode: %w", err)
		}
		selected = strings.TrimSpace(strings.ToLower(selected))
		if selected == "" {
			writeWarningf(errOut, "Runtime mode is required.\n")
			continue
		}
		normalized, err := runtimecfg.NormalizeMode(selected)
		if err != nil {
			return "", fmt.Errorf("normalize mode: %w", err)
		}
		return normalized, nil
	}
}

func resolveDeployModeConflict(
	inferredMode string,
	inferredSource string,
	flagMode string,
	isTTY bool,
	prompter interaction.Prompter,
	previous string,
	errOut io.Writer,
) (string, error) {
	warningMessage := fmt.Sprintf(
		"Warning: running project uses %s mode; ignoring --mode %s (source: %s)\n",
		inferredMode,
		flagMode,
		renderPromptContextValue(inferredSource),
	)
	if !isTTY || prompter == nil {
		writeWarningf(errOut, "%s", warningMessage)
		return inferredMode, nil
	}

	options := []interaction.SelectOption{
		{
			Label: fmt.Sprintf(
				"Use running project mode: %s (source: %s)",
				inferredMode,
				renderPromptContextValue(inferredSource),
			),
			Value: inferredMode,
		},
		{
			Label: fmt.Sprintf("Use --mode flag: %s", flagMode),
			Value: flagMode,
		},
	}
	defaultMode := strings.TrimSpace(strings.ToLower(previous))
	if defaultMode != inferredMode && defaultMode != flagMode {
		defaultMode = inferredMode
	}
	ordered := orderModeConflictOptions(options, defaultMode)
	title := fmt.Sprintf(
		"Runtime mode conflict (running: %s [%s], --mode: %s)",
		inferredMode,
		renderPromptContextValue(inferredSource),
		flagMode,
	)
	for {
		selected, err := prompter.SelectValue(title, ordered)
		if err != nil {
			return "", fmt.Errorf("prompt runtime mode conflict: %w", err)
		}
		normalized, err := runtimecfg.NormalizeMode(selected)
		if err != nil {
			writeWarningf(errOut, "Runtime mode selection is required.\n")
			continue
		}
		return normalized, nil
	}
}

func orderModeConflictOptions(
	options []interaction.SelectOption,
	defaultMode string,
) []interaction.SelectOption {
	normalizedDefault := strings.TrimSpace(strings.ToLower(defaultMode))
	if normalizedDefault == "" {
		return append([]interaction.SelectOption{}, options...)
	}
	ordered := make([]interaction.SelectOption, 0, len(options))
	for _, option := range options {
		if strings.TrimSpace(strings.ToLower(option.Value)) == normalizedDefault {
			ordered = append(ordered, option)
		}
	}
	for _, option := range options {
		if strings.TrimSpace(strings.ToLower(option.Value)) != normalizedDefault {
			ordered = append(ordered, option)
		}
	}
	if len(ordered) == 0 {
		return append([]interaction.SelectOption{}, options...)
	}
	return ordered
}

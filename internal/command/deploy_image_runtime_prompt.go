// Where: cli/internal/command/deploy_image_runtime_prompt.go
// What: Image-function runtime prompt and normalization helpers.
// Why: Let users choose per-image runtime at deploy time without changing SAM templates.
package command

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	runtimecfg "github.com/poruru-code/esb-cli/internal/domain/runtime"
	"github.com/poruru-code/esb-cli/internal/infra/interaction"
	"github.com/poruru-code/esb-cli/internal/infra/sam"
)

const (
	defaultImageRuntimeChoice = "python"
	defaultImageRuntimeValue  = "python3.12"
	imageRuntimeJava21        = "java21"
)

type imageRuntimePromptTarget struct {
	Name        string
	ImageSource string
}

func promptTemplateImageRuntimes(
	templatePath string,
	parameters map[string]string,
	imageSources map[string]string,
	isTTY bool,
	prompter interaction.Prompter,
	previous map[string]string,
	explicit map[string]string,
	errOut io.Writer,
) (map[string]string, error) {
	imageFunctions, err := discoverImageRuntimePromptTargets(templatePath, parameters)
	if err != nil {
		return nil, err
	}
	if len(imageFunctions) == 0 {
		return nil, nil
	}

	values := make(map[string]string, len(imageFunctions))
	for _, target := range imageFunctions {
		functionName := target.Name
		if explicit != nil {
			if override := strings.TrimSpace(explicit[functionName]); override != "" {
				normalized, err := normalizeImageRuntimeSelection(override)
				if err != nil {
					return nil, fmt.Errorf("invalid image runtime for %s: %w", functionName, err)
				}
				values[functionName] = normalized
				continue
			}
		}
		prev := ""
		if previous != nil {
			prev = strings.TrimSpace(previous[functionName])
		}
		defaultRuntime, err := resolveImageRuntimeOrDefault(prev)
		if err != nil {
			writeWarningf(
				errOut,
				"Ignoring previous runtime %q for image function %s: %v\n",
				prev,
				functionName,
				err,
			)
			defaultRuntime = defaultImageRuntimeValue
		}

		if !isTTY || prompter == nil {
			values[functionName] = defaultRuntime
			continue
		}

		defaultChoice := imageRuntimeChoice(defaultRuntime)
		title := fmt.Sprintf(
			"Runtime for image function %s (image: %s, default: %s)",
			functionName,
			resolvePromptImageSource(target, imageSources),
			defaultChoice,
		)
		selected, err := prompter.Select(title, orderedImageRuntimeChoices(defaultChoice))
		if err != nil {
			return nil, fmt.Errorf("prompt image runtime for %s: %w", functionName, err)
		}
		if strings.TrimSpace(selected) == "" {
			selected = defaultChoice
		}

		runtimeValue, err := normalizeImageRuntimeSelection(selected)
		if err != nil {
			return nil, fmt.Errorf("invalid image runtime for %s: %w", functionName, err)
		}
		values[functionName] = runtimeValue
	}

	return values, nil
}

func discoverImageRuntimePromptTargets(
	templatePath string,
	parameters map[string]string,
) ([]imageRuntimePromptTarget, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("read template for image runtime: %w", err)
	}
	parsed, err := sam.ParseSAMTemplate(string(content), cloneStringMap(parameters))
	if err != nil {
		return nil, fmt.Errorf("parse template for image runtime: %w", err)
	}

	targets := make([]imageRuntimePromptTarget, 0, len(parsed.Functions))
	for _, fn := range parsed.Functions {
		source := strings.TrimSpace(fn.ImageSource)
		if source == "" {
			continue
		}
		targets = append(targets, imageRuntimePromptTarget{
			Name:        fn.Name,
			ImageSource: source,
		})
	}
	sort.SliceStable(targets, func(i, j int) bool {
		return targets[i].Name < targets[j].Name
	})
	return targets, nil
}

func resolvePromptImageSource(target imageRuntimePromptTarget, imageSources map[string]string) string {
	if imageSources != nil {
		if override := strings.TrimSpace(imageSources[target.Name]); override != "" {
			return override
		}
	}
	if source := strings.TrimSpace(target.ImageSource); source != "" {
		return source
	}
	return "<unknown>"
}

func resolveImageRuntimeOrDefault(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultImageRuntimeValue, nil
	}
	return normalizeImageRuntimeSelection(trimmed)
}

func normalizeImageRuntimeSelection(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", defaultImageRuntimeChoice:
		normalized = defaultImageRuntimeValue
	case defaultImageRuntimeValue, imageRuntimeJava21:
		// keep as is
	default:
		return "", fmt.Errorf("unsupported runtime %q (use python or java21)", value)
	}

	profile, err := runtimecfg.Resolve(normalized)
	if err != nil {
		return "", err
	}
	switch profile.Kind {
	case runtimecfg.KindPython, runtimecfg.KindJava:
		return profile.Name, nil
	default:
		return "", fmt.Errorf("unsupported runtime kind %q", profile.Kind)
	}
}

func imageRuntimeChoice(runtimeValue string) string {
	normalized, err := normalizeImageRuntimeSelection(runtimeValue)
	if err != nil {
		return defaultImageRuntimeChoice
	}
	if strings.HasPrefix(normalized, "python") {
		return defaultImageRuntimeChoice
	}
	return normalized
}

func orderedImageRuntimeChoices(defaultChoice string) []string {
	out := []string{defaultImageRuntimeChoice}
	if strings.TrimSpace(defaultChoice) != "" {
		out[0] = defaultChoice
	}
	for _, choice := range []string{defaultImageRuntimeChoice, imageRuntimeJava21} {
		if choice == out[0] {
			continue
		}
		out = append(out, choice)
	}
	return out
}

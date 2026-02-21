// Where: cli/internal/command/deploy_template_prompt.go
// What: SAM template parameter extraction and prompt handling.
// Why: Keep parameter prompt behavior isolated from template path resolution.
package command

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/poruru-code/esb-cli/internal/infra/interaction"
	"github.com/poruru-code/esb-cli/internal/infra/sam"
	"github.com/poruru-code/esb/pkg/yamlshape"
)

func promptTemplateParameters(
	templatePath string,
	isTTY bool,
	prompter interaction.Prompter,
	previous map[string]string,
	errOut io.Writer,
) (map[string]string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return map[string]string{}, fmt.Errorf("read template: %w", err)
	}

	data, err := sam.DecodeYAML(string(content))
	if err != nil {
		return map[string]string{}, fmt.Errorf("decode template: %w", err)
	}

	params := extractSAMParameters(data)
	if len(params) == 0 {
		return map[string]string{}, nil
	}

	names := make([]string, 0, len(params))
	for name := range params {
		names = append(names, name)
	}
	sort.Strings(names)

	values := make(map[string]string, len(params))
	for _, name := range names {
		param := params[name]
		hasDefault := param.Default != nil
		defaultStr := ""
		if hasDefault {
			defaultStr = fmt.Sprint(param.Default)
		}
		allowsEmpty := parameterAllowsEmpty(param)
		prevValue := ""
		if previous != nil {
			prevValue = strings.TrimSpace(previous[name])
		}

		if !isTTY || prompter == nil {
			value := ""
			if hasDefault {
				value = defaultStr
			} else if prevValue != "" {
				value = prevValue
			} else if !allowsEmpty {
				return nil, fmt.Errorf("%w: %s", errParameterRequiresValue, name)
			}
			if err := validateAllowedParameterValue(name, value, param.Allowed); err != nil {
				return nil, err
			}
			values[name] = value
			continue
		}

		label := name
		if param.Description != "" {
			label = fmt.Sprintf("%s (%s)", name, param.Description)
		}

		var title string
		switch {
		case hasDefault:
			displayDefault := defaultStr
			if displayDefault == "" {
				displayDefault = "''"
			}
			title = fmt.Sprintf("%s [Default: %s]", label, displayDefault)
		case prevValue != "":
			title = fmt.Sprintf("%s [Previous: %s]", label, prevValue)
		case allowsEmpty:
			title = fmt.Sprintf("%s [Optional: empty allowed]", label)
		default:
			title = fmt.Sprintf("%s [Required]", label)
		}

		suggestions := []string{}
		if prevValue != "" {
			suggestions = append(suggestions, prevValue)
		}
		if hasDefault && defaultStr != "" {
			suggestions = appendSuggestion(suggestions, defaultStr)
		}
		for _, option := range param.Allowed {
			suggestions = appendSuggestion(suggestions, option)
		}
		for {
			input, err := prompter.Input(title, suggestions)
			if err != nil {
				return nil, fmt.Errorf("prompt parameter %s: %w", name, err)
			}
			input = strings.TrimSpace(input)
			if input == "" && hasDefault {
				input = defaultStr
			}
			if input == "" && prevValue != "" {
				input = prevValue
			}
			if input == "" && allowsEmpty {
				values[name] = ""
				break
			}
			if input == "" && !hasDefault {
				writeWarningf(errOut, "Parameter %q is required.\n", name)
				continue
			}
			if err := validateAllowedParameterValue(name, input, param.Allowed); err != nil {
				writeWarningf(errOut, "%v\n", err)
				continue
			}
			values[name] = input
			break
		}
	}

	return values, nil
}

// extractSAMParameters extracts parameter definitions from SAM template data.
func extractSAMParameters(data map[string]any) map[string]samParameter {
	result := make(map[string]samParameter)
	params := yamlshape.AsMap(data["Parameters"])
	if len(params) == 0 {
		return result
	}

	for name, val := range params {
		m := yamlshape.AsMap(val)
		if len(m) == 0 {
			continue
		}

		param := samParameter{}
		if t, ok := m["Type"].(string); ok {
			param.Type = t
		}
		if d, ok := m["Description"].(string); ok {
			param.Description = d
		}
		param.Default = m["Default"]
		param.Allowed = extractSAMAllowedValues(m["AllowedValues"])
		result[name] = param
	}

	return result
}

func extractSAMAllowedValues(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		value := strings.TrimSpace(fmt.Sprint(item))
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func appendSuggestion(suggestions []string, value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return suggestions
	}
	for _, current := range suggestions {
		if current == trimmed {
			return suggestions
		}
	}
	return append(suggestions, trimmed)
}

func parameterAllowsEmpty(param samParameter) bool {
	if !strings.EqualFold(strings.TrimSpace(param.Type), "String") {
		return false
	}
	if len(param.Allowed) == 0 {
		return true
	}
	for _, option := range param.Allowed {
		if strings.TrimSpace(option) == "" {
			return true
		}
	}
	return false
}

func validateAllowedParameterValue(name, value string, allowed []string) error {
	if len(allowed) == 0 {
		return nil
	}
	for _, option := range allowed {
		if value == option {
			return nil
		}
	}
	return fmt.Errorf(
		"parameter %q must be one of [%s]",
		name,
		strings.Join(formatAllowedParameterValues(allowed), ", "),
	)
}

func formatAllowedParameterValues(allowed []string) []string {
	formatted := make([]string, 0, len(allowed))
	for _, option := range allowed {
		display := option
		if strings.TrimSpace(display) == "" {
			display = "''"
		}
		formatted = append(formatted, display)
	}
	return formatted
}

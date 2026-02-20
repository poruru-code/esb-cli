// Where: cli/internal/infra/build/go_builder_paths.go
// What: Lightweight naming/path helpers used by GoBuilder.
// Why: Keep Build() focused on orchestration sequence rather than inline derivations.
package build

import (
	"os"
	"strings"

	"github.com/poruru-code/esb-cli/internal/constants"
	"github.com/poruru-code/esb-cli/internal/infra/staging"
)

func resolveComposeProjectName(projectName, env string) string {
	return staging.ComposeProjectKey(projectName, env)
}

func resolveRuntimeRegistry(defaultRegistry string) string {
	runtimeRegistry := strings.TrimSpace(defaultRegistry)
	if value := strings.TrimSpace(os.Getenv(constants.EnvContainerRegistry)); value != "" {
		if !strings.HasSuffix(value, "/") {
			value += "/"
		}
		runtimeRegistry = value
	}
	return runtimeRegistry
}

// Where: cli/internal/command/deploy_inputs_compose.go
// What: Compose-file normalization helpers for deploy inputs.
// Why: Keep path normalization concerns separate from prompt flow logic.
package command

import (
	"github.com/poruru-code/esb-cli/internal/infra/compose"
)

func normalizeComposeFiles(files []string, baseDir string) []string {
	if len(files) == 0 {
		return nil
	}
	return compose.NormalizeComposeFilePaths(files, baseDir)
}

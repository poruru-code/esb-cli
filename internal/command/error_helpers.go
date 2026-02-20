// Where: cli/internal/command/error_helpers.go
// What: Shared CLI error env.
// Why: Keep build-only CLI output consistent without project/env commands.
package command

import (
	"fmt"
	"io"
)

// exitWithError prints an error message to the output writer and returns
// exit code 1 for CLI error handling.
func exitWithError(out io.Writer, err error) int {
	legacyUI(out).Warn(fmt.Sprintf("✗ %v", err))
	return 1
}

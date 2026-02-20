// Where: cli/internal/infra/interaction/interaction.go
// What: Interactive primitives for CLI prompts and TTY detection.
// Why: Centralize user interaction to keep command handlers focused on orchestration.
package interaction

import (
	"os"

	"github.com/mattn/go-isatty"
)

// SelectOption represents a single option in a selection menu.
type SelectOption struct {
	Label string // Display text
	Value string // Return value
}

// Prompter defines the interface for interactive user input and selection.
type Prompter interface {
	Input(title string, suggestions []string) (string, error)
	Select(title string, options []string) (string, error)
	SelectValue(title string, options []SelectOption) (string, error)
}

// IsTerminal reports whether the file refers to a terminal device.
var IsTerminal = func(file *os.File) bool {
	if file == nil {
		return false
	}
	fd := file.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

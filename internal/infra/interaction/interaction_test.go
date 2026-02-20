// Where: cli/internal/infra/interaction/interaction_test.go
// What: Tests for terminal detection helpers.
// Why: Keep non-interactive detection deterministic in tests.
package interaction

import (
	"os"
	"testing"
)

func TestIsTerminalNilAndPipe(t *testing.T) {
	if IsTerminal(nil) {
		t.Fatal("IsTerminal(nil) must be false")
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	defer func() {
		_ = r.Close()
		_ = w.Close()
	}()
	if IsTerminal(r) {
		t.Fatal("IsTerminal(pipe) must be false")
	}
}

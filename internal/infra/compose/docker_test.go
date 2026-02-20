// Where: cli/internal/infra/compose/docker_test.go
// What: Tests for compose Docker helpers.
// Why: Keep container-name normalization deterministic.
package compose

import "testing"

func TestPrimaryContainerName(t *testing.T) {
	got := PrimaryContainerName([]string{"", "   ", "/gateway", "/other"})
	if got != "gateway" {
		t.Fatalf("expected gateway, got %q", got)
	}

	if empty := PrimaryContainerName([]string{"", " ", "/"}); empty != "" {
		t.Fatalf("expected empty name, got %q", empty)
	}
}

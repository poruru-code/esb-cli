package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/poruru-code/esb-cli/internal/constants"
	"github.com/poruru-code/esb-cli/internal/infra/envutil"
	"github.com/poruru-code/esb-cli/internal/infra/staging"
	"github.com/poruru-code/esb-cli/internal/meta"
)

func TestApplyBuildEnvSetsDefaults(t *testing.T) {
	t.Setenv(constants.EnvProjectName, "")
	t.Setenv("DOCKER_BUILDKIT", "")

	if err := applyBuildEnv("staging", "demo"); err != nil {
		t.Fatalf("apply build env: %v", err)
	}

	if got, want := getenvOrFail(t, constants.EnvProjectName), staging.ComposeProjectKey("demo", "staging"); got != want {
		t.Fatalf("unexpected project name: expected %q, got %q", want, got)
	}
	if got := getenvOrFail(t, "DOCKER_BUILDKIT"); got != "1" {
		t.Fatalf("unexpected DOCKER_BUILDKIT: %q", got)
	}
}

func TestApplyBuildEnvDoesNotOverwriteExisting(t *testing.T) {
	t.Setenv(constants.EnvProjectName, "custom-project")
	t.Setenv("DOCKER_BUILDKIT", "0")

	if err := applyBuildEnv("staging", "demo"); err != nil {
		t.Fatalf("apply build env: %v", err)
	}

	if got := getenvOrFail(t, constants.EnvProjectName); got != "custom-project" {
		t.Fatalf("project name was overwritten: %q", got)
	}
	if got := getenvOrFail(t, "DOCKER_BUILDKIT"); got != "0" {
		t.Fatalf("DOCKER_BUILDKIT was overwritten: %q", got)
	}
}

func TestApplyModeFromRequestSetsLowercaseWhenMissing(t *testing.T) {
	t.Setenv("ENV_PREFIX", meta.EnvPrefix)

	modeKey, err := envutil.HostEnvKey(constants.HostSuffixMode)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(modeKey, "")

	if err := applyModeFromRequest(" ConTainerD "); err != nil {
		t.Fatalf("apply mode from request: %v", err)
	}

	if got := getenvOrFail(t, modeKey); got != "containerd" {
		t.Fatalf("unexpected host mode: %q", got)
	}
}

func TestApplyModeFromRequestKeepsExistingHostMode(t *testing.T) {
	t.Setenv("ENV_PREFIX", meta.EnvPrefix)

	modeKey, err := envutil.HostEnvKey(constants.HostSuffixMode)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(modeKey, "docker")

	if err := applyModeFromRequest("containerd"); err != nil {
		t.Fatalf("apply mode from request: %v", err)
	}

	if got := getenvOrFail(t, modeKey); got != "docker" {
		t.Fatalf("host mode should remain unchanged, got %q", got)
	}
}

func TestApplyModeFromRequestNoopWhenModeEmpty(t *testing.T) {
	t.Setenv("ENV_PREFIX", meta.EnvPrefix)

	modeKey, err := envutil.HostEnvKey(constants.HostSuffixMode)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(modeKey, "")

	if err := applyModeFromRequest("   "); err != nil {
		t.Fatalf("apply mode from request: %v", err)
	}

	if got := getenvOrFail(t, modeKey); got != "" {
		t.Fatalf("expected empty host mode, got %q", got)
	}
}

func TestFindRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	writeComposeFiles(t, repoRoot, "docker-compose.docker.yml")

	start := filepath.Join(repoRoot, "nested", "child")
	writeTestFile(t, filepath.Join(start, ".keep"), "")

	got, err := findRepoRoot(start)
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	if got != repoRoot {
		t.Fatalf("expected repo root %q, got %q", repoRoot, got)
	}
}

func TestFindRepoRootReturnsErrorWhenMissing(t *testing.T) {
	start := t.TempDir()

	if _, err := findRepoRoot(start); err == nil {
		t.Fatalf("expected repo root error")
	}
}

func getenvOrFail(t *testing.T, key string) string {
	t.Helper()
	return os.Getenv(key)
}

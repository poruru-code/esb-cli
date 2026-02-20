package build

import (
	"testing"

	"github.com/poruru-code/esb-cli/internal/constants"
)

func TestResolveComposeProjectName(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		env         string
		want        string
	}{
		{
			name:        "uses explicit project name",
			projectName: "custom-project",
			env:         "staging",
			want:        "custom-project",
		},
		{
			name:        "falls back to brand slug and lower env",
			projectName: "",
			env:         "Staging",
			want:        "esb-staging",
		},
		{
			name:        "falls back to brand slug when env empty",
			projectName: "",
			env:         " ",
			want:        "esb",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := resolveComposeProjectName(tc.projectName, tc.env)
			if got != tc.want {
				t.Fatalf("resolveComposeProjectName(%q, %q) = %q, want %q", tc.projectName, tc.env, got, tc.want)
			}
		})
	}
}

func TestResolveRuntimeRegistry(t *testing.T) {
	t.Setenv(constants.EnvContainerRegistry, "")
	if got := resolveRuntimeRegistry(" registry:5010/ "); got != "registry:5010/" {
		t.Fatalf("unexpected default runtime registry: %q", got)
	}

	t.Setenv(constants.EnvContainerRegistry, "custom-registry:6000")
	if got := resolveRuntimeRegistry("registry:5010/"); got != "custom-registry:6000/" {
		t.Fatalf("unexpected override runtime registry: %q", got)
	}
}

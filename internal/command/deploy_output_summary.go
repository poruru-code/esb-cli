package command

import (
	"path/filepath"
	"strings"

	"github.com/poruru-code/esb-cli/internal/meta"
)

func resolveDeployOutputSummary(projectDir, outputDir, env string) string {
	trimmedOutput := strings.TrimSpace(outputDir)
	base := strings.TrimSpace(projectDir)
	if base == "" {
		base = "."
	}
	if trimmedOutput == "" {
		return filepath.Join(base, meta.OutputDir, strings.TrimSpace(env))
	}
	resolved := filepath.Clean(trimmedOutput)
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(base, resolved)
	}
	return filepath.Clean(resolved)
}

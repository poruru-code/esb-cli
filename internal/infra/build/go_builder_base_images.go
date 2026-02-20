// Where: cli/internal/infra/build/go_builder_base_images.go
// What: Base image build phase helpers for GoBuilder.
// Why: Keep base image bake details separate from top-level orchestration.
package build

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/poruru-code/esb-cli/internal/constants"
	"github.com/poruru-code/esb-cli/internal/infra/compose"
	"github.com/poruru-code/esb-cli/internal/meta"
)

type baseImageBuildInput struct {
	RepoRoot            string
	LockRoot            string
	RegistryForPush     string
	ImageTag            string
	ImageLabels         map[string]string
	RootFingerprint     string
	NoCache             bool
	Verbose             bool
	IncludeDockerOutput bool
	LambdaBaseTag       string
	Out                 io.Writer
}

func (b *GoBuilder) buildBaseImages(input baseImageBuildInput) error {
	out := resolveBuildOutput(input.Out)
	return withBuildLock(input.LockRoot, "base-images", func() error {
		proxyArgs := dockerBuildArgMap()
		commonDir := filepath.Join(input.RepoRoot, "services", "common")

		buildOS := true
		buildPython := true

		osBaseTag := fmt.Sprintf("%s-os-base:latest", meta.ImagePrefix)
		if !input.NoCache && dockerImageHasLabelValue(
			context.Background(),
			b.Runner,
			input.RepoRoot,
			osBaseTag,
			compose.ESBCAFingerprintLabel,
			input.RootFingerprint,
		) {
			buildOS = false
			if input.Verbose {
				_, _ = fmt.Fprintln(out, "Skipping OS base image build (already exists).")
			}
		}

		pythonBaseTag := fmt.Sprintf("%s-python-base:latest", meta.ImagePrefix)
		if !input.NoCache && dockerImageHasLabelValue(
			context.Background(),
			b.Runner,
			input.RepoRoot,
			pythonBaseTag,
			compose.ESBCAFingerprintLabel,
			input.RootFingerprint,
		) {
			buildPython = false
			if input.Verbose {
				_, _ = fmt.Fprintln(out, "Skipping Python base image build (already exists).")
			}
		}

		lambdaTarget := bakeTarget{
			Name:    "lambda-base",
			Tags:    []string{input.LambdaBaseTag},
			Outputs: resolveBakeOutputs(input.RegistryForPush, true, input.IncludeDockerOutput),
			Labels:  input.ImageLabels,
			Args:    proxyArgs,
			NoCache: input.NoCache,
		}

		baseImageLabels := map[string]string{
			compose.ESBManagedLabel:       "true",
			compose.ESBCAFingerprintLabel: input.RootFingerprint,
		}
		baseTargets := []bakeTarget{lambdaTarget}

		rootCAPath := ""
		if buildOS || buildPython {
			path, err := resolveRootCAPath()
			if err != nil {
				return err
			}
			rootCAPath = path
		}

		if buildOS {
			baseTargets = append(
				baseTargets,
				newRootCABaseTarget("os-base", "Dockerfile.os-base", osBaseTag, commonDir, rootCAPath, baseImageLabels, proxyArgs, input),
			)
		}

		if buildPython {
			baseTargets = append(
				baseTargets,
				newRootCABaseTarget("python-base", "Dockerfile.python-base", pythonBaseTag, commonDir, rootCAPath, baseImageLabels, proxyArgs, input),
			)
		}

		return runBakeGroup(
			context.Background(),
			b.Runner,
			input.RepoRoot,
			input.LockRoot,
			"esb-base",
			baseTargets,
			input.Verbose,
		)
	})
}

func newRootCABaseTarget(
	name string,
	dockerfileName string,
	tag string,
	commonDir string,
	rootCAPath string,
	labels map[string]string,
	proxyArgs map[string]string,
	input baseImageBuildInput,
) bakeTarget {
	return bakeTarget{
		Name:       name,
		Context:    commonDir,
		Dockerfile: filepath.Join(commonDir, dockerfileName),
		Tags:       []string{tag},
		Outputs:    resolveBakeOutputs(input.RegistryForPush, false, input.IncludeDockerOutput),
		Labels:     labels,
		Args: mergeStringMap(proxyArgs, map[string]string{
			constants.BuildArgCAFingerprint: input.RootFingerprint,
			"ROOT_CA_MOUNT_ID":              meta.RootCAMountID,
			"ROOT_CA_CERT_FILENAME":         meta.RootCACertFilename,
		}),
		Secrets: []string{fmt.Sprintf("id=%s,src=%s", meta.RootCAMountID, rootCAPath)},
		NoCache: input.NoCache,
	}
}

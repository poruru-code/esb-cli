// Where: cli/internal/command/artifact.go
// What: CLI adapter for artifact apply operations.
// Why: Reuse shared artifact core logic from esb command.
package command

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/poruru-code/esb-cli/internal/infra/interaction"
	"github.com/poruru-code/esb/pkg/deployops"
)

func runArtifactGenerate(cli CLI, deps Dependencies, out io.Writer) int {
	generatedCLI := cli
	generatedCLI.Deploy = artifactGenerateToDeployFlags(cli.Artifact.Generate)
	overrides := deployRunOverrides{
		buildImages:    boolPtr(cli.Artifact.Generate.BuildImages),
		forceBuildOnly: true,
	}
	return runDeployWithOverrides(generatedCLI, deps, out, overrides)
}

func artifactGenerateToDeployFlags(cmd ArtifactGenerateCmd) DeployCmd {
	return DeployCmd{
		Mode:         cmd.Mode,
		Output:       cmd.Output,
		Manifest:     cmd.Manifest,
		Project:      cmd.Project,
		ComposeFiles: append([]string(nil), cmd.ComposeFiles...),
		ImageURI:     append([]string(nil), cmd.ImageURI...),
		ImageRuntime: append([]string(nil), cmd.ImageRuntime...),
		BuildOnly:    true,
		Bundle:       cmd.Bundle,
		NoCache:      cmd.NoCache,
		Verbose:      cmd.Verbose,
		Emoji:        cmd.Emoji,
		NoEmoji:      cmd.NoEmoji,
		Force:        cmd.Force,
		NoSave:       cmd.NoSave,
	}
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}

func runArtifactApply(cli CLI, deps Dependencies, out io.Writer) int {
	args, err := resolveArtifactApplyInputs(
		cli.Artifact.Apply,
		interaction.IsTerminal(os.Stdin),
		deps.Prompter,
		resolveErrWriter(deps.ErrOut),
	)
	if err != nil {
		return exitWithError(out, err)
	}
	result, err := deployops.Execute(deployops.Input{
		ArtifactPath:  args.Artifact,
		OutputDir:     args.OutputDir,
		SecretEnvPath: args.SecretEnv,
	})
	if err != nil {
		return exitWithError(out, err)
	}
	deployUI := legacyUI(out)
	for _, warning := range result.Warnings {
		deployUI.Warn(warning)
	}
	deployUI.Success("Artifact apply complete")
	return 0
}

func resolveArtifactApplyInputs(
	args ArtifactApplyCmd,
	isTTY bool,
	prompter interaction.Prompter,
	errOut io.Writer,
) (ArtifactApplyCmd, error) {
	artifactPath := strings.TrimSpace(args.Artifact)
	outputDir := strings.TrimSpace(args.OutputDir)
	if isTTY && prompter != nil {
		for artifactPath == "" {
			input, err := prompter.Input("Artifact manifest path (--artifact)", nil)
			if err != nil {
				return ArtifactApplyCmd{}, fmt.Errorf("prompt artifact manifest path: %w", err)
			}
			artifactPath = strings.TrimSpace(input)
			if artifactPath == "" {
				writeWarningf(errOut, "Artifact manifest path is required.\n")
			}
		}
		for outputDir == "" {
			input, err := prompter.Input("Output config directory (--out)", nil)
			if err != nil {
				return ArtifactApplyCmd{}, fmt.Errorf("prompt output config directory: %w", err)
			}
			outputDir = strings.TrimSpace(input)
			if outputDir == "" {
				writeWarningf(errOut, "Output config directory is required.\n")
			}
		}
	}
	args.Artifact = artifactPath
	args.OutputDir = outputDir
	return args, nil
}

// Where: cli/internal/command/deploy_entry.go
// What: Deploy command entry and workflow execution.
// Why: Keep deploy command orchestration separate from input/detail helpers.
package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/poruru-code/esb-cli/internal/domain/state"
	"github.com/poruru-code/esb-cli/internal/infra/build"
	"github.com/poruru-code/esb-cli/internal/infra/compose"
	"github.com/poruru-code/esb-cli/internal/infra/interaction"
	"github.com/poruru-code/esb-cli/internal/infra/ui"
	"github.com/poruru-code/esb-cli/internal/usecase/deploy"
)

// runDeploy executes the 'deploy' command.
func runDeploy(cli CLI, deps Dependencies, out io.Writer) int {
	return runDeployWithOverrides(cli, deps, out, deployRunOverrides{})
}

type deployRunOverrides struct {
	buildImages    *bool
	forceBuildOnly bool
}

type deployRunConfig struct {
	tag         string
	noDeps      bool
	buildImages bool
	buildOnly   bool
}

func runDeployWithOverrides(
	cli CLI,
	deps Dependencies,
	out io.Writer,
	overrides deployRunOverrides,
) int {
	emojiEnabled, err := resolveDeployEmojiEnabled(out, cli.Deploy)
	if err != nil {
		return exitWithError(out, err)
	}

	commandConfig, err := resolveDeployCommandConfig(deps.Deploy, out, emojiEnabled)
	if err != nil {
		return exitWithError(out, err)
	}
	cmd := newDeployCommand(commandConfig)

	inputs, err := resolveDeployInputs(cli, deps)
	if err != nil {
		return exitWithError(out, err)
	}

	if err := cmd.runWithOverrides(inputs, cli.Deploy, overrides); err != nil {
		return exitWithError(out, err)
	}
	return 0
}

type deployCommand struct {
	build         func(build.BuildRequest) error
	applyRuntime  func(state.Context) error
	ui            ui.UserInterface
	composeRunner compose.CommandRunner
	workflow      deployWorkflowDeps
	emojiEnabled  bool
}

type deployWorkflowDeps struct {
	composeProvisioner deploy.ComposeProvisioner
	registryWaiter     deploy.RegistryWaiter
	dockerClient       deploy.DockerClientFactory
}

type deployCommandConfig struct {
	build         func(build.BuildRequest) error
	applyRuntime  func(state.Context) error
	ui            ui.UserInterface
	composeRunner compose.CommandRunner
	workflow      deployWorkflowDeps
	emojiEnabled  bool
}

type deployRuntimeComponent struct {
	applyRuntime   func(state.Context) error
	registryWaiter deploy.RegistryWaiter
	dockerClient   deploy.DockerClientFactory
}

type deployProvisionComponent struct {
	ui                 ui.UserInterface
	composeRunner      compose.CommandRunner
	composeProvisioner deploy.ComposeProvisioner
}

func resolveDeployCommandConfig(
	deployDeps DeployDeps,
	out io.Writer,
	emojiEnabled bool,
) (deployCommandConfig, error) {
	buildFn, err := resolveDeployBuildComponent(deployDeps.Build)
	if err != nil {
		return deployCommandConfig{}, err
	}
	runtimeComponent, err := resolveDeployRuntimeComponent(deployDeps.Runtime)
	if err != nil {
		return deployCommandConfig{}, err
	}
	provisionComponent, err := resolveDeployProvisionComponent(deployDeps.Provision, out, emojiEnabled)
	if err != nil {
		return deployCommandConfig{}, err
	}
	return deployCommandConfig{
		build:         buildFn,
		applyRuntime:  runtimeComponent.applyRuntime,
		ui:            provisionComponent.ui,
		composeRunner: provisionComponent.composeRunner,
		workflow: deployWorkflowDeps{
			composeProvisioner: provisionComponent.composeProvisioner,
			registryWaiter:     runtimeComponent.registryWaiter,
			dockerClient:       runtimeComponent.dockerClient,
		},
		emojiEnabled: emojiEnabled,
	}, nil
}

func resolveDeployBuildComponent(buildDeps DeployBuildDeps) (func(build.BuildRequest) error, error) {
	if buildDeps.Build == nil {
		return nil, errDeployBuilderNotConfigured
	}
	return buildDeps.Build, nil
}

func resolveDeployRuntimeComponent(
	runtimeDeps DeployRuntimeDeps,
) (deployRuntimeComponent, error) {
	if runtimeDeps.ApplyRuntimeEnv == nil {
		return deployRuntimeComponent{}, errors.New("deploy: runtime env applier not configured")
	}
	return deployRuntimeComponent{
		applyRuntime:   runtimeDeps.ApplyRuntimeEnv,
		registryWaiter: deploy.RegistryWaiter(runtimeDeps.RegistryWaiter),
		dockerClient:   deploy.DockerClientFactory(runtimeDeps.DockerClient),
	}, nil
}

func resolveDeployProvisionComponent(
	provisionDeps DeployProvisionDeps,
	out io.Writer,
	emojiEnabled bool,
) (deployProvisionComponent, error) {
	if provisionDeps.ComposeRunner == nil {
		return deployProvisionComponent{}, errors.New("deploy: compose runner not configured")
	}
	deployUIFactory := provisionDeps.NewDeployUI
	if deployUIFactory == nil {
		return deployProvisionComponent{}, errors.New("deploy: deploy ui factory not configured")
	}
	deployUI := deployUIFactory(out, emojiEnabled)
	composeProvisioner := provisionDeps.ComposeProvisioner
	if composeProvisioner == nil && provisionDeps.ComposeProvisionerFactory != nil {
		composeProvisioner = provisionDeps.ComposeProvisionerFactory(deployUI)
	}
	if composeProvisioner == nil {
		return deployProvisionComponent{}, errors.New("deploy: compose provisioner not configured")
	}
	return deployProvisionComponent{
		ui:                 deployUI,
		composeRunner:      provisionDeps.ComposeRunner,
		composeProvisioner: composeProvisioner,
	}, nil
}

func newDeployCommand(config deployCommandConfig) *deployCommand {
	return &deployCommand{
		build:         config.build,
		applyRuntime:  config.applyRuntime,
		ui:            config.ui,
		composeRunner: config.composeRunner,
		workflow:      config.workflow,
		emojiEnabled:  config.emojiEnabled,
	}
}

func (c *deployCommand) runWithOverrides(
	inputs deployInputs,
	flags DeployCmd,
	overrides deployRunOverrides,
) error {
	if len(inputs.Templates) == 0 {
		return errTemplatePathRequired
	}
	runConfig, err := resolveDeployRunConfig(flags, overrides)
	if err != nil {
		return err
	}
	workflow := c.newWorkflow()

	if err := c.runGeneratePhase(workflow, inputs, flags, runConfig); err != nil {
		return err
	}
	manifestPath, err := c.writeArtifactManifest(inputs, flags)
	if err != nil {
		return err
	}
	if runConfig.buildOnly {
		return nil
	}
	return c.runApplyPhase(workflow, inputs, flags, runConfig, manifestPath)
}

func resolveDeployRunConfig(flags DeployCmd, overrides deployRunOverrides) (deployRunConfig, error) {
	buildOnly := flags.BuildOnly || overrides.forceBuildOnly
	if buildOnly && flags.WithDeps {
		return deployRunConfig{}, errors.New("deploy: --with-deps cannot be used with --build-only")
	}
	if buildOnly && strings.TrimSpace(flags.SecretEnv) != "" {
		return deployRunConfig{}, errors.New("deploy: --secret-env cannot be used with --build-only")
	}
	buildImages := true
	if overrides.buildImages != nil {
		buildImages = *overrides.buildImages
	}
	return deployRunConfig{
		tag:         resolveBrandTag(),
		noDeps:      !flags.WithDeps,
		buildImages: buildImages,
		buildOnly:   buildOnly,
	}, nil
}

func (c *deployCommand) newWorkflow() deploy.Workflow {
	workflow := deploy.NewDeployWorkflow(c.build, c.applyRuntime, c.ui, c.composeRunner)
	if c.workflow.composeProvisioner != nil {
		workflow.ComposeProvisioner = c.workflow.composeProvisioner
	}
	if c.workflow.registryWaiter != nil {
		workflow.RegistryWaiter = c.workflow.registryWaiter
	}
	if c.workflow.dockerClient != nil {
		workflow.DockerClient = c.workflow.dockerClient
	}
	return workflow
}

func (c *deployCommand) runGeneratePhase(
	workflow deploy.Workflow,
	inputs deployInputs,
	flags DeployCmd,
	runConfig deployRunConfig,
) error {
	templateCount := len(inputs.Templates)
	for idx, tpl := range inputs.Templates {
		c.renderGeneratePlanBlock(inputs, tpl, idx, templateCount, runConfig.buildImages)
		request := c.newGenerateRequest(inputs, tpl, flags, runConfig)
		if err := workflow.Run(request); err != nil {
			return fmt.Errorf("deploy workflow (%s): %w", tpl.TemplatePath, err)
		}
	}
	return nil
}

func (c *deployCommand) renderGeneratePlanBlock(
	inputs deployInputs,
	tpl deployTemplateInput,
	idx int,
	templateCount int,
	buildImages bool,
) {
	if c.ui == nil {
		return
	}
	title := "Generate plan"
	if templateCount > 1 {
		title = fmt.Sprintf("Generate plan (%d/%d)", idx+1, templateCount)
	}
	outputSummary := resolveDeployOutputSummary(inputs.ProjectDir, tpl.OutputDir, inputs.Env)
	composeFiles := "auto"
	if len(inputs.ComposeFiles) > 0 {
		composeFiles = strings.Join(inputs.ComposeFiles, ", ")
	}
	rows := []ui.KeyValue{
		{Key: "Template", Value: tpl.TemplatePath},
		{Key: "Env", Value: inputs.Env},
		{Key: "Mode", Value: inputs.Mode},
		{Key: "Project", Value: inputs.Project},
		{Key: "Output", Value: outputSummary},
		{Key: "BuildOnly", Value: true},
		{Key: "BuildImages", Value: buildImages},
		{Key: "ComposeFiles", Value: composeFiles},
	}
	c.ui.Block("🧭", title, rows)
}

func (c *deployCommand) writeArtifactManifest(
	inputs deployInputs,
	flags DeployCmd,
) (string, error) {
	manifestPath, err := writeDeployArtifactManifest(
		inputs,
		flags.Bundle,
	)
	if err != nil {
		return "", fmt.Errorf("write artifact manifest: %w", err)
	}
	if c.ui != nil {
		c.ui.Info(fmt.Sprintf("Artifact manifest: %s", manifestPath))
	}
	return manifestPath, nil
}

func (c *deployCommand) runApplyPhase(
	workflow deploy.Workflow,
	inputs deployInputs,
	flags DeployCmd,
	runConfig deployRunConfig,
	manifestPath string,
) error {
	applyTemplate := inputs.Templates[0]
	applyReq := c.newApplyRequest(inputs, applyTemplate, flags, runConfig, manifestPath)
	if err := workflow.Apply(applyReq); err != nil {
		return fmt.Errorf("deploy apply (%s): %w", applyTemplate.TemplatePath, err)
	}
	return nil
}

func (c *deployCommand) newGenerateRequest(
	inputs deployInputs,
	tpl deployTemplateInput,
	flags DeployCmd,
	runConfig deployRunConfig,
) deploy.Request {
	request := buildDeployRequestCommon(inputs, tpl, flags, runConfig)
	request.OutputDir = tpl.OutputDir
	request.Parameters = tpl.Parameters
	request.ImageSources = tpl.ImageSources
	request.ImageRuntimes = tpl.ImageRuntimes
	request.NoCache = flags.NoCache
	request.BuildOnly = true
	request.BuildImages = boolPtr(runConfig.buildImages)
	request.BundleManifest = flags.Bundle
	request.Emoji = c.emojiEnabled
	return request
}

func (c *deployCommand) newApplyRequest(
	inputs deployInputs,
	tpl deployTemplateInput,
	flags DeployCmd,
	runConfig deployRunConfig,
	manifestPath string,
) deploy.Request {
	request := buildDeployRequestCommon(inputs, tpl, flags, runConfig)
	request.ArtifactPath = manifestPath
	request.SecretEnvPath = flags.SecretEnv
	request.OutputDir = tpl.OutputDir
	request.BuildOnly = false
	return request
}

func buildDeployRequestCommon(
	inputs deployInputs,
	tpl deployTemplateInput,
	flags DeployCmd,
	runConfig deployRunConfig,
) deploy.Request {
	return deploy.Request{
		Context:      deployTemplateStateContext(inputs, tpl),
		Tag:          runConfig.tag,
		NoDeps:       runConfig.noDeps,
		Verbose:      flags.Verbose,
		ComposeFiles: inputs.ComposeFiles,
	}
}

func deployTemplateStateContext(
	inputs deployInputs,
	tpl deployTemplateInput,
) state.Context {
	return state.Context{
		ProjectDir:     inputs.ProjectDir,
		TemplatePath:   tpl.TemplatePath,
		Env:            inputs.Env,
		Mode:           inputs.Mode,
		ComposeProject: inputs.Project,
	}
}

func resolveDeployEmojiEnabled(out io.Writer, flags DeployCmd) (bool, error) {
	if flags.Emoji && flags.NoEmoji {
		return false, errors.New("deploy: --emoji and --no-emoji cannot be used together")
	}
	if flags.Emoji {
		return true, nil
	}
	if flags.NoEmoji {
		return false, nil
	}
	if strings.TrimSpace(os.Getenv("NO_EMOJI")) != "" {
		return false, nil
	}
	term := strings.ToLower(strings.TrimSpace(os.Getenv("TERM")))
	if term == "dumb" {
		return false, nil
	}
	if file, ok := out.(*os.File); ok {
		return interaction.IsTerminal(file), nil
	}
	return false, nil
}

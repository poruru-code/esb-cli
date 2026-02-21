// Where: cli/internal/command/deploy_inputs_resolve.go
// What: Deploy input orchestration, env resolution, and mode resolution.
// Why: Keep the main input flow isolated from path/output helper details.
package command

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/poruru-code/esb-cli/internal/constants"
	runtimecfg "github.com/poruru-code/esb-cli/internal/domain/runtime"
	"github.com/poruru-code/esb-cli/internal/infra/config"
	"github.com/poruru-code/esb-cli/internal/infra/envutil"
	"github.com/poruru-code/esb-cli/internal/infra/interaction"
	runtimeinfra "github.com/poruru-code/esb-cli/internal/infra/runtime"
)

type deployInputsResolver struct {
	isTTY               bool
	prompter            interaction.Prompter
	errOut              io.Writer
	repoResolver        func(string) (string, error)
	runtimeResolver     runtimeinfra.EnvResolver
	dockerClientFactory DockerClientFactory
}

type deployRuntimeContext struct {
	repoRoot         string
	selectedStack    deployTargetStack
	composeProject   string
	projectSource    string
	selectedEnv      envChoice
	prevEnv          string
	inferredMode     string
	inferredModeSrc  string
	modeInferenceErr error
}

type runtimeProjectSelection struct {
	prevEnv       string
	prevProject   string
	projectValue  string
	projectSource string
	selectedStack deployTargetStack
}

type deployTemplateOverrideInputs struct {
	imageSources  map[string]string
	imageRuntimes map[string]string
}

type templateInputResolveContext struct {
	deployFlags   DeployCmd
	repoRoot      string
	artifactRoot  string
	templatePaths []string
	prevTemplates map[string]deployTemplateInput
	overrides     deployTemplateOverrideInputs
}

func resolveDeployInputs(cli CLI, deps Dependencies) (deployInputs, error) {
	resolver := newDeployInputsResolver(deps)

	var last deployInputs
	for {
		inputs, err := resolver.resolveInputIteration(cli, last)
		if err != nil {
			return deployInputs{}, err
		}
		accepted, err := resolver.confirmAndPersistResolvedInputs(cli.Deploy.NoSave, inputs)
		if err != nil {
			return deployInputs{}, err
		}
		if accepted {
			return inputs, nil
		}
		last = inputs
	}
}

func (r deployInputsResolver) resolveInputIteration(
	cli CLI,
	last deployInputs,
) (deployInputs, error) {
	runtimeCtx, err := r.resolveRuntimeContext(cli, last)
	if err != nil {
		return deployInputs{}, err
	}

	templatePaths, err := resolveDeployTemplates(
		cli.Template,
		r.isTTY,
		r.prompter,
		previousTemplatePath(last),
		r.errOut,
	)
	if err != nil {
		return deployInputs{}, err
	}

	storedDefaults := loadDeployDefaults(runtimeCtx.repoRoot, templatePaths[0])

	selectedEnv, err := r.resolveSelectedEnv(cli, runtimeCtx, templatePaths[0])
	if err != nil {
		return deployInputs{}, err
	}

	mode, err := r.resolveMode(cli, runtimeCtx, storedDefaults, last)
	if err != nil {
		return deployInputs{}, err
	}
	artifactRoot, err := r.resolveArtifactRoot(cli, last, runtimeCtx.repoRoot, runtimeCtx.composeProject, selectedEnv.Value)
	if err != nil {
		return deployInputs{}, err
	}

	templateInputs, err := r.resolveTemplateInputs(
		cli,
		last,
		runtimeCtx.repoRoot,
		artifactRoot,
		templatePaths,
	)
	if err != nil {
		return deployInputs{}, err
	}
	composeFiles, err := r.resolveComposeFiles(cli, last, runtimeCtx.repoRoot)
	if err != nil {
		return deployInputs{}, err
	}

	return deployInputs{
		ProjectDir:    runtimeCtx.repoRoot,
		ArtifactRoot:  artifactRoot,
		TargetStack:   runtimeCtx.selectedStack.Name,
		Env:           selectedEnv.Value,
		EnvSource:     selectedEnv.Source,
		Mode:          mode,
		Templates:     templateInputs,
		Project:       runtimeCtx.composeProject,
		ProjectSource: runtimeCtx.projectSource,
		ComposeFiles:  composeFiles,
	}, nil
}

func (r deployInputsResolver) confirmAndPersistResolvedInputs(
	noSave bool,
	inputs deployInputs,
) (bool, error) {
	confirmed, err := confirmDeployInputs(inputs, r.isTTY, r.prompter)
	if err != nil {
		return false, err
	}
	if !confirmed {
		return false, nil
	}
	if noSave {
		return true, nil
	}
	for _, tpl := range inputs.Templates {
		if err := saveDeployDefaults(inputs.ProjectDir, tpl, inputs); err != nil {
			writeWarningf(r.errOut, "Warning: failed to save deploy defaults: %v\n", err)
		}
	}
	return true, nil
}

func newDeployInputsResolver(deps Dependencies) deployInputsResolver {
	repoResolver := deps.RepoResolver
	if repoResolver == nil {
		repoResolver = config.ResolveRepoRoot
	}
	runtimeResolver := deps.Deploy.Runtime.RuntimeEnvResolver
	if runtimeResolver == nil {
		runtimeResolver = runtimeinfra.NewEnvResolver()
	}
	return deployInputsResolver{
		isTTY:               interaction.IsTerminal(os.Stdin),
		prompter:            deps.Prompter,
		errOut:              resolveErrWriter(deps.ErrOut),
		repoResolver:        repoResolver,
		runtimeResolver:     runtimeResolver,
		dockerClientFactory: deps.Deploy.Runtime.DockerClient,
	}
}

func (r deployInputsResolver) resolveRuntimeContext(cli CLI, last deployInputs) (deployRuntimeContext, error) {
	repoRoot, err := r.resolveRuntimeRepoRoot()
	if err != nil {
		return deployRuntimeContext{}, err
	}
	selection, err := r.resolveRuntimeProjectSelection(cli, last)
	if err != nil {
		return deployRuntimeContext{}, err
	}
	composeProject, projectSource, err := r.resolveRuntimeComposeProject(cli, selection)
	if err != nil {
		return deployRuntimeContext{}, err
	}
	if strings.TrimSpace(composeProject) == "" {
		return deployRuntimeContext{}, errComposeProjectRequired
	}
	selectedEnv, err := r.resolveRuntimeSelectedEnv(cli, selection, composeProject)
	if err != nil {
		return deployRuntimeContext{}, err
	}
	inferredMode, inferredModeSource, modeInferErr := r.resolveRuntimeModeInference(composeProject)
	return deployRuntimeContext{
		repoRoot:         repoRoot,
		selectedStack:    selection.selectedStack,
		composeProject:   composeProject,
		projectSource:    projectSource,
		selectedEnv:      selectedEnv,
		prevEnv:          selection.prevEnv,
		inferredMode:     inferredMode,
		inferredModeSrc:  inferredModeSource,
		modeInferenceErr: modeInferErr,
	}, nil
}

func (r deployInputsResolver) resolveRuntimeRepoRoot() (string, error) {
	repoRoot, err := r.repoResolver("")
	if err != nil {
		return "", fmt.Errorf("resolve repo root: %w", err)
	}
	if err := config.EnsureProjectConfig(repoRoot); err != nil {
		return "", err
	}
	return repoRoot, nil
}

func (r deployInputsResolver) resolveRuntimeProjectSelection(
	cli CLI,
	last deployInputs,
) (runtimeProjectSelection, error) {
	prevEnv := strings.TrimSpace(last.Env)
	projectValue, projectValueSource, projectExplicit := resolveProjectValue(cli.Deploy.Project)
	selectedStack, err := r.resolveRuntimeSelectedStack(projectExplicit)
	if err != nil {
		return runtimeProjectSelection{}, err
	}
	return runtimeProjectSelection{
		prevEnv:       prevEnv,
		prevProject:   strings.TrimSpace(last.Project),
		projectValue:  projectValue,
		projectSource: projectValueSource,
		selectedStack: selectedStack,
	}, nil
}

func (r deployInputsResolver) resolveRuntimeSelectedStack(
	projectExplicit bool,
) (deployTargetStack, error) {
	if projectExplicit {
		return deployTargetStack{}, nil
	}
	runningStacks, stackDiscoverErr := discoverRunningDeployTargetStacks(r.dockerClientFactory)
	if stackDiscoverErr != nil {
		writeWarningf(r.errOut, "Warning: failed to discover running stacks: %v\n", stackDiscoverErr)
	}
	selectedStack, err := resolveDeployTargetStack(runningStacks, r.isTTY, r.prompter, r.errOut)
	if err != nil {
		return deployTargetStack{}, err
	}
	return selectedStack, nil
}

func (r deployInputsResolver) resolveRuntimeComposeProject(
	cli CLI,
	selection runtimeProjectSelection,
) (string, string, error) {
	composeProject, projectSource := resolveComposeProjectValue(
		selection.projectValue,
		selection.projectSource,
		selection.selectedStack,
		cli.EnvFlag,
		selection.prevEnv,
	)
	if projectSource == "default" {
		selectedProject, selectedSource, err := resolveDeployProject(
			composeProject,
			r.isTTY,
			r.prompter,
			selection.prevProject,
			r.errOut,
		)
		if err != nil {
			return "", "", err
		}
		return selectedProject, selectedSource, nil
	}
	return composeProject, projectSource, nil
}

func (r deployInputsResolver) resolveRuntimeSelectedEnv(
	cli CLI,
	selection runtimeProjectSelection,
	composeProject string,
) (envChoice, error) {
	return resolveDeployEnvFromStack(
		cli.EnvFlag,
		selection.selectedStack,
		composeProject,
		r.isTTY,
		r.prompter,
		r.runtimeResolver,
		selection.prevEnv,
	)
}

func (r deployInputsResolver) resolveRuntimeModeInference(
	composeProject string,
) (string, string, error) {
	return inferDeployModeFromProject(composeProject, r.dockerClientFactory)
}

func (r deployInputsResolver) resolveSelectedEnv(
	cli CLI,
	ctx deployRuntimeContext,
	templatePath string,
) (envChoice, error) {
	selectedEnv := ctx.selectedEnv
	if ctx.inferredMode != "" {
		var err error
		selectedEnv, err = reconcileEnvWithRuntime(
			selectedEnv,
			ctx.composeProject,
			templatePath,
			r.isTTY,
			r.prompter,
			r.runtimeResolver,
			cli.Deploy.Force,
			r.errOut,
		)
		if err != nil {
			return envChoice{}, err
		}
	}
	if strings.TrimSpace(selectedEnv.Value) == "" {
		chosen, err := resolveDeployEnv("", r.isTTY, r.prompter, ctx.prevEnv)
		if err != nil {
			return envChoice{}, err
		}
		selectedEnv = chosen
	}
	return selectedEnv, nil
}

func (r deployInputsResolver) resolveMode(
	cli CLI,
	ctx deployRuntimeContext,
	stored storedDeployDefaults,
	last deployInputs,
) (string, error) {
	previousSelectedMode := strings.TrimSpace(last.Mode)
	prevMode := previousSelectedMode
	if prevMode == "" {
		prevMode = strings.TrimSpace(stored.Mode)
	}

	flagMode := strings.TrimSpace(cli.Deploy.Mode)
	if flagMode != "" {
		normalized, err := runtimecfg.NormalizeMode(flagMode)
		if err != nil {
			return "", fmt.Errorf("normalize mode: %w", err)
		}
		flagMode = normalized
	}

	if ctx.modeInferenceErr != nil {
		writeWarningf(r.errOut, "Warning: failed to infer runtime mode: %v\n", ctx.modeInferenceErr)
	}

	switch {
	case ctx.inferredMode != "":
		if flagMode != "" && ctx.inferredMode != flagMode {
			return resolveDeployModeConflict(
				ctx.inferredMode,
				ctx.inferredModeSrc,
				flagMode,
				r.isTTY,
				r.prompter,
				previousSelectedMode,
				r.errOut,
			)
		}
		return ctx.inferredMode, nil
	case flagMode != "":
		return flagMode, nil
	default:
		return resolveDeployMode(r.isTTY, r.prompter, prevMode, r.errOut)
	}
}

func (r deployInputsResolver) resolveArtifactRoot(
	cli CLI,
	last deployInputs,
	repoRoot string,
	project string,
	env string,
) (string, error) {
	prev := strings.TrimSpace(last.ArtifactRoot)
	return resolveDeployArtifactRoot(
		cli.Deploy.ArtifactRoot,
		r.isTTY,
		r.prompter,
		prev,
		repoRoot,
		project,
		env,
	)
}

func (r deployInputsResolver) resolveTemplateInputs(
	cli CLI,
	last deployInputs,
	repoRoot string,
	artifactRoot string,
	templatePaths []string,
) ([]deployTemplateInput, error) {
	overrides, err := parseDeployTemplateOverrideInputs(cli.Deploy)
	if err != nil {
		return nil, err
	}
	ctx := templateInputResolveContext{
		deployFlags:   cli.Deploy,
		repoRoot:      repoRoot,
		artifactRoot:  artifactRoot,
		templatePaths: templatePaths,
		prevTemplates: buildPreviousTemplateInputMap(last.Templates),
		overrides:     overrides,
	}
	templateInputs := make([]deployTemplateInput, 0, len(templatePaths))
	for _, templatePath := range templatePaths {
		templateInput, err := r.resolveSingleTemplateInput(templatePath, ctx)
		if err != nil {
			return nil, err
		}
		templateInputs = append(templateInputs, templateInput)
	}
	return templateInputs, nil
}

func parseDeployTemplateOverrideInputs(deployFlags DeployCmd) (deployTemplateOverrideInputs, error) {
	imageSources, err := parseFunctionOverrideFlag(deployFlags.ImageURI, "--image-uri")
	if err != nil {
		return deployTemplateOverrideInputs{}, err
	}
	imageRuntimes, err := parseFunctionOverrideFlag(deployFlags.ImageRuntime, "--image-runtime")
	if err != nil {
		return deployTemplateOverrideInputs{}, err
	}
	return deployTemplateOverrideInputs{
		imageSources:  imageSources,
		imageRuntimes: imageRuntimes,
	}, nil
}

func buildPreviousTemplateInputMap(
	templates []deployTemplateInput,
) map[string]deployTemplateInput {
	prevTemplates := map[string]deployTemplateInput{}
	for _, tpl := range templates {
		prevTemplates[tpl.TemplatePath] = tpl
	}
	return prevTemplates
}

func (r deployInputsResolver) resolveSingleTemplateInput(
	templatePath string,
	ctx templateInputResolveContext,
) (deployTemplateInput, error) {
	storedTemplate := loadDeployDefaults(ctx.repoRoot, templatePath)

	prevParams := resolvePreviousTemplateParameters(
		templatePath,
		ctx.prevTemplates,
		storedTemplate.Params,
	)
	params, err := promptTemplateParameters(templatePath, r.isTTY, r.prompter, prevParams, r.errOut)
	if err != nil {
		return deployTemplateInput{}, err
	}
	imageTargets, err := discoverImageRuntimePromptTargets(templatePath, params)
	if err != nil {
		return deployTemplateInput{}, err
	}
	imageFunctionNames := make([]string, 0, len(imageTargets))
	defaultImageSources := map[string]string{}
	for _, target := range imageTargets {
		imageFunctionNames = append(imageFunctionNames, target.Name)
		if source := strings.TrimSpace(target.ImageSource); source != "" {
			defaultImageSources[target.Name] = source
		}
	}
	overrideImageSources := filterFunctionOverrides(ctx.overrides.imageSources, imageFunctionNames)
	templateImageSources := mergeTemplateImageSources(defaultImageSources, overrideImageSources)
	templateImageRuntimeOverrides := filterFunctionOverrides(
		ctx.overrides.imageRuntimes,
		imageFunctionNames,
	)

	prevImageRuntimes := resolvePreviousTemplateImageRuntimes(
		templatePath,
		ctx.prevTemplates,
		storedTemplate.ImageRuntimes,
	)
	imageRuntimes, err := promptTemplateImageRuntimes(
		templatePath,
		params,
		templateImageSources,
		r.isTTY,
		r.prompter,
		prevImageRuntimes,
		templateImageRuntimeOverrides,
		r.errOut,
	)
	if err != nil {
		return deployTemplateInput{}, err
	}
	artifactID, err := deriveTemplateArtifactID(ctx.repoRoot, templatePath, params)
	if err != nil {
		return deployTemplateInput{}, err
	}
	outputDir := deriveTemplateArtifactOutputDir(ctx.artifactRoot, artifactID)

	return deployTemplateInput{
		TemplatePath:  templatePath,
		OutputDir:     outputDir,
		Parameters:    params,
		ImageSources:  templateImageSources,
		ImageRuntimes: imageRuntimes,
	}, nil
}

func mergeTemplateImageSources(
	defaults map[string]string,
	overrides map[string]string,
) map[string]string {
	if len(defaults) == 0 && len(overrides) == 0 {
		return nil
	}
	merged := make(map[string]string, len(defaults)+len(overrides))
	for key, value := range defaults {
		merged[key] = value
	}
	for key, value := range overrides {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		merged[key] = trimmed
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func (r deployInputsResolver) resolveComposeFiles(
	cli CLI,
	last deployInputs,
	repoRoot string,
) ([]string, error) {
	return resolveDeployComposeFiles(
		cli.Deploy.ComposeFiles,
		r.isTTY,
		r.prompter,
		last.ComposeFiles,
		repoRoot,
	)
}

func resolvePreviousTemplateParameters(
	templatePath string,
	prevTemplates map[string]deployTemplateInput,
	storedParams map[string]string,
) map[string]string {
	if prev, ok := prevTemplates[templatePath]; ok && len(prev.Parameters) > 0 {
		return prev.Parameters
	}
	return storedParams
}

func resolvePreviousTemplateImageRuntimes(
	templatePath string,
	prevTemplates map[string]deployTemplateInput,
	storedImageRuntimes map[string]string,
) map[string]string {
	if prev, ok := prevTemplates[templatePath]; ok && len(prev.ImageRuntimes) > 0 {
		return prev.ImageRuntimes
	}
	return storedImageRuntimes
}

func resolveProjectValue(flagProject string) (string, string, bool) {
	projectValue := strings.TrimSpace(flagProject)
	if projectValue != "" {
		return projectValue, "flag", true
	}
	if envProject := strings.TrimSpace(os.Getenv(constants.EnvProjectName)); envProject != "" {
		return envProject, "env", true
	}
	if hostProject, err := envutil.GetHostEnv(constants.HostSuffixProject); err == nil {
		if trimmed := strings.TrimSpace(hostProject); trimmed != "" {
			return trimmed, "host", true
		}
	}
	return "", "", false
}

func resolveComposeProjectValue(
	projectValue string,
	projectSource string,
	selectedStack deployTargetStack,
	flagEnv string,
	prevEnv string,
) (string, string) {
	if strings.TrimSpace(projectValue) != "" {
		return projectValue, projectSource
	}
	if strings.TrimSpace(selectedStack.Project) != "" {
		return selectedStack.Project, "stack"
	}
	defaultEnv := strings.TrimSpace(flagEnv)
	if defaultEnv == "" {
		defaultEnv = strings.TrimSpace(selectedStack.Env)
	}
	if defaultEnv == "" {
		defaultEnv = strings.TrimSpace(prevEnv)
	}
	return defaultDeployProject(defaultEnv), "default"
}

func previousTemplatePath(last deployInputs) string {
	if len(last.Templates) == 0 {
		return ""
	}
	return last.Templates[0].TemplatePath
}

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

func resolveDeployInputs(cli CLI, deps Dependencies) (deployInputs, error) {
	resolver := newDeployInputsResolver(deps)

	var last deployInputs
	for {
		runtimeCtx, err := resolver.resolveRuntimeContext(cli, last)
		if err != nil {
			return deployInputs{}, err
		}

		templatePaths, err := resolveDeployTemplates(
			cli.Template,
			resolver.isTTY,
			resolver.prompter,
			previousTemplatePath(last),
			resolver.errOut,
		)
		if err != nil {
			return deployInputs{}, err
		}

		storedDefaults := loadDeployDefaults(runtimeCtx.repoRoot, templatePaths[0])

		selectedEnv, err := resolver.resolveSelectedEnv(cli, runtimeCtx, templatePaths[0])
		if err != nil {
			return deployInputs{}, err
		}
		envChanged := selectedEnv.Value != runtimeCtx.prevEnv

		mode, err := resolver.resolveMode(cli, runtimeCtx, storedDefaults, last)
		if err != nil {
			return deployInputs{}, err
		}

		templateInputs, err := resolver.resolveTemplateInputs(
			cli,
			last,
			runtimeCtx.repoRoot,
			templatePaths,
			envChanged,
		)
		if err != nil {
			return deployInputs{}, err
		}

		inputs := deployInputs{
			ProjectDir:    runtimeCtx.repoRoot,
			TargetStack:   runtimeCtx.selectedStack.Name,
			Env:           selectedEnv.Value,
			EnvSource:     selectedEnv.Source,
			Mode:          mode,
			Templates:     templateInputs,
			Project:       runtimeCtx.composeProject,
			ProjectSource: runtimeCtx.projectSource,
			ComposeFiles:  normalizeComposeFiles(cli.Deploy.ComposeFiles, runtimeCtx.repoRoot),
		}

		confirmed, err := confirmDeployInputs(inputs, resolver.isTTY, resolver.prompter)
		if err != nil {
			return deployInputs{}, err
		}
		if confirmed {
			if !cli.Deploy.NoSave {
				for _, tpl := range templateInputs {
					if err := saveDeployDefaults(runtimeCtx.repoRoot, tpl, inputs); err != nil {
						writeWarningf(resolver.errOut, "Warning: failed to save deploy defaults: %v\n", err)
					}
				}
			}
			return inputs, nil
		}
		last = inputs
	}
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
	repoRoot, err := r.repoResolver("")
	if err != nil {
		return deployRuntimeContext{}, fmt.Errorf("resolve repo root: %w", err)
	}
	if err := config.EnsureProjectConfig(repoRoot); err != nil {
		return deployRuntimeContext{}, err
	}

	prevEnv := strings.TrimSpace(last.Env)
	projectValue, projectValueSource, projectExplicit := resolveProjectValue(cli.Deploy.Project)

	var selectedStack deployTargetStack
	if !projectExplicit {
		runningStacks, stackDiscoverErr := discoverRunningDeployTargetStacks(r.dockerClientFactory)
		if stackDiscoverErr != nil {
			writeWarningf(r.errOut, "Warning: failed to discover running stacks: %v\n", stackDiscoverErr)
		}
		selectedStack, err = resolveDeployTargetStack(runningStacks, r.isTTY, r.prompter)
		if err != nil {
			return deployRuntimeContext{}, err
		}
	}

	composeProject, projectSource := resolveComposeProjectValue(
		projectValue,
		projectValueSource,
		selectedStack,
		cli.EnvFlag,
		prevEnv,
	)
	if strings.TrimSpace(composeProject) == "" {
		return deployRuntimeContext{}, errComposeProjectRequired
	}

	selectedEnv, err := resolveDeployEnvFromStack(
		cli.EnvFlag,
		selectedStack,
		composeProject,
		r.isTTY,
		r.prompter,
		r.runtimeResolver,
		prevEnv,
	)
	if err != nil {
		return deployRuntimeContext{}, err
	}

	inferredMode, inferredModeSource, modeInferErr := inferDeployModeFromProject(composeProject, r.dockerClientFactory)
	return deployRuntimeContext{
		repoRoot:         repoRoot,
		selectedStack:    selectedStack,
		composeProject:   composeProject,
		projectSource:    projectSource,
		selectedEnv:      selectedEnv,
		prevEnv:          prevEnv,
		inferredMode:     inferredMode,
		inferredModeSrc:  inferredModeSource,
		modeInferenceErr: modeInferErr,
	}, nil
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
	prevMode := strings.TrimSpace(last.Mode)
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
			writeWarningf(
				r.errOut,
				"Warning: running project uses %s mode; ignoring --mode %s (source: %s)\n",
				ctx.inferredMode,
				flagMode,
				ctx.inferredModeSrc,
			)
		}
		return ctx.inferredMode, nil
	case flagMode != "":
		return flagMode, nil
	default:
		return resolveDeployMode("", r.isTTY, r.prompter, prevMode, r.errOut)
	}
}

func (r deployInputsResolver) resolveTemplateInputs(
	cli CLI,
	last deployInputs,
	repoRoot string,
	templatePaths []string,
	envChanged bool,
) ([]deployTemplateInput, error) {
	if len(templatePaths) > 1 && strings.TrimSpace(cli.Deploy.Output) != "" {
		return nil, errMultipleTemplateOutput
	}
	cliImageSources, err := parseFunctionOverrideFlag(cli.Deploy.ImageURI, "--image-uri")
	if err != nil {
		return nil, err
	}
	cliImageRuntimes, err := parseFunctionOverrideFlag(cli.Deploy.ImageRuntime, "--image-runtime")
	if err != nil {
		return nil, err
	}

	prevTemplates := map[string]deployTemplateInput{}
	for _, tpl := range last.Templates {
		prevTemplates[tpl.TemplatePath] = tpl
	}
	outputKeyCounts := map[string]int{}
	templateInputs := make([]deployTemplateInput, 0, len(templatePaths))
	for _, templatePath := range templatePaths {
		storedTemplate := loadDeployDefaults(repoRoot, templatePath)
		outputDir := ""
		if len(templatePaths) == 1 {
			prevOutput := ""
			if prev, ok := prevTemplates[templatePath]; ok && strings.TrimSpace(prev.OutputDir) != "" {
				prevOutput = prev.OutputDir
			} else if strings.TrimSpace(storedTemplate.OutputDir) != "" {
				prevOutput = storedTemplate.OutputDir
			}
			if envChanged && strings.TrimSpace(cli.Deploy.Output) == "" {
				prevOutput = ""
			}
			outputDir, err = resolveDeployOutput(
				cli.Deploy.Output,
				r.isTTY,
				r.prompter,
				prevOutput,
			)
			if err != nil {
				return nil, err
			}
		} else {
			outputDir = deriveMultiTemplateOutputDir(templatePath, outputKeyCounts)
		}

		prevParams := storedTemplate.Params
		if prev, ok := prevTemplates[templatePath]; ok && len(prev.Parameters) > 0 {
			prevParams = prev.Parameters
		}
		params, err := promptTemplateParameters(templatePath, r.isTTY, r.prompter, prevParams, r.errOut)
		if err != nil {
			return nil, err
		}
		imageFunctionNames, err := discoverImageFunctionNames(templatePath, params)
		if err != nil {
			return nil, err
		}
		templateImageSources := filterFunctionOverrides(cliImageSources, imageFunctionNames)
		templateImageRuntimeOverrides := filterFunctionOverrides(cliImageRuntimes, imageFunctionNames)

		prevImageRuntimes := storedTemplate.ImageRuntimes
		if prev, ok := prevTemplates[templatePath]; ok && len(prev.ImageRuntimes) > 0 {
			prevImageRuntimes = prev.ImageRuntimes
		}
		imageRuntimes, err := promptTemplateImageRuntimes(
			templatePath,
			params,
			r.isTTY,
			r.prompter,
			prevImageRuntimes,
			templateImageRuntimeOverrides,
			r.errOut,
		)
		if err != nil {
			return nil, err
		}

		templateInputs = append(templateInputs, deployTemplateInput{
			TemplatePath:  templatePath,
			OutputDir:     outputDir,
			Parameters:    params,
			ImageSources:  templateImageSources,
			ImageRuntimes: imageRuntimes,
		})
	}
	return templateInputs, nil
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

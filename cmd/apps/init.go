package apps

import (
	"bytes"
	"context"
	_ "embed" // For embedding template files
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/cmd/apps/internal"
	"github.com/databricks/cli/cmd/apps/legacytemplates"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apps/features"
	"github.com/databricks/cli/libs/apps/initializer"
	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

const (
	templatePathEnvVar = "DATABRICKS_APPKIT_TEMPLATE_PATH"
	defaultTemplateURL = "https://github.com/databricks/appkit/tree/main/template"
	templateTypeAppKit = "appkit"
)

func newInitCmd() *cobra.Command {
	var (
		templatePath    string
		branch          string
		name            string
		warehouseID     string
		servingEndpoint string
		experimentID    string
		databaseName    string
		instanceName    string
		ucVolume        string
		description     string
		outputDir       string
		featuresFlag    []string
		deploy          bool
		run             string
	)

	cmd := &cobra.Command{
		Use:    "init",
		Short:  "Initialize a new AppKit application from a template",
		Hidden: true,
		Long: `Initialize a new application from a template.

When run without arguments, an interactive prompt allows you to choose a framework:
  - AppKit (TypeScript): Modern TypeScript framework (default)
  - Dash, Flask, Gradio, Node.js, Shiny, Streamlit: Python/Node.js frameworks

When run with --name, runs in non-interactive mode (all required flags must be provided).

Examples:
  # Interactive mode - choose template type (recommended)
  databricks apps init

  # Non-interactive AppKit with flags
  databricks apps init --name my-app

  # With analytics feature (requires --warehouse-id)
  databricks apps init --name my-app --features=analytics --warehouse-id=abc123

  # Create, deploy, and run with dev-remote
  databricks apps init --name my-app --deploy --run=dev-remote

  # Use a legacy template by path identifier
  databricks apps init --template streamlit-chatbot-app
  databricks apps init --template dash-data-app
  databricks apps init --template gradio-hello-world-app

  # With a custom template from a local path
  databricks apps init --template /path/to/template --name my-app

  # With a GitHub URL
  databricks apps init --template https://github.com/user/repo --name my-app

Feature dependencies:
  Some features require additional flags:
  - analytics: requires --warehouse-id (SQL Warehouse ID)

Environment variables:
  DATABRICKS_APPKIT_TEMPLATE_PATH  Override the default template source`,
		Args:    cobra.NoArgs,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return runCreate(ctx, createOptions{
				templatePath:    templatePath,
				branch:          branch,
				name:            name,
				nameProvided:    cmd.Flags().Changed("name"),
				warehouseID:     warehouseID,
				servingEndpoint: servingEndpoint,
				experimentID:    experimentID,
				databaseName:    databaseName,
				instanceName:    instanceName,
				ucVolume:        ucVolume,
				description:     description,
				outputDir:       outputDir,
				features:        featuresFlag,
				deploy:          deploy,
				deployChanged:   cmd.Flags().Changed("deploy"),
				run:             run,
				runChanged:      cmd.Flags().Changed("run"),
				featuresChanged: cmd.Flags().Changed("features"),
			})
		},
	}

	// General flags
	cmd.Flags().StringVar(&templatePath, "template", "", "Template identifier (legacy template path), local directory, or GitHub URL")
	cmd.Flags().StringVar(&branch, "branch", "", "Git branch or tag (for GitHub templates)")
	cmd.Flags().StringVar(&name, "name", "", "Project name (prompts if not provided)")
	cmd.Flags().StringVar(&description, "description", "", "App description")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the project to")
	cmd.Flags().StringSliceVar(&featuresFlag, "features", nil, "Features to enable (comma-separated). Available: "+strings.Join(features.GetFeatureIDs(), ", "))

	// Resource flags (for legacy templates)
	cmd.Flags().StringVar(&warehouseID, "warehouse-id", "", "[Resource] SQL warehouse ID")
	cmd.Flags().StringVar(&servingEndpoint, "serving-endpoint", "", "[Resource] Model serving endpoint name")
	cmd.Flags().StringVar(&experimentID, "experiment-id", "", "[Resource] MLflow experiment ID")
	cmd.Flags().StringVar(&databaseName, "database-name", "", "[Resource] Lakebase database name")
	cmd.Flags().StringVar(&instanceName, "instance-name", "", "[Resource] Lakebase database instance name")
	cmd.Flags().StringVar(&ucVolume, "uc-volume", "", "[Resource] Unity Catalog volume path")

	// Post-creation flags
	cmd.Flags().BoolVar(&deploy, "deploy", false, "Deploy the app after creation")
	cmd.Flags().StringVar(&run, "run", "", "Run the app after creation (none, dev, dev-remote)")

	return cmd
}

type createOptions struct {
	templatePath    string
	branch          string
	name            string
	nameProvided    bool // true if --name flag was explicitly set (enables "flags mode")
	warehouseID     string
	servingEndpoint string
	experimentID    string
	databaseName    string
	instanceName    string
	ucVolume        string
	description     string
	outputDir       string
	features        []string
	deploy          bool
	deployChanged   bool // true if --deploy flag was explicitly set
	run             string
	runChanged      bool // true if --run flag was explicitly set
	featuresChanged bool // true if --features flag was explicitly set
}

// templateVars holds the variables for template substitution.
type templateVars struct {
	ProjectName    string
	AppName        string
	SQLWarehouseID string
	AppDescription string
	Profile        string
	WorkspaceHost  string
	PluginImport   string
	PluginUsage    string
	// Feature resource fragments (aggregated from selected features)
	BundleVariables  string
	BundleResources  string
	TargetVariables  string
	AppEnv           string
	DotEnv           string
	DotEnvExample    string
	ResourceBindings string   // For databricks.yml resource bindings
	UserAPIScopes    []string // User API scopes for legacy templates
}

// featureFragments holds aggregated content from feature resource files.
type featureFragments struct {
	BundleVariables string
	BundleResources string
	TargetVariables string
	AppEnv          string
	DotEnv          string
	DotEnvExample   string
}

// promptForFeaturesAndDeps prompts for features and their dependencies.
// Used when the template uses the feature-fragment system.
// skipDeployRunPrompt indicates whether to skip prompting for deploy/run (because flags were provided).
func promptForFeaturesAndDeps(ctx context.Context, preSelectedFeatures []string, skipDeployRunPrompt bool) (*prompt.CreateProjectConfig, error) {
	config := &prompt.CreateProjectConfig{
		Dependencies: make(map[string]string),
		Features:     preSelectedFeatures,
	}
	theme := prompt.AppkitTheme()

	// Step 1: Feature selection (skip if features already provided via flag)
	if len(config.Features) == 0 && len(features.AvailableFeatures) > 0 {
		options := make([]huh.Option[string], 0, len(features.AvailableFeatures))
		for _, f := range features.AvailableFeatures {
			label := f.Name + " - " + f.Description
			options = append(options, huh.NewOption(label, f.ID))
		}

		err := huh.NewMultiSelect[string]().
			Title("Select features").
			Description("space to toggle, enter to confirm").
			Options(options...).
			Value(&config.Features).
			Height(8).
			WithTheme(theme).
			Run()
		if err != nil {
			return nil, err
		}
		if len(config.Features) == 0 {
			prompt.PrintAnswered(ctx, "Features", "None")
		} else {
			prompt.PrintAnswered(ctx, "Features", fmt.Sprintf("%d selected", len(config.Features)))
		}
	}

	// Step 2: Prompt for feature dependencies
	deps := features.CollectDependencies(config.Features)
	for _, dep := range deps {
		// Special handling for SQL warehouse - show picker instead of text input
		if dep.ID == "sql_warehouse_id" {
			warehouseID, err := prompt.PromptForWarehouse(ctx)
			if err != nil {
				return nil, err
			}
			config.Dependencies[dep.ID] = warehouseID
			continue
		}

		var value string
		description := dep.Description
		if !dep.Required {
			description += " (optional)"
		}

		input := huh.NewInput().
			Title(dep.Title).
			Description(description).
			Placeholder(dep.Placeholder).
			Value(&value)

		if dep.Required {
			input = input.Validate(func(s string) error {
				if s == "" {
					return errors.New("this field is required")
				}
				return nil
			})
		}

		if err := input.WithTheme(theme).Run(); err != nil {
			return nil, err
		}
		prompt.PrintAnswered(ctx, dep.Title, value)
		config.Dependencies[dep.ID] = value
	}

	// Step 3: Description
	config.Description = prompt.DefaultAppDescription
	err := huh.NewInput().
		Title("Description").
		Placeholder(prompt.DefaultAppDescription).
		Value(&config.Description).
		WithTheme(theme).
		Run()
	if err != nil {
		return nil, err
	}
	if config.Description == "" {
		config.Description = prompt.DefaultAppDescription
	}
	prompt.PrintAnswered(ctx, "Description", config.Description)

	// Step 4: Deploy and run options (skip if any deploy/run flag was provided)
	if !skipDeployRunPrompt {
		config.Deploy, config.RunMode, err = prompt.PromptForDeployAndRun(ctx)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

// loadFeatureFragments reads and aggregates resource fragments for selected features.
// templateDir is the path to the template directory (containing the "features" subdirectory).
func loadFeatureFragments(templateDir string, featureIDs []string, vars templateVars) (*featureFragments, error) {
	featuresDir := filepath.Join(templateDir, "features")

	resourceFiles := features.CollectResourceFiles(featureIDs)
	if len(resourceFiles) == 0 {
		return &featureFragments{}, nil
	}

	var bundleVarsList, bundleResList, targetVarsList, appEnvList, dotEnvList, dotEnvExampleList []string

	for _, rf := range resourceFiles {
		if rf.BundleVariables != "" {
			content, err := readAndSubstitute(filepath.Join(featuresDir, rf.BundleVariables), vars)
			if err != nil {
				return nil, fmt.Errorf("read bundle variables: %w", err)
			}
			bundleVarsList = append(bundleVarsList, content)
		}
		if rf.BundleResources != "" {
			content, err := readAndSubstitute(filepath.Join(featuresDir, rf.BundleResources), vars)
			if err != nil {
				return nil, fmt.Errorf("read bundle resources: %w", err)
			}
			bundleResList = append(bundleResList, content)
		}
		if rf.TargetVariables != "" {
			content, err := readAndSubstitute(filepath.Join(featuresDir, rf.TargetVariables), vars)
			if err != nil {
				return nil, fmt.Errorf("read target variables: %w", err)
			}
			targetVarsList = append(targetVarsList, content)
		}
		if rf.AppEnv != "" {
			content, err := readAndSubstitute(filepath.Join(featuresDir, rf.AppEnv), vars)
			if err != nil {
				return nil, fmt.Errorf("read app env: %w", err)
			}
			appEnvList = append(appEnvList, content)
		}
		if rf.DotEnv != "" {
			content, err := readAndSubstitute(filepath.Join(featuresDir, rf.DotEnv), vars)
			if err != nil {
				return nil, fmt.Errorf("read dotenv: %w", err)
			}
			dotEnvList = append(dotEnvList, content)
		}
		if rf.DotEnvExample != "" {
			content, err := readAndSubstitute(filepath.Join(featuresDir, rf.DotEnvExample), vars)
			if err != nil {
				return nil, fmt.Errorf("read dotenv example: %w", err)
			}
			dotEnvExampleList = append(dotEnvExampleList, content)
		}
	}

	// Join fragments (they already have proper indentation from the fragment files)
	return &featureFragments{
		BundleVariables: strings.TrimSuffix(strings.Join(bundleVarsList, ""), "\n"),
		BundleResources: strings.TrimSuffix(strings.Join(bundleResList, ""), "\n"),
		TargetVariables: strings.TrimSuffix(strings.Join(targetVarsList, ""), "\n"),
		AppEnv:          strings.TrimSuffix(strings.Join(appEnvList, ""), "\n"),
		DotEnv:          strings.TrimSuffix(strings.Join(dotEnvList, ""), "\n"),
		DotEnvExample:   strings.TrimSuffix(strings.Join(dotEnvExampleList, ""), "\n"),
	}, nil
}

// readAndSubstitute reads a file and applies variable substitution.
func readAndSubstitute(path string, vars templateVars) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // Fragment file doesn't exist, skip it
		}
		return "", err
	}
	return substituteVars(string(content), vars), nil
}

// cloneRepo clones a git repository to a temporary directory.
func cloneRepo(ctx context.Context, repoURL, branch string) (string, error) {
	tempDir, err := os.MkdirTemp("", "appkit-template-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	if err := git.Clone(ctx, repoURL, branch, tempDir); err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	return tempDir, nil
}

// resolveTemplate resolves a template path, handling both local paths and GitHub URLs.
// Returns the local path to use, a cleanup function (for temp dirs), and any error.
func resolveTemplate(ctx context.Context, templatePath, branch string) (localPath string, cleanup func(), err error) {
	// Case 1: Local path - return as-is
	if !strings.HasPrefix(templatePath, "https://") {
		return templatePath, nil, nil
	}

	// Case 2: GitHub URL - parse and clone
	repoURL, subdir, urlBranch := git.ParseGitHubURL(templatePath)
	if branch == "" {
		branch = urlBranch // Use branch from URL if not overridden by flag
	}

	// Clone to temp dir with spinner
	var tempDir string
	err = prompt.RunWithSpinnerCtx(ctx, "Cloning template...", func() error {
		var cloneErr error
		tempDir, cloneErr = cloneRepo(ctx, repoURL, branch)
		return cloneErr
	})
	if err != nil {
		return "", nil, err
	}

	cleanup = func() { os.RemoveAll(tempDir) }

	// Return path to subdirectory if specified
	if subdir != "" {
		return filepath.Join(tempDir, subdir), cleanup, nil
	}
	return tempDir, cleanup, nil
}

// deployRunConfig handles deploy and run mode determination.
type deployRunConfig struct {
	// From flags
	deploy        bool
	deployChanged bool
	run           string
	runChanged    bool
	// Context
	isInteractive bool
}

// resolve determines the final deploy and run mode values.
// It handles the logic of using flags vs prompting based on interactive mode.
func (c *deployRunConfig) resolve(ctx context.Context) (bool, prompt.RunMode, error) {
	// Parse flags first
	shouldDeploy, runMode, err := internal.ParseDeployAndRunFlags(c.deploy, c.run)
	if err != nil {
		return false, prompt.RunModeNone, err
	}

	// Prompt if interactive and no flags were set
	skipPrompt := c.deployChanged || c.runChanged
	if c.isInteractive && !skipPrompt {
		shouldDeploy, runMode, err = prompt.PromptForDeployAndRun(ctx)
		if err != nil {
			return false, prompt.RunModeNone, err
		}
	}

	return shouldDeploy, runMode, nil
}

func runCreate(ctx context.Context, opts createOptions) error {
	var selectedFeatures []string
	var dependencies map[string]string
	var shouldDeploy bool
	var runMode prompt.RunMode
	isInteractive := cmdio.IsPromptSupported(ctx)

	// Get workspace host and profile from context early (needed for legacy templates)
	workspaceHost := ""
	profile := ""
	if w := cmdctx.WorkspaceClient(ctx); w != nil && w.Config != nil {
		workspaceHost = w.Config.Host
		profile = w.Config.Profile
	}

	// Use features from flags if provided
	if len(opts.features) > 0 {
		selectedFeatures = opts.features
	}

	// Step 0: Check if --template flag specifies a legacy template path
	if opts.templatePath != "" {
		// Check if it's a legacy template identifier (not a URL or local path)
		if !strings.HasPrefix(opts.templatePath, "https://") && !strings.HasPrefix(opts.templatePath, "/") && !strings.HasPrefix(opts.templatePath, "./") && !strings.HasPrefix(opts.templatePath, "../") {
			templates, err := legacytemplates.LoadLegacyTemplates()
			if err != nil {
				return err
			}

			// Check if the template path matches a legacy template
			if legacyTemplate := legacytemplates.FindLegacyTemplateByPath(templates, opts.templatePath); legacyTemplate != nil {
				log.Infof(ctx, "Using legacy template: %s", opts.templatePath)
				absOutputDir, shouldDeploy, runMode, err := legacytemplates.HandleLegacyTemplateInit(ctx, legacyTemplate, opts.name, opts.nameProvided, opts.outputDir, opts.warehouseID, opts.servingEndpoint, opts.experimentID, opts.instanceName, opts.databaseName, opts.ucVolume, opts.deploy, opts.deployChanged, opts.run, opts.runChanged, isInteractive, workspaceHost, profile)
				if err != nil {
					return err
				}
				// Extract project name from the absolute output directory
				projectName := filepath.Base(absOutputDir)
				return runPostCreationSteps(ctx, absOutputDir, projectName, 0, shouldDeploy, runMode)
			}
		}
	}

	// Step 1: Prompt for template type (AppKit or framework type) in interactive mode
	selectedTemplateType := templateTypeAppKit // default
	if isInteractive && opts.templatePath == "" {
		tmplType, err := legacytemplates.PromptForTemplateType(ctx)
		if err != nil {
			return err
		}
		selectedTemplateType = tmplType
	}

	// Check if a framework type was selected (any value other than "appkit")
	if selectedTemplateType != templateTypeAppKit {
		templates, err := legacytemplates.LoadLegacyTemplates()
		if err != nil {
			return err
		}

		// Use the selected template type as the framework type filter
		selectedTemplate, err := legacytemplates.PromptForLegacyTemplate(ctx, templates, selectedTemplateType)
		if err != nil {
			return err
		}

		absOutputDir, shouldDeploy, runMode, err := legacytemplates.HandleLegacyTemplateInit(ctx, selectedTemplate, opts.name, opts.nameProvided, opts.outputDir, opts.warehouseID, opts.servingEndpoint, opts.experimentID, opts.instanceName, opts.databaseName, opts.ucVolume, opts.deploy, opts.deployChanged, opts.run, opts.runChanged, isInteractive, workspaceHost, profile)
		if err != nil {
			return err
		}
		// Extract project name from the absolute output directory
		projectName := filepath.Base(absOutputDir)
		return runPostCreationSteps(ctx, absOutputDir, projectName, 0, shouldDeploy, runMode)
	}

	// Use features from flags if provided
	if len(opts.features) > 0 {
		selectedFeatures = opts.features
	}

	// Resolve template path (supports local paths and GitHub URLs)
	templateSrc := opts.templatePath
	if templateSrc == "" {
		templateSrc = os.Getenv(templatePathEnvVar)
	}
	if templateSrc == "" {
		// Use default template from GitHub
		templateSrc = defaultTemplateURL
	}

	// Step 1: Get project name first (needed before we can check destination)
	// Determine output directory for validation
	destDir := opts.name
	if opts.outputDir != "" {
		destDir = filepath.Join(opts.outputDir, opts.name)
	}

	if opts.name == "" {
		if !isInteractive {
			return errors.New("--name is required in non-interactive mode")
		}
		// Prompt includes validation for name format AND directory existence
		name, err := prompt.PromptForProjectName(ctx, opts.outputDir)
		if err != nil {
			return err
		}
		opts.name = name
		// Update destDir with the actual name
		destDir = opts.name
		if opts.outputDir != "" {
			destDir = filepath.Join(opts.outputDir, opts.name)
		}
	} else {
		// Non-interactive mode: validate name and directory existence
		if err := prompt.ValidateProjectName(opts.name); err != nil {
			return err
		}
		if _, err := os.Stat(destDir); err == nil {
			return fmt.Errorf("directory %s already exists", destDir)
		}
	}

	// Step 2: Resolve template (handles GitHub URLs by cloning)
	resolvedPath, cleanup, err := resolveTemplate(ctx, templateSrc, opts.branch)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Check for generic subdirectory first (default for multi-template repos)
	templateDir := filepath.Join(resolvedPath, "generic")
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		// Fall back to the provided path directly
		templateDir = resolvedPath
		if _, err := os.Stat(templateDir); os.IsNotExist(err) {
			return fmt.Errorf("template not found at %s (also checked %s/generic)", resolvedPath, resolvedPath)
		}
	}

	// Step 3: Determine template type and gather configuration
	usesFeatureFragments := features.HasFeaturesDirectory(templateDir)

	// When --name is provided, user is in "flags mode" - use defaults instead of prompting
	flagsMode := opts.nameProvided

	if usesFeatureFragments {
		// Feature-fragment template: prompt for features and their dependencies
		// Skip deploy/run prompts if in flags mode or if deploy/run flags were explicitly set
		skipDeployRunPrompt := flagsMode || opts.deployChanged || opts.runChanged

		if isInteractive && !opts.featuresChanged && !flagsMode {
			// Interactive mode without --features flag: prompt for features, dependencies, description
			config, err := promptForFeaturesAndDeps(ctx, selectedFeatures, skipDeployRunPrompt)
			if err != nil {
				return err
			}
			selectedFeatures = config.Features
			dependencies = config.Dependencies
			if config.Description != "" {
				opts.description = config.Description
			}
			// Use prompted values for deploy/run (only set if we prompted)
			if !skipDeployRunPrompt {
				shouldDeploy = config.Deploy
				runMode = config.RunMode
			}

			// Get warehouse from dependencies if provided
			if wh, ok := dependencies["sql_warehouse_id"]; ok && wh != "" {
				opts.warehouseID = wh
			}
		} else if isInteractive && opts.featuresChanged && !flagsMode {
			// Interactive mode with --features flag: validate features, prompt for deploy/run if no flags
			flagValues := map[string]string{
				"warehouse-id": opts.warehouseID,
			}
			if len(selectedFeatures) > 0 {
				if err := features.ValidateFeatureDependencies(selectedFeatures, flagValues); err != nil {
					return err
				}
			}
			dependencies = make(map[string]string)
			if opts.warehouseID != "" {
				dependencies["sql_warehouse_id"] = opts.warehouseID
			}

			// Prompt for deploy/run if no flags were set
			if !skipDeployRunPrompt {
				var err error
				shouldDeploy, runMode, err = prompt.PromptForDeployAndRun(ctx)
				if err != nil {
					return err
				}
			}
		} else {
			// Flags mode or non-interactive: validate features and use flag values
			flagValues := map[string]string{
				"warehouse-id": opts.warehouseID,
			}
			if len(selectedFeatures) > 0 {
				if err := features.ValidateFeatureDependencies(selectedFeatures, flagValues); err != nil {
					return err
				}
			}
			dependencies = make(map[string]string)
			if opts.warehouseID != "" {
				dependencies["sql_warehouse_id"] = opts.warehouseID
			}
		}

		// Apply flag values for deploy/run when in flags mode, flags were explicitly set, or non-interactive
		if skipDeployRunPrompt || !isInteractive {
			var err error
			shouldDeploy, runMode, err = internal.ParseDeployAndRunFlags(opts.deploy, opts.run)
			if err != nil {
				return err
			}
		}

		// Validate feature IDs
		if err := features.ValidateFeatureIDs(selectedFeatures); err != nil {
			return err
		}
	} else {
		// Pre-assembled template: detect plugins and prompt for their dependencies
		detectedPlugins, err := features.DetectPluginsFromServer(templateDir)
		if err != nil {
			return fmt.Errorf("failed to detect plugins: %w", err)
		}

		log.Debugf(ctx, "Detected plugins: %v", detectedPlugins)

		// Map detected plugins to feature IDs for ApplyFeatures
		selectedFeatures = features.MapPluginsToFeatures(detectedPlugins)
		log.Debugf(ctx, "Mapped to features: %v", selectedFeatures)

		pluginDeps := features.GetPluginDependencies(detectedPlugins)

		log.Debugf(ctx, "Plugin dependencies: %d", len(pluginDeps))

		if isInteractive && len(pluginDeps) > 0 {
			// Prompt for plugin dependencies
			dependencies, err = prompt.PromptForPluginDependencies(ctx, pluginDeps)
			if err != nil {
				return err
			}
			if wh, ok := dependencies["sql_warehouse_id"]; ok && wh != "" {
				opts.warehouseID = wh
			}
		} else {
			// Non-interactive: check flags
			dependencies = make(map[string]string)
			if opts.warehouseID != "" {
				dependencies["sql_warehouse_id"] = opts.warehouseID
			}

			// Validate required dependencies are provided
			for _, dep := range pluginDeps {
				if dep.Required {
					if _, ok := dependencies[dep.ID]; !ok {
						return fmt.Errorf("missing required flag --%s for detected plugin", dep.FlagName)
					}
				}
			}
		}

		// Set default description if not provided
		if opts.description == "" {
			opts.description = prompt.DefaultAppDescription
		}

		// Determine deploy and run options
		deployRun := &deployRunConfig{
			deploy:        opts.deploy,
			deployChanged: opts.deployChanged || flagsMode,
			run:           opts.run,
			runChanged:    opts.runChanged || flagsMode,
			isInteractive: isInteractive,
		}
		shouldDeploy, runMode, err = deployRun.resolve(ctx)
		if err != nil {
			return err
		}
	}

	// Track whether we started creating the project for cleanup on failure
	var projectCreated bool
	var runErr error
	defer func() {
		if runErr != nil && projectCreated {
			// Clean up partially created project on failure
			os.RemoveAll(destDir)
		}
	}()

	// Set description default
	if opts.description == "" {
		opts.description = prompt.DefaultAppDescription
	}

	// Build plugin imports and usages from selected features
	pluginImport, pluginUsage := features.BuildPluginStrings(selectedFeatures)

	// Template variables (initial, without feature fragments)
	vars := templateVars{
		ProjectName:    opts.name,
		AppName:        opts.name,
		SQLWarehouseID: opts.warehouseID,
		AppDescription: opts.description,
		Profile:        profile,
		WorkspaceHost:  workspaceHost,
		PluginImport:   pluginImport,
		PluginUsage:    pluginUsage,
	}

	// Load feature resource fragments
	fragments, err := loadFeatureFragments(templateDir, selectedFeatures, vars)
	if err != nil {
		return fmt.Errorf("load feature fragments: %w", err)
	}
	vars.BundleVariables = fragments.BundleVariables
	vars.BundleResources = fragments.BundleResources
	vars.TargetVariables = fragments.TargetVariables
	vars.AppEnv = fragments.AppEnv
	vars.DotEnv = fragments.DotEnv
	vars.DotEnvExample = fragments.DotEnvExample

	// Copy template with variable substitution
	var fileCount int
	runErr = prompt.RunWithSpinnerCtx(ctx, "Creating project...", func() error {
		var copyErr error
		fileCount, copyErr = copyTemplate(ctx, templateDir, destDir, vars)
		return copyErr
	})
	if runErr != nil {
		return runErr
	}
	projectCreated = true // From here on, cleanup on failure

	// Get absolute path
	absOutputDir, err := filepath.Abs(destDir)
	if err != nil {
		absOutputDir = destDir
	}

	// Apply features (adds selected features, removes unselected feature files)
	runErr = prompt.RunWithSpinnerCtx(ctx, "Configuring features...", func() error {
		return features.ApplyFeatures(absOutputDir, selectedFeatures)
	})
	if runErr != nil {
		return runErr
	}

	return runPostCreationSteps(ctx, absOutputDir, opts.name, fileCount, shouldDeploy, runMode)
}

// runPostCreationSteps handles post-creation initialization, validation, and optional deploy/run actions.
func runPostCreationSteps(ctx context.Context, absOutputDir, projectName string, fileCount int, shouldDeploy bool, runMode prompt.RunMode) error {
	// Initialize project based on type (Node.js, Python, etc.)
	var nextStepsCmd string
	projectInitializer := initializer.GetProjectInitializer(absOutputDir)
	if projectInitializer != nil {
		result := projectInitializer.Initialize(ctx, absOutputDir)
		if !result.Success {
			if result.Error != nil {
				return fmt.Errorf("%s: %w", result.Message, result.Error)
			}
			return errors.New(result.Message)
		}
		nextStepsCmd = projectInitializer.NextSteps()
	}

	// Validate dev-remote is only supported for appkit projects
	if runMode == prompt.RunModeDevRemote {
		if projectInitializer == nil || !projectInitializer.SupportsDevRemote() {
			return errors.New("--run=dev-remote is only supported for Node.js projects with @databricks/appkit")
		}
	}

	// Show next steps only if user didn't choose to deploy or run
	showNextSteps := !shouldDeploy && runMode == prompt.RunModeNone
	if showNextSteps {
		prompt.PrintSuccess(ctx, projectName, absOutputDir, fileCount, nextStepsCmd)
	} else {
		prompt.PrintSuccess(ctx, projectName, absOutputDir, fileCount, "")
	}

	// Execute post-creation actions (deploy and/or run)
	if shouldDeploy || runMode != prompt.RunModeNone {
		// Change to project directory for subsequent commands
		if err := os.Chdir(absOutputDir); err != nil {
			return fmt.Errorf("failed to change to project directory: %w", err)
		}
	}

	if shouldDeploy {
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "Deploying app...")
		if err := runPostCreateDeploy(ctx); err != nil {
			cmdio.LogString(ctx, fmt.Sprintf("⚠ Deploy failed: %v", err))
			cmdio.LogString(ctx, "  You can deploy manually with: databricks apps deploy")
		}
	}

	if runMode != prompt.RunModeNone {
		cmdio.LogString(ctx, "")
		if err := runPostCreateDev(ctx, runMode, projectInitializer, absOutputDir); err != nil {
			return err
		}
	}

	return nil
}

// runPostCreateDeploy runs the deploy command in the current directory.
func runPostCreateDeploy(ctx context.Context) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	cmd := exec.CommandContext(ctx, executable, "apps", "deploy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// runPostCreateDev runs the dev or dev-remote command in the current directory.
func runPostCreateDev(ctx context.Context, mode prompt.RunMode, projectInit initializer.Initializer, workDir string) error {
	switch mode {
	case prompt.RunModeDev:
		if projectInit != nil {
			return projectInit.RunDev(ctx, workDir)
		}
		// Fallback for unknown project types
		cmdio.LogString(ctx, "⚠ Unknown project type, cannot start development server automatically")
		return nil
	case prompt.RunModeDevRemote:
		cmdio.LogString(ctx, "Starting remote development server...")
		executable, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		cmd := exec.CommandContext(ctx, executable, "apps", "dev-remote")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	default:
		return nil
	}
}

// renameFiles maps source file names to destination names (for files that can't use special chars).
var renameFiles = map[string]string{
	"_gitignore":  ".gitignore",
	"_env":        ".env",
	"_env.local":  ".env.local",
	"_npmrc":      ".npmrc",
	"_prettierrc": ".prettierrc",
	"_eslintrc":   ".eslintrc",
}

// copyTemplate copies the template directory to dest, substituting variables.
func copyTemplate(ctx context.Context, src, dest string, vars templateVars) (int, error) {
	fileCount := 0

	// Find the project_name placeholder directory
	srcProjectDir := ""
	entries, err := os.ReadDir(src)
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		if e.IsDir() && strings.Contains(e.Name(), "{{.project_name}}") {
			srcProjectDir = filepath.Join(src, e.Name())
			break
		}
	}

	// If no {{.project_name}} dir found, copy src directly
	if srcProjectDir == "" {
		srcProjectDir = src
	}

	log.Debugf(ctx, "Copying template from: %s", srcProjectDir)

	// Files and directories to skip
	skipFiles := map[string]bool{
		"CLAUDE.md":                       true,
		"AGENTS.md":                       true,
		"databricks_template_schema.json": true,
	}
	skipDirs := map[string]bool{
		"docs":     true,
		"features": true, // Feature fragments are processed separately, not copied
	}

	err = filepath.Walk(srcProjectDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		baseName := filepath.Base(srcPath)

		// Skip certain files
		if skipFiles[baseName] {
			log.Debugf(ctx, "Skipping file: %s", baseName)
			return nil
		}

		// Skip certain directories
		if info.IsDir() && skipDirs[baseName] {
			log.Debugf(ctx, "Skipping directory: %s", baseName)
			return filepath.SkipDir
		}

		// Calculate relative path from source project dir
		relPath, err := filepath.Rel(srcProjectDir, srcPath)
		if err != nil {
			return err
		}

		// Substitute variables in path
		relPath = substituteVars(relPath, vars)

		// Handle .tmpl extension - strip it
		relPath = strings.TrimSuffix(relPath, ".tmpl")

		// Apply file renames (e.g., _gitignore -> .gitignore)
		fileName := filepath.Base(relPath)
		if newName, ok := renameFiles[fileName]; ok {
			relPath = filepath.Join(filepath.Dir(relPath), newName)
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			log.Debugf(ctx, "Creating directory: %s", relPath)
			return os.MkdirAll(destPath, info.Mode())
		}

		log.Debugf(ctx, "Copying file: %s", relPath)

		// Read file content
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}

		// Handle special files
		switch filepath.Base(srcPath) {
		case "package.json":
			content, err = processPackageJSON(content, vars)
			if err != nil {
				return fmt.Errorf("process package.json: %w", err)
			}
		default:
			// Use Go template engine for .tmpl files (handles conditionals)
			if strings.HasSuffix(srcPath, ".tmpl") {
				content, err = executeTemplate(srcPath, content, vars)
				if err != nil {
					return fmt.Errorf("process template %s: %w", srcPath, err)
				}
			} else if isTextFile(srcPath) {
				// Simple substitution for other text files
				content = []byte(substituteVars(string(content), vars))
			}
		}

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		// Write file
		if err := os.WriteFile(destPath, content, info.Mode()); err != nil {
			return err
		}

		fileCount++
		return nil
	})
	if err != nil {
		log.Debugf(ctx, "Error during template copy: %v", err)
	}
	log.Debugf(ctx, "Copied %d files", fileCount)

	return fileCount, err
}

// processPackageJSON updates the package.json with project-specific values.
func processPackageJSON(content []byte, vars templateVars) ([]byte, error) {
	// Just do string substitution to preserve key order and formatting
	return []byte(substituteVars(string(content), vars)), nil
}

// substituteVars replaces template variables in a string.
func substituteVars(s string, vars templateVars) string {
	s = strings.ReplaceAll(s, "{{.project_name}}", vars.ProjectName)
	s = strings.ReplaceAll(s, "{{.sql_warehouse_id}}", vars.SQLWarehouseID)
	s = strings.ReplaceAll(s, "{{.app_description}}", vars.AppDescription)
	s = strings.ReplaceAll(s, "{{.profile}}", vars.Profile)
	s = strings.ReplaceAll(s, "{{workspace_host}}", vars.WorkspaceHost)

	// Handle plugin placeholders
	if vars.PluginImport != "" {
		s = strings.ReplaceAll(s, "{{.plugin_import}}", vars.PluginImport)
		s = strings.ReplaceAll(s, "{{.plugin_usage}}", vars.PluginUsage)
	} else {
		// No plugins selected - clean up the template
		// Remove ", {{.plugin_import}}" from import line
		s = strings.ReplaceAll(s, ", {{.plugin_import}} ", " ")
		s = strings.ReplaceAll(s, ", {{.plugin_import}}", "")
		// Remove the plugin_usage line entirely
		s = strings.ReplaceAll(s, "    {{.plugin_usage}},\n", "")
		s = strings.ReplaceAll(s, "    {{.plugin_usage}},", "")
	}

	return s
}

// executeTemplate processes a .tmpl file using Go's text/template engine.
func executeTemplate(path string, content []byte, vars templateVars) ([]byte, error) {
	tmpl, err := template.New(filepath.Base(path)).
		Funcs(template.FuncMap{
			"workspace_host": func() string { return vars.WorkspaceHost },
		}).
		Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	// Use a map to match template variable names exactly (snake_case)
	data := map[string]string{
		"project_name":     vars.ProjectName,
		"sql_warehouse_id": vars.SQLWarehouseID,
		"app_description":  vars.AppDescription,
		"profile":          vars.Profile,
		"workspace_host":   vars.WorkspaceHost,
		"plugin_import":    vars.PluginImport,
		"plugin_usage":     vars.PluginUsage,
		"bundle_variables": vars.BundleVariables,
		"bundle_resources": vars.BundleResources,
		"target_variables": vars.TargetVariables,
		"app_env":          vars.AppEnv,
		"dotenv":           vars.DotEnv,
		"dotenv_example":   vars.DotEnvExample,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// textExtensions contains file extensions that should be treated as text files.
var textExtensions = map[string]bool{
	".ts": true, ".tsx": true, ".js": true, ".jsx": true,
	".json": true, ".yaml": true, ".yml": true,
	".md": true, ".txt": true, ".html": true, ".css": true,
	".scss": true, ".less": true, ".sql": true,
	".sh": true, ".bash": true, ".zsh": true,
	".py": true, ".go": true, ".rs": true,
	".toml": true, ".ini": true, ".cfg": true,
	".env": true, ".gitignore": true, ".npmrc": true,
	".prettierrc": true, ".eslintrc": true,
}

// textBaseNames contains file names (without extension) that should be treated as text files.
var textBaseNames = map[string]bool{
	"Makefile": true, "Dockerfile": true, "LICENSE": true,
	"README": true, ".gitignore": true, ".env": true,
	".nvmrc": true, ".node-version": true,
	"_gitignore": true, "_env": true, "_npmrc": true,
}

// isTextFile checks if a file is likely a text file based on extension.
func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if textExtensions[ext] {
		return true
	}
	return textBaseNames[filepath.Base(path)]
}

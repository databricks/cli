package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

const (
	templatePathEnvVar = "DATABRICKS_APPKIT_TEMPLATE_PATH"
	defaultTemplateURL = "https://github.com/databricks/appkit/tree/main/template"
)

func newInitCmd() *cobra.Command {
	var (
		templatePath string
		branch       string
		name         string
		warehouseID  string
		description  string
		outputDir    string
		features     []string
		deploy       bool
		run          string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new AppKit application from a template",
		Long: `Initialize a new AppKit application from a template.

When run without arguments, uses the default AppKit template and an interactive prompt
guides you through the setup. When run with --name, runs in non-interactive mode
(all required flags must be provided).

Examples:
  # Interactive mode with default template (recommended)
  databricks experimental dev app init

  # Non-interactive with flags
  databricks experimental dev app init --name my-app

  # With analytics feature (requires --warehouse-id)
  databricks experimental dev app init --name my-app --features=analytics --warehouse-id=abc123

  # Create, deploy, and run with dev-remote
  databricks experimental dev app init --name my-app --deploy --run=dev-remote

  # With a custom template from a local path
  databricks experimental dev app init --template /path/to/template --name my-app

  # With a GitHub URL
  databricks experimental dev app init --template https://github.com/user/repo --name my-app

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
				templatePath: templatePath,
				branch:       branch,
				name:         name,
				warehouseID:  warehouseID,
				description:  description,
				outputDir:    outputDir,
				features:     features,
				deploy:       deploy,
				run:          run,
			})
		},
	}

	cmd.Flags().StringVar(&templatePath, "template", "", "Template path (local directory or GitHub URL)")
	cmd.Flags().StringVar(&branch, "branch", "", "Git branch or tag (for GitHub templates)")
	cmd.Flags().StringVar(&name, "name", "", "Project name (prompts if not provided)")
	cmd.Flags().StringVar(&warehouseID, "warehouse-id", "", "SQL warehouse ID")
	cmd.Flags().StringVar(&description, "description", "", "App description")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the project to")
	cmd.Flags().StringSliceVar(&features, "features", nil, "Features to enable (comma-separated). Available: "+strings.Join(GetFeatureIDs(), ", "))
	cmd.Flags().BoolVar(&deploy, "deploy", false, "Deploy the app after creation")
	cmd.Flags().StringVar(&run, "run", "", "Run the app after creation (none, dev, dev-remote)")

	return cmd
}

type createOptions struct {
	templatePath string
	branch       string
	name         string
	warehouseID  string
	description  string
	outputDir    string
	features     []string
	deploy       bool
	run          string
}

// templateVars holds the variables for template substitution.
type templateVars struct {
	ProjectName    string
	SQLWarehouseID string
	AppDescription string
	Profile        string
	WorkspaceHost  string
	PluginImport   string
	PluginUsage    string
	// Feature resource fragments (aggregated from selected features)
	BundleVariables string
	BundleResources string
	TargetVariables string
	AppEnv          string
	DotEnv          string
	DotEnvExample   string
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

// parseDeployAndRunFlags parses the deploy and run flag values into typed values.
func parseDeployAndRunFlags(deploy bool, run string) (bool, RunMode, error) {
	var runMode RunMode
	switch run {
	case "dev":
		runMode = RunModeDev
	case "dev-remote":
		runMode = RunModeDevRemote
	case "", "none":
		runMode = RunModeNone
	default:
		return false, RunModeNone, fmt.Errorf("invalid --run value: %q (must be none, dev, or dev-remote)", run)
	}
	return deploy, runMode, nil
}

// promptForFeaturesAndDeps prompts for features and their dependencies.
// Used when the template uses the feature-fragment system.
func promptForFeaturesAndDeps(ctx context.Context, preSelectedFeatures []string) (*CreateProjectConfig, error) {
	config := &CreateProjectConfig{
		Dependencies: make(map[string]string),
		Features:     preSelectedFeatures,
	}
	theme := appkitTheme()

	// Step 1: Feature selection (skip if features already provided via flag)
	if len(config.Features) == 0 && len(AvailableFeatures) > 0 {
		options := make([]huh.Option[string], 0, len(AvailableFeatures))
		for _, f := range AvailableFeatures {
			label := f.Name + " - " + f.Description
			options = append(options, huh.NewOption(label, f.ID))
		}

		err := huh.NewMultiSelect[string]().
			Title("Select features").
			Description("space to toggle, enter to confirm").
			Options(options...).
			Value(&config.Features).
			WithTheme(theme).
			Run()
		if err != nil {
			return nil, err
		}
	}

	// Step 2: Prompt for feature dependencies
	deps := CollectDependencies(config.Features)
	for _, dep := range deps {
		// Special handling for SQL warehouse - show picker instead of text input
		if dep.ID == "sql_warehouse_id" {
			warehouseID, err := PromptForWarehouse(ctx)
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
		config.Dependencies[dep.ID] = value
	}

	// Step 3: Description
	config.Description = DefaultAppDescription
	err := huh.NewInput().
		Title("Description").
		Placeholder(DefaultAppDescription).
		Value(&config.Description).
		WithTheme(theme).
		Run()
	if err != nil {
		return nil, err
	}

	if config.Description == "" {
		config.Description = DefaultAppDescription
	}

	// Step 4: Deploy and run options
	config.Deploy, config.RunMode, err = PromptForDeployAndRun()
	if err != nil {
		return nil, err
	}

	return config, nil
}

// loadFeatureFragments reads and aggregates resource fragments for selected features.
// templateDir is the path to the template directory (containing the "features" subdirectory).
func loadFeatureFragments(templateDir string, featureIDs []string, vars templateVars) (*featureFragments, error) {
	featuresDir := filepath.Join(templateDir, "features")

	resourceFiles := CollectResourceFiles(featureIDs)
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

// parseGitHubURL extracts the repository URL, subdirectory, and branch from a GitHub URL.
// Input: https://github.com/user/repo/tree/main/templates/starter
// Output: repoURL="https://github.com/user/repo", subdir="templates/starter", branch="main"
func parseGitHubURL(url string) (repoURL, subdir, branch string) {
	// Remove trailing slash
	url = strings.TrimSuffix(url, "/")

	// Check for /tree/branch/path pattern
	if idx := strings.Index(url, "/tree/"); idx != -1 {
		repoURL = url[:idx]
		rest := url[idx+6:] // Skip "/tree/"

		// Split into branch and path
		parts := strings.SplitN(rest, "/", 2)
		branch = parts[0]
		if len(parts) > 1 {
			subdir = parts[1]
		}
		return repoURL, subdir, branch
	}

	// No /tree/ pattern, just a repo URL
	return url, "", ""
}

// cloneRepo clones a git repository to a temporary directory.
func cloneRepo(ctx context.Context, repoURL, branch string) (string, error) {
	tempDir, err := os.MkdirTemp("", "appkit-template-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	args := []string{"clone", "--depth", "1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, repoURL, tempDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = nil
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		if stderr.Len() > 0 {
			return "", fmt.Errorf("git clone failed: %s: %w", strings.TrimSpace(stderr.String()), err)
		}
		return "", fmt.Errorf("git clone failed: %w", err)
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
	repoURL, subdir, urlBranch := parseGitHubURL(templatePath)
	if branch == "" {
		branch = urlBranch // Use branch from URL if not overridden by flag
	}

	// Clone to temp dir with spinner
	var tempDir string
	err = RunWithSpinnerCtx(ctx, "Cloning template...", func() error {
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

func runCreate(ctx context.Context, opts createOptions) error {
	var selectedFeatures []string
	var dependencies map[string]string
	var shouldDeploy bool
	var runMode RunMode
	isInteractive := cmdio.IsPromptSupported(ctx)

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
		name, err := PromptForProjectName(opts.outputDir)
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
		if err := ValidateProjectName(opts.name); err != nil {
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
	usesFeatureFragments := HasFeaturesDirectory(templateDir)

	if usesFeatureFragments {
		// Feature-fragment template: prompt for features and their dependencies
		if isInteractive && len(selectedFeatures) == 0 {
			// Need to prompt for features (but we already have the name)
			config, err := promptForFeaturesAndDeps(ctx, selectedFeatures)
			if err != nil {
				return err
			}
			selectedFeatures = config.Features
			dependencies = config.Dependencies
			if config.Description != "" {
				opts.description = config.Description
			}
			shouldDeploy = config.Deploy
			runMode = config.RunMode

			// Get warehouse from dependencies if provided
			if wh, ok := dependencies["sql_warehouse_id"]; ok && wh != "" {
				opts.warehouseID = wh
			}
		} else {
			// Non-interactive or features provided via flag
			flagValues := map[string]string{
				"warehouse-id": opts.warehouseID,
			}
			if len(selectedFeatures) > 0 {
				if err := ValidateFeatureDependencies(selectedFeatures, flagValues); err != nil {
					return err
				}
			}
			dependencies = make(map[string]string)
			if opts.warehouseID != "" {
				dependencies["sql_warehouse_id"] = opts.warehouseID
			}
			var err error
			shouldDeploy, runMode, err = parseDeployAndRunFlags(opts.deploy, opts.run)
			if err != nil {
				return err
			}
		}

		// Validate feature IDs
		if err := ValidateFeatureIDs(selectedFeatures); err != nil {
			return err
		}
	} else {
		// Pre-assembled template: detect plugins and prompt for their dependencies
		detectedPlugins, err := DetectPluginsFromServer(templateDir)
		if err != nil {
			return fmt.Errorf("failed to detect plugins: %w", err)
		}

		log.Debugf(ctx, "Detected plugins: %v", detectedPlugins)

		// Map detected plugins to feature IDs for ApplyFeatures
		selectedFeatures = MapPluginsToFeatures(detectedPlugins)
		log.Debugf(ctx, "Mapped to features: %v", selectedFeatures)

		pluginDeps := GetPluginDependencies(detectedPlugins)

		log.Debugf(ctx, "Plugin dependencies: %d", len(pluginDeps))

		if isInteractive && len(pluginDeps) > 0 {
			// Prompt for plugin dependencies
			dependencies, err = PromptForPluginDependencies(ctx, pluginDeps)
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

		// Prompt for description and post-creation actions
		if isInteractive {
			if opts.description == "" {
				opts.description = DefaultAppDescription
			}
			var deployVal bool
			var runVal RunMode
			deployVal, runVal, err = PromptForDeployAndRun()
			if err != nil {
				return err
			}
			shouldDeploy = deployVal
			runMode = runVal
		} else {
			var err error
			shouldDeploy, runMode, err = parseDeployAndRunFlags(opts.deploy, opts.run)
			if err != nil {
				return err
			}
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
		opts.description = DefaultAppDescription
	}

	// Get workspace host and profile from context
	workspaceHost := ""
	profile := ""
	if w := cmdctx.WorkspaceClient(ctx); w != nil && w.Config != nil {
		workspaceHost = w.Config.Host
		profile = w.Config.Profile
	}

	// Build plugin imports and usages from selected features
	pluginImport, pluginUsage := BuildPluginStrings(selectedFeatures)

	// Template variables (initial, without feature fragments)
	vars := templateVars{
		ProjectName:    opts.name,
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
	runErr = RunWithSpinnerCtx(ctx, "Creating project...", func() error {
		var copyErr error
		fileCount, copyErr = copyTemplate(templateDir, destDir, vars)
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
	runErr = RunWithSpinnerCtx(ctx, "Configuring features...", func() error {
		return ApplyFeatures(absOutputDir, selectedFeatures)
	})
	if runErr != nil {
		return runErr
	}

	// Run npm install
	runErr = runNpmInstall(ctx, absOutputDir)
	if runErr != nil {
		return runErr
	}

	// Run npm run setup
	runErr = runNpmSetup(ctx, absOutputDir)
	if runErr != nil {
		return runErr
	}

	// Show next steps only if user didn't choose to deploy or run
	showNextSteps := !shouldDeploy && runMode == RunModeNone
	PrintSuccess(opts.name, absOutputDir, fileCount, showNextSteps)

	// Execute post-creation actions (deploy and/or run)
	if shouldDeploy || runMode != RunModeNone {
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
			cmdio.LogString(ctx, "  You can deploy manually with: databricks experimental dev app deploy")
		}
	}

	if runMode != RunModeNone {
		cmdio.LogString(ctx, "")
		if err := runPostCreateDev(ctx, runMode); err != nil {
			return err
		}
	}

	return nil
}

// runPostCreateDeploy runs the deploy command in the current directory.
func runPostCreateDeploy(ctx context.Context) error {
	// Use os.Args[0] to get the path to the current executable
	executable := os.Args[0]
	cmd := exec.CommandContext(ctx, executable, "experimental", "dev", "app", "deploy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// runPostCreateDev runs the dev or dev-remote command in the current directory.
func runPostCreateDev(ctx context.Context, mode RunMode) error {
	switch mode {
	case RunModeDev:
		cmdio.LogString(ctx, "Starting development server (npm run dev)...")
		cmd := exec.CommandContext(ctx, "npm", "run", "dev")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	case RunModeDevRemote:
		cmdio.LogString(ctx, "Starting remote development server...")
		// Use os.Args[0] to get the path to the current executable
		executable := os.Args[0]
		cmd := exec.CommandContext(ctx, executable, "experimental", "dev", "app", "dev-remote")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	default:
		return nil
	}
}

// runNpmInstall runs npm install in the project directory.
func runNpmInstall(ctx context.Context, projectDir string) error {
	// Check if npm is available
	if _, err := exec.LookPath("npm"); err != nil {
		cmdio.LogString(ctx, "⚠ npm not found. Please install Node.js and run 'npm install' manually.")
		return nil
	}

	return RunWithSpinnerCtx(ctx, "Installing dependencies...", func() error {
		cmd := exec.CommandContext(ctx, "npm", "install")
		cmd.Dir = projectDir
		cmd.Stdout = nil // Suppress output
		cmd.Stderr = nil
		return cmd.Run()
	})
}

// runNpmSetup runs npx appkit-setup in the project directory.
func runNpmSetup(ctx context.Context, projectDir string) error {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		return nil
	}

	return RunWithSpinnerCtx(ctx, "Running setup...", func() error {
		cmd := exec.CommandContext(ctx, "npx", "appkit-setup", "--write")
		cmd.Dir = projectDir
		cmd.Stdout = nil // Suppress output
		cmd.Stderr = nil
		return cmd.Run()
	})
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
func copyTemplate(src, dest string, vars templateVars) (int, error) {
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

	log.Debugf(context.Background(), "Copying template from: %s", srcProjectDir)

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
			log.Debugf(context.Background(), "Skipping file: %s", baseName)
			return nil
		}

		// Skip certain directories
		if info.IsDir() && skipDirs[baseName] {
			log.Debugf(context.Background(), "Skipping directory: %s", baseName)
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
			log.Debugf(context.Background(), "Creating directory: %s", relPath)
			return os.MkdirAll(destPath, info.Mode())
		}

		log.Debugf(context.Background(), "Copying file: %s", relPath)

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
		log.Debugf(context.Background(), "Error during template copy: %v", err)
	}
	log.Debugf(context.Background(), "Copied %d files", fileCount)

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

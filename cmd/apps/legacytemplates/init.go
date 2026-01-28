package legacytemplates

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/databricks/cli/cmd/apps/internal/yamlutil"
	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/git"
	"go.yaml.in/yaml/v3"
)

//go:embed databricks-yml.tmpl
var databricksYmlTemplate string

//go:embed gitignore.tmpl
var gitignoreTemplate string

// TemplateVars holds the variables for template substitution.
type TemplateVars struct {
	ProjectName      string
	AppName          string
	AppDescription   string
	WorkspaceHost    string
	BundleVariables  string
	ResourceBindings string
	UserAPIScopes    []string
}

// RunLegacyTemplateInit initializes a project using a legacy template.
// All resource parameters are optional and will be passed to the template if provided.
// startCommand from the manifest (if provided) overrides the default command for running the app.
func RunLegacyTemplateInit(ctx context.Context, selectedTemplate *AppTemplateManifest, appName, outputDir, warehouseID, servingEndpoint, experimentID, instanceName, databaseName, ucVolume, workspaceHost, profile string, shouldDeploy bool, runMode prompt.RunMode) error {
	// Determine the destination directory
	destDir := appName
	if outputDir != "" {
		destDir = filepath.Join(outputDir, appName)
	}

	// Check if directory already exists
	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("directory %s already exists", destDir)
	}

	// Create a temporary directory for cloning the repo
	tmpDir, err := os.MkdirTemp("", "databricks-app-template-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmdio.LogString(ctx, "Cloning template repository...")

	// Clone the repository (shallow clone)
	if err := git.Clone(ctx, selectedTemplate.GitRepo, "", tmpDir); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Source path is the template directory within the cloned repo
	srcPath := filepath.Join(tmpDir, selectedTemplate.Path)
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("template path %s not found in repository: %w", selectedTemplate.Path, err)
	}

	// Copy the template directory to the destination
	cmdio.LogString(ctx, fmt.Sprintf("Copying template files to %s...", destDir))
	if err := copyDir(srcPath, destDir); err != nil {
		return fmt.Errorf("failed to copy template: %w", err)
	}

	// Remove .git directory if it exists in the destination
	gitDir := filepath.Join(destDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		if err := os.RemoveAll(gitDir); err != nil {
			return fmt.Errorf("failed to remove .git directory: %w", err)
		}
	}

	// Build bundle variables for databricks.yml
	variablesBuilder := newVariablesBuilder()
	variablesBuilder.addWarehouse(warehouseID)
	variablesBuilder.addServingEndpoint(servingEndpoint)
	variablesBuilder.addExperiment(experimentID)
	variablesBuilder.addDatabase(instanceName, databaseName)
	variablesBuilder.addUCVolume(ucVolume)

	// Build resource bindings for databricks.yml
	bindingsBuilder := newResourceBindingsBuilder()
	bindingsBuilder.addWarehouse(warehouseID)
	bindingsBuilder.addServingEndpoint(servingEndpoint)
	bindingsBuilder.addExperiment(experimentID)
	bindingsBuilder.addDatabase(instanceName, databaseName)

	// Create databricks.yml using template
	vars := TemplateVars{
		ProjectName:      appName,
		AppName:          appName,
		AppDescription:   selectedTemplate.Manifest.Description,
		WorkspaceHost:    workspaceHost,
		BundleVariables:  variablesBuilder.build(),
		ResourceBindings: bindingsBuilder.build(),
		UserAPIScopes:    selectedTemplate.Manifest.UserAPIScopes,
	}

	tmpl, err := template.New("databricks.yml").Parse(databricksYmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse databricks.yml template: %w", err)
	}

	var databricksYmlBuf bytes.Buffer
	if err := tmpl.Execute(&databricksYmlBuf, vars); err != nil {
		return fmt.Errorf("failed to execute databricks.yml template: %w", err)
	}

	databricksYmlPath := filepath.Join(destDir, "databricks.yml")
	if err := os.WriteFile(databricksYmlPath, databricksYmlBuf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write databricks.yml: %w", err)
	}

	cmdio.LogString(ctx, "✓ Created databricks.yml")

	// Write .gitignore if it doesn't exist
	if err := WriteGitignoreIfMissing(ctx, destDir, gitignoreTemplate); err != nil {
		// Log warning but don't fail - .gitignore is optional
		cmdio.LogString(ctx, fmt.Sprintf("⚠ failed to create .gitignore: %v", err))
	}

	// Generate .env file from app.yml BEFORE inlining (inlining deletes app.yml)
	if err := generateEnvFileForLegacyTemplate(ctx, destDir, workspaceHost, profile, appName, warehouseID, servingEndpoint, experimentID, instanceName, databaseName, ucVolume); err != nil {
		// Log warning but don't fail - .env is optional
		cmdio.LogString(ctx, fmt.Sprintf("⚠ failed to generate .env file: %v", err))
	}

	// Check for app.yml in the destination directory and inline it into databricks.yml
	if err := inlineAppYmlIntoBundle(ctx, destDir); err != nil {
		return fmt.Errorf("failed to inline app.yml: %w", err)
	}

	// Get absolute path
	absOutputDir, err := filepath.Abs(destDir)
	if err != nil {
		absOutputDir = destDir
	}

	cmdio.LogString(ctx, fmt.Sprintf("✓ Successfully created %s in %s", appName, absOutputDir))
	return nil
}

// HandleLegacyTemplateInit handles the common logic for initializing a legacy template.
// It gets the app name, collects resources, determines deploy/run options, and calls RunLegacyTemplateInit.
func HandleLegacyTemplateInit(ctx context.Context, legacyTemplate *AppTemplateManifest, name string, nameProvided bool, outputDir, warehouseID, servingEndpoint, experimentID, instanceName, databaseName, ucVolume string, deploy, deployChanged bool, run string, runChanged, isInteractive bool, workspaceHost, profile string) error {
	// Get app name
	appName := name
	if appName == "" {
		if !isInteractive {
			return errors.New("--name is required in non-interactive mode")
		}
		var err error
		appName, err = prompt.PromptForProjectName(ctx, outputDir, legacyTemplate.Path)
		if err != nil {
			return err
		}
	} else {
		// Validate name in non-interactive mode
		if err := prompt.ValidateProjectName(appName); err != nil {
			return err
		}
	}

	// Collect all required resources
	collector := NewLegacyResourceCollector(
		legacyTemplate,
		isInteractive,
		warehouseID,
		servingEndpoint,
		experimentID,
		instanceName,
		databaseName,
		ucVolume,
	)
	resources, err := collector.CollectAll(ctx)
	if err != nil {
		return err
	}

	// Parse deploy and run flags
	shouldDeploy, runMode, err := parseDeployAndRunFlags(deploy, run)
	if err != nil {
		return err
	}

	// Prompt for deploy/run if in interactive mode and no flags were set
	if isInteractive && !deployChanged && !runChanged {
		shouldDeploy, runMode, err = prompt.PromptForDeployAndRun(ctx)
		if err != nil {
			return err
		}
	}

	return RunLegacyTemplateInit(ctx, legacyTemplate, appName, outputDir,
		resources.WarehouseID, resources.ServingEndpoint, resources.ExperimentID,
		resources.InstanceName, resources.DatabaseName, resources.UCVolume,
		workspaceHost, profile, shouldDeploy, runMode)
}

// parseDeployAndRunFlags parses the deploy and run flag values into typed values.
func parseDeployAndRunFlags(deploy bool, run string) (bool, prompt.RunMode, error) {
	var runMode prompt.RunMode
	switch run {
	case "dev":
		runMode = prompt.RunModeDev
	case "dev-remote":
		runMode = prompt.RunModeDevRemote
	case "", "none":
		runMode = prompt.RunModeNone
	default:
		return false, prompt.RunModeNone, fmt.Errorf("invalid --run value: %q (must be none, dev, or dev-remote)", run)
	}

	// dev-remote requires --deploy because it needs a deployed app to connect to
	if runMode == prompt.RunModeDevRemote && !deploy {
		return false, prompt.RunModeNone, errors.New("--run=dev-remote requires --deploy (dev-remote needs a deployed app to connect to)")
	}

	return deploy, runMode, nil
}

// copyDir recursively copies a directory tree from src to dst.
func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := srcFile.WriteTo(dstFile); err != nil {
		return err
	}

	return nil
}

// generateEnvFileForLegacyTemplate creates a .env file from app.yml with resource configuration.
func generateEnvFileForLegacyTemplate(ctx context.Context, destDir, workspaceHost, profile, appName, warehouseID, servingEndpoint, experimentID, instanceName, databaseName, ucVolume string) error {
	// Check if app.yml or app.yaml exists
	var appYmlPath string
	for _, filename := range []string{"app.yml", "app.yaml"} {
		path := filepath.Join(destDir, filename)
		if _, err := os.Stat(path); err == nil {
			appYmlPath = path
			break
		}
	}

	if appYmlPath == "" {
		// No app.yml found, skip .env generation
		return nil
	}

	// Build resources map from collected parameters
	// Resource names should match what's in databricks.yml
	resources := make(map[string]string)

	if warehouseID != "" {
		resources[resourceNameSQLWarehouse] = warehouseID
	}
	if servingEndpoint != "" {
		resources[resourceNameServingEndpoint] = servingEndpoint
	}
	if experimentID != "" {
		resources[resourceNameExperiment] = experimentID
	}
	if instanceName != "" {
		resources[resourceNameDatabase] = instanceName
	}
	if databaseName != "" {
		resources[resourceNameDatabaseName] = databaseName
	}
	if ucVolume != "" {
		resources[resourceNameUCVolume] = ucVolume
	}

	// Create EnvFileBuilder
	builder, err := NewEnvFileBuilder(workspaceHost, profile, appName, appYmlPath, resources)
	if err != nil {
		return fmt.Errorf("failed to create env builder: %w", err)
	}

	// Write .env file
	err = builder.WriteEnvFile(destDir)
	if err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	cmdio.LogString(ctx, "✓ Generated .env file from app.yml")
	return nil
}

// inlineAppYmlIntoBundle checks for app.yml in the directory, inlines it into databricks.yml, and deletes app.yml.
func inlineAppYmlIntoBundle(ctx context.Context, dir string) error {
	// Change to the directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to change to directory: %w", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	// Read the databricks.yml file
	databricksYmlPath := "databricks.yml"
	databricksData, err := os.ReadFile(databricksYmlPath)
	if err != nil {
		return fmt.Errorf("failed to read databricks.yml: %w", err)
	}

	// Parse databricks.yml as yaml.Node to preserve field names
	var databricksNode yaml.Node
	err = yaml.Unmarshal(databricksData, &databricksNode)
	if err != nil {
		return fmt.Errorf("failed to parse databricks.yml: %w", err)
	}

	// Convert yaml.Node to dyn.Value preserving field names
	configValue, err := yamlutil.YamlNodeToDynValue(&databricksNode)
	if err != nil {
		return fmt.Errorf("failed to convert databricks config: %w", err)
	}

	// Get the app value from resources.apps.app
	appValue, err := dyn.GetByPath(configValue, dyn.MustPathFromString("resources.apps.app"))
	if err != nil {
		return fmt.Errorf("failed to get app from databricks.yml: %w", err)
	}

	// Inline the app config file (checks for app.yml or app.yaml and inlines if found)
	appConfigFile, err := yamlutil.InlineAppConfigFile(&appValue)
	if err != nil {
		return fmt.Errorf("failed to inline app config: %w", err)
	}

	// If no app config file was found, nothing to do
	if appConfigFile == "" {
		return nil
	}

	// Set the updated app value back
	configValue, err = dyn.SetByPath(configValue, dyn.MustPathFromString("resources.apps.app"), appValue)
	if err != nil {
		return fmt.Errorf("failed to set updated app value: %w", err)
	}

	// Extract the top-level map back and set explicit line numbers for ordering
	configMap, ok := configValue.AsMap()
	if !ok {
		return errors.New("config is not a map")
	}

	// Define the desired order with explicit line numbers
	keyOrder := map[string]int{
		"bundle":    1,
		"workspace": 2,
		"variables": 3,
		"resources": 4,
	}

	updatedConfig := make(map[string]dyn.Value)
	for _, pair := range configMap.Pairs() {
		key := pair.Key.MustString()
		value := pair.Value

		// Set the line number based on the desired order
		if lineNum, ok := keyOrder[key]; ok {
			value = dyn.NewValue(value.Value(), []dyn.Location{{Line: lineNum}})
		}

		updatedConfig[key] = value
	}

	// Save the updated databricks.yml (force=true since we're updating the file we just created)
	saver := yamlsaver.NewSaver()
	err = saver.SaveAsYAML(updatedConfig, databricksYmlPath, true)
	if err != nil {
		return fmt.Errorf("failed to save databricks.yml: %w", err)
	}

	// Add blank lines between top-level keys for better readability
	err = yamlutil.AddBlankLinesBetweenTopLevelKeys(databricksYmlPath)
	if err != nil {
		return fmt.Errorf("failed to format databricks.yml: %w", err)
	}

	// Delete the app config file
	if err := os.Remove(appConfigFile); err != nil {
		return fmt.Errorf("failed to remove %s: %w", appConfigFile, err)
	}
	cmdio.LogString(ctx, "✓ Inlined and removed "+appConfigFile)
	return nil
}

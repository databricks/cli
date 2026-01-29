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

	"github.com/databricks/cli/cmd/apps/internal"
	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/git"
	"gopkg.in/yaml.v3"
)

//go:embed databricks-yml.tmpl
var databricksYmlTemplate string

//go:embed gitignore.tmpl
var gitignoreTemplate string

// TemplateVars holds the variables for template substitution.
type TemplateVars struct {
	ProjectName    string
	AppName        string
	AppDescription string
	WorkspaceHost  string
	Resources      *ResourceValues
	UserAPIScopes  []string
	AppConfig      *AppConfig
}

// AppConfig represents the parsed content of app.yml from a legacy template.
type AppConfig struct {
	Command       []string
	Env           []EnvVar
	ResourcesYAML string // Pre-formatted YAML string with proper indentation
}

// ParseAppYmlForTemplate reads app.yml from the source template directory,
// converts camelCase keys to snake_case, and returns structured data for template rendering.
// Returns nil if app.yml doesn't exist (not an error - some templates don't have it).
func ParseAppYmlForTemplate(templateSrcPath string) (*AppConfig, error) {
	// Try both app.yml and app.yaml
	var appYmlPath string
	for _, name := range []string{"app.yml", "app.yaml"} {
		path := filepath.Join(templateSrcPath, name)
		if _, err := os.Stat(path); err == nil {
			appYmlPath = path
			break
		}
	}

	if appYmlPath == "" {
		return nil, nil // No app.yml, not an error
	}

	// Read app.yml
	data, err := os.ReadFile(appYmlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read app.yml: %w", err)
	}

	// Parse into yaml.Node
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("failed to parse app.yml: %w", err)
	}

	// Convert camelCase to snake_case
	ConvertKeysToSnakeCase(&node)

	// Marshal back to get snake_case YAML
	snakeCaseData, err := yaml.Marshal(&node)
	if err != nil {
		return nil, fmt.Errorf("failed to convert app.yml to snake_case: %w", err)
	}

	// Parse into structured format
	var parsed struct {
		Command   []string       `yaml:"command"`
		Env       []EnvVar       `yaml:"env"`
		Resources map[string]any `yaml:"resources"`
	}

	if err := yaml.Unmarshal(snakeCaseData, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse app.yml structure: %w", err)
	}

	config := &AppConfig{
		Command: parsed.Command,
		Env:     parsed.Env,
	}

	// If resources exist, format them as indented YAML for template inclusion
	if len(parsed.Resources) > 0 {
		resourcesData, err := yaml.Marshal(parsed.Resources)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resources: %w", err)
		}

		// Add proper indentation (10 spaces for "resources:" under "config:" under "app:")
		lines := bytes.Split(bytes.TrimSpace(resourcesData), []byte("\n"))
		indentedLines := make([][]byte, len(lines))
		for i, line := range lines {
			if len(line) > 0 {
				indentedLines[i] = append([]byte("          "), line...)
			}
		}
		config.ResourcesYAML = string(bytes.Join(indentedLines, []byte("\n")))
	}

	return config, nil
}

// RunLegacyTemplateInit initializes a project using a legacy template.
// All resource parameters are optional and will be passed to the template if provided.
// Returns the absolute output directory, start command from manifest, and error.
func RunLegacyTemplateInit(ctx context.Context, selectedTemplate *AppTemplateManifest, appName, outputDir, warehouseID, servingEndpoint, experimentID, instanceName, databaseName, ucVolume, workspaceHost, profile string, shouldDeploy bool, runMode prompt.RunMode) (string, string, error) {
	// Determine the destination directory
	destDir := appName
	if outputDir != "" {
		destDir = filepath.Join(outputDir, appName)
	}

	// Check if directory already exists
	if _, err := os.Stat(destDir); err == nil {
		return "", "", fmt.Errorf("directory %s already exists", destDir)
	}

	// Create a temporary directory for cloning the repo
	tmpDir, err := os.MkdirTemp("", "databricks-app-template-*")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmdio.LogString(ctx, "Cloning template repository...")

	// Clone the repository (shallow clone)
	if err := git.Clone(ctx, selectedTemplate.GitRepo, "", tmpDir); err != nil {
		return "", "", fmt.Errorf("failed to clone repository: %w", err)
	}

	// Source path is the template directory within the cloned repo
	srcPath := filepath.Join(tmpDir, selectedTemplate.Path)
	if _, err := os.Stat(srcPath); err != nil {
		return "", "", fmt.Errorf("template path %s not found in repository: %w", selectedTemplate.Path, err)
	}

	// Parse app.yml from the source template directory BEFORE copying
	appConfig, err := ParseAppYmlForTemplate(srcPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse app.yml: %w", err)
	}

	// Copy the template directory to the destination
	cmdio.LogString(ctx, fmt.Sprintf("Copying template files to %s...", destDir))
	if err := copyDir(srcPath, destDir); err != nil {
		return "", "", fmt.Errorf("failed to copy template: %w", err)
	}

	// Remove .git directory if it exists in the destination
	gitDir := filepath.Join(destDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		if err := os.RemoveAll(gitDir); err != nil {
			return "", "", fmt.Errorf("failed to remove .git directory: %w", err)
		}
	}

	// Build resource values from collected parameters
	resourceValues := NewResourceValues()
	resourceValues.Set(ResourceTypeSQLWarehouse, warehouseID)
	resourceValues.Set(ResourceTypeServingEndpoint, servingEndpoint)
	resourceValues.Set(ResourceTypeExperiment, experimentID)
	resourceValues.Set(ResourceTypeDatabase, instanceName, databaseName)
	resourceValues.Set(ResourceTypeUCVolume, ucVolume)

	// Create databricks.yml using template
	vars := TemplateVars{
		ProjectName:    appName,
		AppName:        appName,
		AppDescription: selectedTemplate.Manifest.Description,
		WorkspaceHost:  workspaceHost,
		Resources:      resourceValues,
		UserAPIScopes:  selectedTemplate.Manifest.UserAPIScopes,
		AppConfig:      appConfig,
	}

	// Create template with custom functions for resource handling
	tmpl, err := template.New("databricks.yml").Funcs(getTemplateFuncs(resourceValues, appConfig)).Parse(databricksYmlTemplate)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse databricks.yml template: %w", err)
	}

	var databricksYmlBuf bytes.Buffer
	if err := tmpl.Execute(&databricksYmlBuf, vars); err != nil {
		return "", "", fmt.Errorf("failed to execute databricks.yml template: %w", err)
	}

	databricksYmlPath := filepath.Join(destDir, "databricks.yml")
	if err := os.WriteFile(databricksYmlPath, databricksYmlBuf.Bytes(), 0o644); err != nil {
		return "", "", fmt.Errorf("failed to write databricks.yml: %w", err)
	}

	cmdio.LogString(ctx, "✓ Created databricks.yml")

	// Write .gitignore if it doesn't exist
	if err := WriteGitignoreIfMissing(ctx, destDir, gitignoreTemplate); err != nil {
		// Log warning but don't fail - .gitignore is optional
		cmdio.LogString(ctx, fmt.Sprintf("⚠ failed to create .gitignore: %v", err))
	}

	// Generate .env file from app.yml in destination directory
	if err := generateEnvFileForLegacyTemplate(ctx, destDir, workspaceHost, profile, appName, warehouseID, servingEndpoint, experimentID, instanceName, databaseName, ucVolume); err != nil {
		// Log warning but don't fail - .env is optional
		cmdio.LogString(ctx, fmt.Sprintf("⚠ failed to generate .env file: %v", err))
	}

	// Delete app.yml from destination if it exists (already inlined into databricks.yml)
	for _, name := range []string{"app.yml", "app.yaml"} {
		appYmlPath := filepath.Join(destDir, name)
		if _, err := os.Stat(appYmlPath); err == nil {
			if err := os.Remove(appYmlPath); err != nil {
				cmdio.LogString(ctx, fmt.Sprintf("⚠ failed to remove %s: %v", name, err))
			}
			break
		}
	}

	// Get absolute path
	absOutputDir, err := filepath.Abs(destDir)
	if err != nil {
		absOutputDir = destDir
	}

	cmdio.LogString(ctx, fmt.Sprintf("✓ Successfully created %s in %s", appName, absOutputDir))

	// Return the absolute path and start command for post-creation steps
	startCommand := selectedTemplate.Manifest.StartCommand
	return absOutputDir, startCommand, nil
}

// HandleLegacyTemplateInit handles the common logic for initializing a legacy template.
// It gets the app name, collects resources, determines deploy/run options, and calls RunLegacyTemplateInit.
// Returns the absolute output directory, start command, should deploy, run mode, and error.
func HandleLegacyTemplateInit(ctx context.Context, legacyTemplate *AppTemplateManifest, name string, nameProvided bool, outputDir, warehouseID, servingEndpoint, experimentID, instanceName, databaseName, ucVolume string, deploy, deployChanged bool, run string, runChanged, isInteractive bool, workspaceHost, profile string) (string, string, bool, prompt.RunMode, error) {
	// Get app name
	appName := name
	if appName == "" {
		if !isInteractive {
			return "", "", false, prompt.RunModeNone, errors.New("--name is required in non-interactive mode")
		}
		var err error
		appName, err = prompt.PromptForProjectName(ctx, outputDir, legacyTemplate.Path)
		if err != nil {
			return "", "", false, prompt.RunModeNone, err
		}
	} else {
		// Validate name in non-interactive mode
		if err := prompt.ValidateProjectName(appName); err != nil {
			return "", "", false, prompt.RunModeNone, err
		}
	}

	// Collect all required resources
	collector := NewResourceCollector(
		legacyTemplate,
		isInteractive,
		warehouseID,
		servingEndpoint,
		experimentID,
		instanceName,
		databaseName,
		ucVolume,
	)
	resourceValues, err := collector.CollectAll(ctx)
	if err != nil {
		return "", "", false, prompt.RunModeNone, err
	}

	// Parse deploy and run flags
	shouldDeploy, runMode, err := internal.ParseDeployAndRunFlags(deploy, run)
	if err != nil {
		return "", "", false, prompt.RunModeNone, err
	}

	// Prompt for deploy/run if in interactive mode and no flags were set
	if isInteractive && !deployChanged && !runChanged {
		shouldDeploy, runMode, err = prompt.PromptForDeployAndRun(ctx)
		if err != nil {
			return "", "", false, prompt.RunModeNone, err
		}
	}

	// Extract individual resource values for RunLegacyTemplateInit
	var warehouseIDValue, servingEndpointValue, experimentIDValue, instanceNameValue, databaseNameValue, ucVolumeValue string
	if val := resourceValues.Get(ResourceTypeSQLWarehouse); val != nil {
		warehouseIDValue = val.SingleValue()
	}
	if val := resourceValues.Get(ResourceTypeServingEndpoint); val != nil {
		servingEndpointValue = val.SingleValue()
	}
	if val := resourceValues.Get(ResourceTypeExperiment); val != nil {
		experimentIDValue = val.SingleValue()
	}
	if val := resourceValues.Get(ResourceTypeDatabase); val != nil && len(val.Values) >= 2 {
		instanceNameValue = val.Values[0]
		databaseNameValue = val.Values[1]
	}
	if val := resourceValues.Get(ResourceTypeUCVolume); val != nil {
		ucVolumeValue = val.SingleValue()
	}

	absOutputDir, startCommand, err := RunLegacyTemplateInit(ctx, legacyTemplate, appName, outputDir,
		warehouseIDValue, servingEndpointValue, experimentIDValue,
		instanceNameValue, databaseNameValue, ucVolumeValue,
		workspaceHost, profile, shouldDeploy, runMode)
	if err != nil {
		return "", "", false, prompt.RunModeNone, err
	}

	return absOutputDir, startCommand, shouldDeploy, runMode, nil
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
		resources[internal.ResourceNameSQLWarehouse] = warehouseID
	}
	if servingEndpoint != "" {
		resources[internal.ResourceNameServingEndpoint] = servingEndpoint
	}
	if experimentID != "" {
		resources[internal.ResourceNameExperiment] = experimentID
	}
	if instanceName != "" {
		resources[internal.ResourceNameDatabase] = instanceName
	}
	if databaseName != "" {
		resources[internal.ResourceNameDatabaseName] = databaseName
	}
	if ucVolume != "" {
		resources[internal.ResourceNameUCVolume] = ucVolume
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

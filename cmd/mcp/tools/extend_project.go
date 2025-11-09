package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/mcp/auth"
)

// ExtendProjectArgs represents the arguments for the extend_project tool.
type ExtendProjectArgs struct {
	ProjectPath string `json:"project_path"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Template    string `json:"template,omitempty"`
}

// ExtendProject extends a Databricks project with a new resource.
func ExtendProject(ctx context.Context, args ExtendProjectArgs) (string, error) {
	// Validate project path and ensure it's a Databricks project
	if err := ValidateDatabricksProject(args.ProjectPath); err != nil {
		return "", err
	}

	// Check authentication
	if err := auth.CheckAuthentication(ctx); err != nil {
		return "", err
	}

	// Validate type
	validTypes := []string{"app", "job", "pipeline", "dashboard"}
	if !contains(validTypes, args.Type) {
		return "", fmt.Errorf("invalid type: %s. Must be one of: app, job, pipeline, dashboard", args.Type)
	}

	// Validate name
	if args.Name == "" {
		return "", errors.New("name is required")
	}

	// Check if resource already exists
	resourcesDir := filepath.Join(args.ProjectPath, "resources")
	if _, err := os.Stat(resourcesDir); os.IsNotExist(err) {
		if err := os.MkdirAll(resourcesDir, 0o755); err != nil {
			return "", fmt.Errorf("failed to create resources directory: %w", err)
		}
	}

	switch args.Type {
	case "app":
		return extendWithApp(ctx, args)
	case "job":
		return extendWithJob(ctx, args)
	case "pipeline":
		return extendWithPipeline(ctx, args)
	case "dashboard":
		return extendWithDashboard(ctx, args)
	default:
		return "", fmt.Errorf("unsupported type: %s", args.Type)
	}
}

func extendWithApp(ctx context.Context, args ExtendProjectArgs) (string, error) {
	// FIXME: This should rely on the databricks bundle generate command

	// Clone app-templates repo to temp directory to get list of templates
	tmpDir, err := os.MkdirTemp("", "app-templates-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if args.Template == "" {
		// Clone to get list of available templates
		cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "https://github.com/databricks/app-templates", tmpDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to clone app-templates: %w\nOutput: %s", err, string(output))
		}

		// List available templates
		entries, err := os.ReadDir(tmpDir)
		if err != nil {
			return "", fmt.Errorf("failed to read app-templates directory: %w", err)
		}

		var templates []string
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				templates = append(templates, entry.Name())
			}
		}

		templateList := formatTemplateList(templates)
		return "", fmt.Errorf("template parameter was not specified for the app\n\nYou have two options:\n\n1. Ask the user which template they want to use from this list:\n%s\n2. If the user described what they want the app to do, call extend_project again with template='nodejs-fastapi-hello-world-app' as a sensible default\n\nExample: extend_project(project_path='%s', type='app', name='%s', template='nodejs-fastapi-hello-world-app')",
			templateList, args.ProjectPath, args.Name)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "https://github.com/databricks/app-templates", tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to clone app-templates: %w\nOutput: %s", err, string(output))
	}

	// Check if template exists
	templateSrc := filepath.Join(tmpDir, args.Template)
	if _, err := os.Stat(templateSrc); os.IsNotExist(err) {
		return "", fmt.Errorf("template '%s' not found in app-templates repository", args.Template)
	}

	// Copy template to project directory
	appDest := filepath.Join(args.ProjectPath, args.Name)
	if err := copyDir(templateSrc, appDest); err != nil {
		return "", fmt.Errorf("failed to copy app template: %w", err)
	}

	// Replace template name references in all copied files
	replacements := map[string]string{
		args.Template: args.Name,
	}
	if err := replaceInDirectory(appDest, replacements); err != nil {
		return "", fmt.Errorf("failed to replace template references: %w", err)
	}

	// Create resources YAML file
	resourceYAML := fmt.Sprintf(`resources:
  apps:
    %s:
      name: %s
      description: Databricks app created from %s template
      source_code_path: ../%s
`, args.Name, args.Name, args.Template, args.Name)

	resourceFile := filepath.Join(args.ProjectPath, "resources", args.Name+".yml")
	if err := os.WriteFile(resourceFile, []byte(resourceYAML), 0o644); err != nil {
		return "", fmt.Errorf("failed to create resource file: %w", err)
	}

	return buildSuccessResponse(args.Type, args.Name, args.ProjectPath), nil
}

func extendWithJob(ctx context.Context, args ExtendProjectArgs) (string, error) {
	// FIXME: This should rely on the databricks bundle generate command

	if args.Template == "" {
		return "", errors.New("template is required for jobs\n\nPlease specify the language: 'python' or 'sql'")
	}

	if args.Template != "python" && args.Template != "sql" {
		return "", fmt.Errorf("invalid template for job: %s. Must be 'python' or 'sql'", args.Template)
	}

	// Clone bundle-examples repo to temp directory
	tmpDir, err := os.MkdirTemp("", "bundle-examples-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "https://github.com/databricks/bundle-examples", tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to clone bundle-examples: %w\nOutput: %s", err, string(output))
	}

	templateName := "default_" + args.Template
	templateSrc := filepath.Join(tmpDir, templateName)

	// Copy job resource YAML
	resourceSrc := filepath.Join(templateSrc, "resources")
	resourceFiles, err := os.ReadDir(resourceSrc)
	if err != nil {
		return "", fmt.Errorf("failed to read template resources: %w", err)
	}

	for _, file := range resourceFiles {
		if strings.HasSuffix(file.Name(), ".job.yml") {
			srcFile := filepath.Join(resourceSrc, file.Name())
			dstFile := filepath.Join(args.ProjectPath, "resources", args.Name+".job.yml")

			content, err := os.ReadFile(srcFile)
			if err != nil {
				return "", fmt.Errorf("failed to read job template: %w", err)
			}

			// Replace template name with actual name
			newContent := strings.ReplaceAll(string(content), templateName, args.Name)

			if err := os.WriteFile(dstFile, []byte(newContent), 0o644); err != nil {
				return "", fmt.Errorf("failed to write job file: %w", err)
			}
			break
		}
	}

	// Copy source files
	srcDir := filepath.Join(args.ProjectPath, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create src directory: %w", err)
	}

	if args.Template == "python" {
		// Copy python source
		pythonSrc := filepath.Join(templateSrc, "src", templateName)
		pythonDest := filepath.Join(srcDir, args.Name)
		if err := copyDir(pythonSrc, pythonDest); err != nil {
			return "", fmt.Errorf("failed to copy python source: %w", err)
		}

		// Replace template references in source files
		replacements := map[string]string{
			templateName: args.Name,
		}
		if err := replaceInDirectory(pythonDest, replacements); err != nil {
			return "", fmt.Errorf("failed to replace template references in source: %w", err)
		}

		// Copy tests if they don't exist
		testsDest := filepath.Join(args.ProjectPath, "tests")
		if _, err := os.Stat(testsDest); os.IsNotExist(err) {
			testsSrc := filepath.Join(templateSrc, "tests")
			if err := copyDir(testsSrc, testsDest); err != nil {
				// Tests are optional, just warn
				fmt.Fprintf(os.Stderr, "Warning: failed to copy tests: %v\n", err)
			} else {
				// Replace template references in tests
				if err := replaceInDirectory(testsDest, replacements); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to replace template references in tests: %v\n", err)
				}
			}
		}
	} else {
		// Copy SQL files
		sqlFiles, err := filepath.Glob(filepath.Join(templateSrc, "src", "*.sql"))
		if err != nil {
			return "", fmt.Errorf("failed to find SQL files: %w", err)
		}

		replacements := map[string]string{
			templateName: args.Name,
		}

		for _, sqlFile := range sqlFiles {
			basename := filepath.Base(sqlFile)
			newName := strings.ReplaceAll(basename, templateName, args.Name)
			dstFile := filepath.Join(srcDir, newName)

			if _, err := os.Stat(dstFile); err == nil {
				continue // Don't overwrite existing files
			}

			content, err := os.ReadFile(sqlFile)
			if err != nil {
				return "", fmt.Errorf("failed to read SQL file: %w", err)
			}

			// Replace template references in SQL content
			modified := string(content)
			for old, new := range replacements {
				modified = strings.ReplaceAll(modified, old, new)
			}

			if err := os.WriteFile(dstFile, []byte(modified), 0o644); err != nil {
				return "", fmt.Errorf("failed to write SQL file: %w", err)
			}
		}
	}

	return buildSuccessResponse(args.Type, args.Name, args.ProjectPath), nil
}

func extendWithPipeline(ctx context.Context, args ExtendProjectArgs) (string, error) {
	// FIXME: This should rely on the databricks bundle generate command

	if args.Template == "" {
		return "", errors.New("template is required for pipelines\n\nPlease specify the language: 'python' or 'sql'")
	}

	if args.Template != "python" && args.Template != "sql" {
		return "", fmt.Errorf("invalid template for pipeline: %s. Must be 'python' or 'sql'", args.Template)
	}

	// Clone bundle-examples repo to temp directory
	tmpDir, err := os.MkdirTemp("", "bundle-examples-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "https://github.com/databricks/bundle-examples", tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to clone bundle-examples: %w\nOutput: %s", err, string(output))
	}

	templateName := "lakeflow_pipelines_" + args.Template
	templateSrc := filepath.Join(tmpDir, templateName)

	// Copy source files first - use the actual directory name from bundle-examples
	srcDir := filepath.Join(templateSrc, "src")
	srcEntries, err := os.ReadDir(srcDir)
	if err != nil {
		return "", fmt.Errorf("failed to read pipeline src directory: %w", err)
	}

	// Find the first directory in src/
	var srcSubdir string
	for _, entry := range srcEntries {
		if entry.IsDir() {
			srcSubdir = entry.Name()
			break
		}
	}
	if srcSubdir == "" {
		return "", fmt.Errorf("no source directory found in %s", srcDir)
	}

	srcPattern := filepath.Join(srcDir, srcSubdir)
	srcDest := filepath.Join(args.ProjectPath, "src", args.Name)
	if err := copyDir(srcPattern, srcDest); err != nil {
		return "", fmt.Errorf("failed to copy pipeline source: %w", err)
	}

	// Copy pipeline resource YAML
	resourceSrc := filepath.Join(templateSrc, "resources")
	resourceFiles, err := os.ReadDir(resourceSrc)
	if err != nil {
		return "", fmt.Errorf("failed to read template resources: %w", err)
	}

	// Prepare replacements map
	replacements := map[string]string{
		templateName: args.Name,
		srcSubdir:    args.Name,
	}

	for _, file := range resourceFiles {
		if strings.HasSuffix(file.Name(), ".pipeline.yml") {
			srcFile := filepath.Join(resourceSrc, file.Name())
			dstFile := filepath.Join(args.ProjectPath, "resources", args.Name+".pipeline.yml")

			content, err := os.ReadFile(srcFile)
			if err != nil {
				return "", fmt.Errorf("failed to read pipeline template: %w", err)
			}

			// Apply all replacements
			newContent := string(content)
			for old, new := range replacements {
				newContent = strings.ReplaceAll(newContent, old, new)
			}

			if err := os.WriteFile(dstFile, []byte(newContent), 0o644); err != nil {
				return "", fmt.Errorf("failed to write pipeline file: %w", err)
			}
			break
		}
	}

	// Replace template references in all pipeline source files
	if err := replaceInDirectory(srcDest, replacements); err != nil {
		return "", fmt.Errorf("failed to replace template references: %w", err)
	}

	return buildSuccessResponse(args.Type, args.Name, args.ProjectPath), nil
}

func extendWithDashboard(ctx context.Context, args ExtendProjectArgs) (string, error) {
	// FIXME: This should rely on the databricks bundle generate command

	// Clone bundle-examples repo to temp directory
	tmpDir, err := os.MkdirTemp("", "bundle-examples-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "https://github.com/databricks/bundle-examples", tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to clone bundle-examples: %w\nOutput: %s", err, string(output))
	}

	templateSrc := filepath.Join(tmpDir, "knowledge_base", "dashboard_nyc_taxi")

	// Copy dashboard resource YAML
	resourceSrc := filepath.Join(templateSrc, "resources")
	resourceFiles, err := os.ReadDir(resourceSrc)
	if err != nil {
		return "", fmt.Errorf("failed to read template resources: %w", err)
	}

	for _, file := range resourceFiles {
		if strings.HasSuffix(file.Name(), ".dashboard.yml") {
			srcFile := filepath.Join(resourceSrc, file.Name())
			dstFile := filepath.Join(args.ProjectPath, "resources", args.Name+".dashboard.yml")

			content, err := os.ReadFile(srcFile)
			if err != nil {
				return "", fmt.Errorf("failed to read dashboard template: %w", err)
			}

			// Replace template name with actual name
			oldName := strings.TrimSuffix(file.Name(), ".dashboard.yml")
			newContent := strings.ReplaceAll(string(content), oldName, args.Name)

			if err := os.WriteFile(dstFile, []byte(newContent), 0o644); err != nil {
				return "", fmt.Errorf("failed to write dashboard file: %w", err)
			}
			break
		}
	}

	// Copy dashboard JSON files
	srcDir := filepath.Join(args.ProjectPath, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create src directory: %w", err)
	}

	dashFiles, err := filepath.Glob(filepath.Join(templateSrc, "src", "*.lvdash.json"))
	if err != nil {
		return "", fmt.Errorf("failed to find dashboard files: %w", err)
	}

	for _, dashFile := range dashFiles {
		basename := filepath.Base(dashFile)
		// Extract the old name from the filename (e.g., "nyc_taxi.lvdash.json" -> "nyc_taxi")
		oldName := strings.TrimSuffix(basename, ".lvdash.json")
		newName := args.Name + ".lvdash.json"
		dstFile := filepath.Join(srcDir, newName)

		content, err := os.ReadFile(dashFile)
		if err != nil {
			return "", fmt.Errorf("failed to read dashboard JSON: %w", err)
		}

		// Replace template references in dashboard content
		modified := strings.ReplaceAll(string(content), oldName, args.Name)

		if err := os.WriteFile(dstFile, []byte(modified), 0o644); err != nil {
			return "", fmt.Errorf("failed to write dashboard JSON: %w", err)
		}
	}

	return buildSuccessResponse(args.Type, args.Name, args.ProjectPath), nil
}

func buildSuccessResponse(resourceType, name, projectPath string) string {
	// Determine the correct file extension for the resource YAML
	var resourceExt string
	switch resourceType {
	case "app":
		resourceExt = "yml"
	case "job":
		resourceExt = "job.yml"
	case "pipeline":
		resourceExt = "pipeline.yml"
	case "dashboard":
		resourceExt = "dashboard.yml"
	default:
		resourceExt = "yml"
	}

	// Build base response
	response := fmt.Sprintf(`Successfully added %s '%s' to the project!

Files created:
- resources/%s.%s
- src/%s/

IMPORTANT: This is just a starting point! You need to iterate over the generated files to complete the setup.

Next steps:

1. Use the analyze_project tool to learn about the current project structure and how to use it

2. Validate that the extensions are correct using invoke_databricks_cli tool:
   invoke_databricks_cli(command="bundle validate", working_directory="%s")
`, resourceType, name, name, resourceExt, name, projectPath)

	// Add resource-specific validation guidance
	switch resourceType {
	case "app":
		response += fmt.Sprintf(`
   For apps: Also check that any warehouse references in resources/%s.yml are valid

3. Fix any validation errors in the configuration files

4. Test the app locally before deployment:
   Use invoke_databricks_cli tool: invoke_databricks_cli(command="apps run-local --source-dir %s", working_directory="%s")

5. Deploy to your development workspace:
   Use invoke_databricks_cli tool: invoke_databricks_cli(command="bundle deploy --target dev", working_directory="%s")
`, name, name, projectPath, projectPath)
	case "pipeline":
		response += fmt.Sprintf(`
3. Fix any validation errors in the configuration files

4. Optionally, validate the pipeline definition before deployment:
   Use invoke_databricks_cli tool: invoke_databricks_cli(command="bundle run %s --validate-only", working_directory="%s")

5. Deploy to your development workspace:
   Use invoke_databricks_cli tool: invoke_databricks_cli(command="bundle deploy --target dev", working_directory="%s")
`, name, projectPath, projectPath)
	default:
		response += fmt.Sprintf(`
3. Fix any validation errors in the configuration files

4. Deploy to your development workspace:
   Use invoke_databricks_cli tool: invoke_databricks_cli(command="bundle deploy --target dev", working_directory="%s")
`, projectPath)
	}

	response += `

For more information about bundle resources, visit:
https://docs.databricks.com/dev-tools/bundles/settings`

	return response
}

// Helper functions

func formatTemplateList(templates []string) string {
	var result strings.Builder
	for _, t := range templates {
		result.WriteString("- ")
		result.WriteString(t)
		result.WriteString("\n")
	}
	return result.String()
}

// replaceInFile replaces old string with new string in a file.
func replaceInFile(filePath string, replacements map[string]string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	modified := string(content)
	for old, new := range replacements {
		modified = strings.ReplaceAll(modified, old, new)
	}

	// Only write if content changed
	if modified != string(content) {
		return os.WriteFile(filePath, []byte(modified), 0o644)
	}
	return nil
}

// replaceInDirectory walks through a directory and replaces strings in all text files.
// It processes .py, .sql, .yml, .yaml, .md, .txt, .toml files.
func replaceInDirectory(dir string, replacements map[string]string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file extension is text-based
		ext := strings.ToLower(filepath.Ext(path))
		textExts := []string{".py", ".sql", ".yml", ".yaml", ".md", ".txt", ".toml", ".json", ".sh"}
		isTextFile := false
		for _, textExt := range textExts {
			if ext == textExt {
				isTextFile = true
				break
			}
		}

		if !isTextFile {
			return nil
		}

		return replaceInFile(path, replacements)
	})
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, content, info.Mode())
	})
}

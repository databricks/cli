package init_template

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/apps-mcp/lib/common"
	"github.com/databricks/cli/experimental/apps-mcp/lib/prompts"
	"github.com/databricks/cli/experimental/apps-mcp/lib/state"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

const (
	defaultTemplateRepo = "https://github.com/databricks/cli"
	defaultTemplateDir  = "experimental/apps-mcp/templates/appkit"
	defaultBranch       = "main"
	templatePathEnvVar  = "DATABRICKS_APPKIT_TEMPLATE_PATH"
)

func validateAppNameLength(projectName string) error {
	const maxAppNameLength = 30
	const devTargetPrefix = "dev-"
	totalLength := len(devTargetPrefix) + len(projectName)
	if totalLength > maxAppNameLength {
		maxAllowed := maxAppNameLength - len(devTargetPrefix)
		return fmt.Errorf(
			"app name too long: 'dev-%s' is %d chars (max: %d). App name must be ≤%d characters",
			projectName, totalLength, maxAppNameLength, maxAllowed,
		)
	}
	return nil
}

func readClaudeMd(ctx context.Context, configFile string) {
	showFallback := func() {
		cmdio.LogString(ctx, "\nConsult with CLAUDE.md provided in the bundle if present.")
	}

	if configFile == "" {
		showFallback()
		return
	}

	configBytes, err := os.ReadFile(configFile)
	if err != nil {
		showFallback()
		return
	}

	var config map[string]any
	if err := json.Unmarshal(configBytes, &config); err != nil {
		showFallback()
		return
	}

	projectName, ok := config["project_name"].(string)
	if !ok || projectName == "" {
		showFallback()
		return
	}

	claudePath := filepath.Join(".", projectName, "CLAUDE.md")
	content, err := os.ReadFile(claudePath)
	if err != nil {
		showFallback()
		return
	}

	cmdio.LogString(ctx, "\n=== CLAUDE.md ===")
	cmdio.LogString(ctx, string(content))
	cmdio.LogString(ctx, "=================\n")
}

// newAppCmd creates the app subcommand for init-template.
func newAppCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Initialize a Databricks App using the appkit template",
		Args:  cobra.NoArgs,
		Long: `Initialize a Databricks App using the appkit template.

Examples:
  experimental apps-mcp tools init-template app --name my-app
  experimental apps-mcp tools init-template app --name my-app --warehouse abc123
  experimental apps-mcp tools init-template app --name my-app --description "My cool app"
  experimental apps-mcp tools init-template app --name my-app --output-dir ./projects

Environment variables:
  DATABRICKS_APPKIT_TEMPLATE_PATH  Override template source with local path (for development)

After initialization:
  databricks bundle deploy --target dev
`,
	}

	var name string
	var warehouse string
	var description string
	var outputDir string
	var describe bool

	cmd.Flags().StringVar(&name, "name", "", "Project name (required)")
	cmd.Flags().StringVar(&warehouse, "warehouse", "", "SQL warehouse ID")
	cmd.Flags().StringVar(&warehouse, "warehouse-id", "", "SQL warehouse ID (alias for --warehouse)")
	cmd.Flags().StringVar(&warehouse, "sql-warehouse-id", "", "SQL warehouse ID (alias for --warehouse)")
	cmd.Flags().StringVar(&warehouse, "sql_warehouse_id", "", "SQL warehouse ID (alias for --warehouse)")
	cmd.Flags().StringVar(&description, "description", "", "App description")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the initialized template to")
	cmd.Flags().BoolVar(&describe, "describe", false, "Display template schema without initializing")

	// Hide the alias flags from help
	cmd.Flags().MarkHidden("warehouse-id")
	cmd.Flags().MarkHidden("sql-warehouse-id")
	cmd.Flags().MarkHidden("sql_warehouse_id")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Resolve template source: env var override or default remote
		templatePathOrUrl := os.Getenv(templatePathEnvVar)
		templateDir := ""
		branch := ""

		if templatePathOrUrl == "" {
			templatePathOrUrl = defaultTemplateRepo
			templateDir = defaultTemplateDir
			branch = defaultBranch
		}

		// Describe mode - show schema only
		if describe {
			r := template.Resolver{
				TemplatePathOrUrl: templatePathOrUrl,
				ConfigFile:        "",
				OutputDir:         outputDir,
				TemplateDir:       templateDir,
				Branch:            branch,
			}

			tmpl, err := r.Resolve(ctx)
			if err != nil {
				return err
			}
			defer tmpl.Reader.Cleanup(ctx)

			schema, _, err := tmpl.Reader.LoadSchemaAndTemplateFS(ctx)
			if err != nil {
				return fmt.Errorf("failed to load template schema: %w", err)
			}

			schemaJSON, err := json.MarshalIndent(schema, "", "  ")
			if err != nil {
				return err
			}
			cmdio.LogString(ctx, string(schemaJSON))
			return nil
		}

		// Validate required flag
		if name == "" {
			return errors.New("--name is required")
		}

		if err := validateAppNameLength(name); err != nil {
			return err
		}

		// Build config map from flags
		configMap := map[string]any{
			"project_name": name,
		}
		if warehouse != "" {
			configMap["sql_warehouse_id"] = warehouse
		}
		if description != "" {
			configMap["app_description"] = description
		}

		// Write config to temp file
		configFile, err := writeConfigToTempFile(configMap)
		if err != nil {
			return err
		}
		defer os.Remove(configFile)

		// Create output directory if specified and doesn't exist
		if outputDir != "" {
			if err := os.MkdirAll(outputDir, 0o755); err != nil {
				return fmt.Errorf("create output directory: %w", err)
			}
		}

		r := template.Resolver{
			TemplatePathOrUrl: templatePathOrUrl,
			ConfigFile:        configFile,
			OutputDir:         outputDir,
			TemplateDir:       templateDir,
			Branch:            branch,
		}

		tmpl, err := r.Resolve(ctx)
		if err != nil {
			return err
		}
		defer tmpl.Reader.Cleanup(ctx)

		err = tmpl.Writer.Materialize(ctx, tmpl.Reader)
		if err != nil {
			return err
		}
		tmpl.Writer.LogTelemetry(ctx)

		// Determine actual output directory (template writes to subdirectory with project name)
		actualOutputDir := name
		if outputDir != "" {
			actualOutputDir = filepath.Join(outputDir, name)
		}

		// Count files and get absolute path
		absOutputDir, err := filepath.Abs(actualOutputDir)
		if err != nil {
			absOutputDir = actualOutputDir
		}
		fileCount := countFiles(absOutputDir)
		cmdio.LogString(ctx, common.FormatScaffoldSuccess("appkit", absOutputDir, fileCount))

		// Generate and print file tree structure
		fileTree, err := generateFileTree(absOutputDir)
		if err == nil && fileTree != "" {
			cmdio.LogString(ctx, "\nFile structure:")
			cmdio.LogString(ctx, fileTree)
		}

		// Inject L2 (target-specific guidance for apps)
		targetApps := prompts.MustExecuteTemplate("target_apps.tmpl", map[string]any{})
		cmdio.LogString(ctx, targetApps)

		// Inject L3 (template-specific guidance from CLAUDE.md)
		readClaudeMd(ctx, configFile)

		// Save initial scaffolded state
		if err := state.SaveState(absOutputDir, state.NewScaffolded()); err != nil {
			return fmt.Errorf("failed to save project state: %w", err)
		}

		return nil
	}
	return cmd
}

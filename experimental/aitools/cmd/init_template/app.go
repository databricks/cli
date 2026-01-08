package init_template

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/aitools/lib/state"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

const (
	defaultTemplateRepo = "https://github.com/databricks/cli"
	defaultTemplateDir  = "experimental/aitools/templates/appkit"
	defaultBranch       = "main"
	templatePathEnvVar  = "DATABRICKS_APPKIT_TEMPLATE_PATH"
)

func readClaudeMd(ctx context.Context, projectDir string) {
	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	content, err := os.ReadFile(claudePath)
	if err != nil {
		cmdio.LogString(ctx, "\nConsult with CLAUDE.md provided in the bundle if present.")
		return
	}

	cmdio.LogString(ctx, "\n=== CLAUDE.md ===")
	cmdio.LogString(ctx, string(content))
	cmdio.LogString(ctx, "=================\n")
}

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

// newAppCmd creates the app subcommand for init-template.
func newAppCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Initialize a Databricks App using the appkit template",
		Args:  cobra.NoArgs,
		Long: `Initialize a Databricks App using the appkit template.

Examples:
  experimental aitools tools init-template app --name my-app
  experimental aitools tools init-template app --name my-app --warehouse abc123
  experimental aitools tools init-template app --name my-app --description "My cool app"
  experimental aitools tools init-template app --name my-app --output-dir ./projects

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

		err := MaterializeTemplate(ctx, TemplateConfig{
			TemplatePath: templatePathOrUrl,
			TemplateName: "appkit",
			TemplateDir:  templateDir,
			Branch:       branch,
		}, configMap, name, outputDir)
		if err != nil {
			return err
		}

		projectDir := filepath.Join(outputDir, name)

		// Inject L3 (template-specific guidance from CLAUDE.md)
		// (we only do this for the app template; other templates use a generic CLAUDE.md)
		readClaudeMd(ctx, projectDir)

		// Save initial scaffolded state for app state machine
		if err := state.SaveState(projectDir, state.NewScaffolded()); err != nil {
			return fmt.Errorf("failed to save project state: %w", err)
		}
		return nil
	}
	return cmd
}

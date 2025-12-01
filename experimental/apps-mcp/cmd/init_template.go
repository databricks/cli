package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/apps-mcp/lib/common"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
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

func newInitTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init-template [TEMPLATE_PATH]",
		Short: "Initialize using a bundle template",
		Args:  root.MaximumNArgs(1),
		Long: fmt.Sprintf(`Initialize using a bundle template to get started quickly.

TEMPLATE_PATH optionally specifies which template to use. It can be one of the following:
%s
- a local file system path with a template directory
- a Git repository URL, e.g. https://github.com/my/repository

Supports the same options as 'databricks bundle init' plus:
  --describe: Display template schema without materializing
  --config_json: Provide config as JSON string instead of file

Examples:
  experimental apps-mcp tools init-template                   # Choose from built-in templates
  experimental apps-mcp tools init-template default-python    # Python jobs and notebooks
  experimental apps-mcp tools init-template --output-dir ./my-project
  experimental apps-mcp tools init-template default-python --describe
  experimental apps-mcp tools init-template default-python --config_json '{"project_name":"my-app"}'

After initialization:
  databricks bundle deploy --target dev

See https://docs.databricks.com/en/dev-tools/bundles/templates.html for more information on templates.`, template.HelpDescriptions()),
	}

	var configFile string
	var outputDir string
	var templateDir string
	var tag string
	var branch string
	var configJSON string
	var describe bool

	cmd.Flags().StringVar(&configFile, "config-file", "", "JSON file containing key value pairs of input parameters required for template initialization.")
	cmd.Flags().StringVar(&templateDir, "template-dir", "", "Directory path within a Git repository containing the template.")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the initialized template to.")
	cmd.Flags().StringVar(&branch, "tag", "", "Git tag to use for template initialization")
	cmd.Flags().StringVar(&tag, "branch", "", "Git branch to use for template initialization")
	cmd.Flags().StringVar(&configJSON, "config-json", "", "JSON string containing key value pairs (alternative to --config-file).")
	cmd.Flags().BoolVar(&describe, "describe", false, "Display template schema without initializing")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if tag != "" && branch != "" {
			return errors.New("only one of --tag or --branch can be specified")
		}

		if configFile != "" && configJSON != "" {
			return errors.New("only one of --config-file or --config-json can be specified")
		}

		if configFile != "" {
			if configBytes, err := os.ReadFile(configFile); err == nil {
				var userConfigMap map[string]any
				if err := json.Unmarshal(configBytes, &userConfigMap); err == nil {
					if projectName, ok := userConfigMap["project_name"].(string); ok {
						if err := validateAppNameLength(projectName); err != nil {
							return err
						}
					}
				}
			}
		}

		var templatePathOrUrl string
		if len(args) > 0 {
			templatePathOrUrl = args[0]
		}

		ctx := cmd.Context()

		// NEW: Describe mode - show schema only
		if describe {
			r := template.Resolver{
				TemplatePathOrUrl: templatePathOrUrl,
				ConfigFile:        "",
				OutputDir:         outputDir,
				TemplateDir:       templateDir,
				Tag:               tag,
				Branch:            branch,
			}

			tmpl, err := r.Resolve(ctx)
			if errors.Is(err, template.ErrCustomSelected) {
				cmdio.LogString(ctx, "Please specify a path or Git repository to use a custom template.")
				cmdio.LogString(ctx, "See https://docs.databricks.com/en/dev-tools/bundles/templates.html to learn more about custom templates.")
				return nil
			}
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

		if configJSON != "" {
			var userConfigMap map[string]any
			if err := json.Unmarshal([]byte(configJSON), &userConfigMap); err != nil {
				return fmt.Errorf("invalid JSON in --config-json: %w", err)
			}

			// Validate app name length
			if projectName, ok := userConfigMap["project_name"].(string); ok {
				if err := validateAppNameLength(projectName); err != nil {
					return err
				}
			}

			tmpFile, err := os.CreateTemp("", "mcp-template-config-*.json")
			if err != nil {
				return fmt.Errorf("create temp config file: %w", err)
			}
			defer os.Remove(tmpFile.Name())

			configBytes, err := json.Marshal(userConfigMap)
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}
			if _, err := tmpFile.Write(configBytes); err != nil {
				return fmt.Errorf("write config file: %w", err)
			}
			if err := tmpFile.Close(); err != nil {
				return fmt.Errorf("close config file: %w", err)
			}

			configFile = tmpFile.Name()
		}

		// Standard materialize flow (identical to bundle/init.go)
		r := template.Resolver{
			TemplatePathOrUrl: templatePathOrUrl,
			ConfigFile:        configFile,
			OutputDir:         outputDir,
			TemplateDir:       templateDir,
			Tag:               tag,
			Branch:            branch,
		}

		tmpl, err := r.Resolve(ctx)
		if errors.Is(err, template.ErrCustomSelected) {
			cmdio.LogString(ctx, "Please specify a path or Git repository to use a custom template.")
			cmdio.LogString(ctx, "See https://docs.databricks.com/en/dev-tools/bundles/templates.html to learn more about custom templates.")
			return nil
		}
		if err != nil {
			return err
		}
		defer tmpl.Reader.Cleanup(ctx)

		err = tmpl.Writer.Materialize(ctx, tmpl.Reader)
		if err != nil {
			return err
		}
		tmpl.Writer.LogTelemetry(ctx)

		// Show branded success message
		templateName := "bundle"
		if templatePathOrUrl != "" {
			templateName = filepath.Base(templatePathOrUrl)
		}
		outputPath := outputDir
		if outputPath == "" {
			outputPath = "."
		}
		// Count files if we can
		fileCount := 0
		if absPath, err := filepath.Abs(outputPath); err == nil {
			_ = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() {
					fileCount++
				}
				return nil
			})
		}
		cmdio.LogString(ctx, common.FormatScaffoldSuccess(templateName, outputPath, fileCount))

		// Try to read and display CLAUDE.md if present
		readClaudeMd(ctx, configFile)

		return nil
	}
	return cmd
}

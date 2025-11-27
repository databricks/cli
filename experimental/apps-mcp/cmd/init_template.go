package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

type templateConfig struct {
	repo string
	dir  string
}

var templateRegistry = map[string]templateConfig{
	"apps": {
		repo: "https://github.com/neondatabase/appdotbuild-agent",
		dir:  "edda/edda_templates/trpc_bundle",
	},
}

func getTemplateTypes() []string {
	types := make([]string, 0, len(templateRegistry))
	for t := range templateRegistry {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

func formatSchemaForDisplay(ctx *cobra.Command, schema *jsonschema.Schema, templateType string) {
	if len(schema.Properties) == 0 {
		return // Skip display for empty schemas
	}

	cmdio.LogString(ctx.Context(), "\nTemplate Configuration Variables:")
	cmdio.LogString(ctx.Context(), "==================================\n")

	for _, prop := range schema.OrderedProperties() {
		if prop.Schema.SkipPromptIf != nil && prop.Schema.Default == nil {
			continue
		}

		cmdio.LogString(ctx.Context(), fmt.Sprintf("\n%s (%s)", prop.Name, prop.Schema.Type))

		if prop.Schema.Description != "" {
			desc := strings.TrimSpace(prop.Schema.Description)
			desc = strings.ReplaceAll(desc, "\\n", "\n")
			cmdio.LogString(ctx.Context(), "  Description: "+desc)
		}

		if prop.Schema.Default != nil {
			cmdio.LogString(ctx.Context(), fmt.Sprintf("  Default: %v", prop.Schema.Default))
		}
		if len(prop.Schema.Enum) > 0 {
			cmdio.LogString(ctx.Context(), "  Options:")
			for _, opt := range prop.Schema.Enum {
				cmdio.LogString(ctx.Context(), fmt.Sprintf("    - %v", opt))
			}
		}

		for _, req := range schema.Required {
			if req == prop.Name {
				cmdio.LogString(ctx.Context(), "  Required: yes")
				break
			}
		}
	}

	cmdio.LogString(ctx.Context(), "\n\nTo initialize the template with these values, use:")
	cmdio.LogString(ctx.Context(), fmt.Sprintf("  experimental apps-mcp tools init-template %s --config_json '{\"key\":\"value\",...}'", templateType))
}

func newInitTemplateCmd() *cobra.Command {
	var configJSON string

	cmd := &cobra.Command{
		Use:   "init-template TEMPLATE_TYPE",
		Short: "Initialize a new app from template",
		Long: `Initialize a new Databricks app from a template.

Supported template types: apps

When run without --config_json, displays the template schema and exits.
When run with --config_json, initializes the template with the provided configuration.`,
		Example: `  # Display template schema
  experimental apps-mcp tools init-template apps

  # Initialize with configuration
  experimental apps-mcp tools init-template apps --config_json '{"project_name":"my-app"}'`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("accepts 1 arg, received %d", len(args))
			}
			templateType := args[0]
			if _, ok := templateRegistry[templateType]; !ok {
				return fmt.Errorf("unknown template type %q. Supported types: %s",
					templateType, strings.Join(getTemplateTypes(), ", "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			templateType := args[0]
			tmplCfg := templateRegistry[templateType]

			outputDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			var userConfigMap map[string]any
			if configJSON != "" {
				if err := json.Unmarshal([]byte(configJSON), &userConfigMap); err != nil {
					return fmt.Errorf("invalid JSON in --config_json: %w", err)
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

			r := template.Resolver{
				TemplatePathOrUrl: tmplCfg.repo,
				ConfigFile:        tmpFile.Name(),
				OutputDir:         outputDir,
				TemplateDir:       tmplCfg.dir,
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

			if configJSON == "" {
				if len(schema.Properties) > 0 {
					formatSchemaForDisplay(cmd, schema, templateType)
					return nil // Exit without materializing
				}
			}

			err = tmpl.Writer.Materialize(ctx, tmpl.Reader)
			if err != nil {
				return err
			}

			tmpl.Writer.LogTelemetry(ctx)
			return nil
		},
	}

	cmd.Flags().StringVar(&configJSON, "config_json", "", "JSON string with configuration values")

	return cmd
}

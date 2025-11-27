package mcp

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/apps-mcp/lib/middlewares"
	"github.com/databricks/cli/experimental/apps-mcp/lib/session"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

const (
	defaultTemplateRepo = "https://github.com/neondatabase/appdotbuild-agent"
	defaultTemplateDir  = "edda/edda_templates/trpc_bundle"
)

func newInitTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init-template PROJECT_NAME",
		Short: "Initialize a new app from template",
		Long: `Initialize a new Databricks app from a template.

This is a shortcut for 'bundle init' with the default MCP app template.
Auto-detects the SQL warehouse ID unless DATABRICKS_WAREHOUSE_ID is set.`,
		Example: `  databricks experimental apps-mcp tools init-template my-app`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			projectName := args[0]
			outputDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			// set up session with client for middleware compatibility
			sess := session.NewSession()
			sess.Set(middlewares.DatabricksClientKey, w)
			ctx = session.WithSession(ctx, sess)

			warehouseID, err := middlewares.GetWarehouseID(ctx)
			if err != nil {
				return err
			}

			// create temp config file with parameters
			configMap := map[string]string{
				"project_name":     projectName,
				"sql_warehouse_id": warehouseID,
			}
			configBytes, err := json.Marshal(configMap)
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}

			tmpFile, err := os.CreateTemp("", "mcp-template-config-*.json")
			if err != nil {
				return fmt.Errorf("create temp config file: %w", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.Write(configBytes); err != nil {
				return fmt.Errorf("write config file: %w", err)
			}
			if err := tmpFile.Close(); err != nil {
				return fmt.Errorf("close config file: %w", err)
			}

			r := template.Resolver{
				TemplatePathOrUrl: defaultTemplateRepo,
				ConfigFile:        tmpFile.Name(),
				OutputDir:         outputDir,
				TemplateDir:       defaultTemplateDir,
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
			return nil
		},
	}

	return cmd
}

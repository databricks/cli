package generate

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

func NewGenerateAlertCommand() *cobra.Command {
	var alertID string
	var configDir string
	var sourceDir string
	var force bool

	cmd := &cobra.Command{
		Use:   "alert",
		Short: "Generate configuration for an alert",
		Long: `Generate bundle configuration for an existing Databricks alert.

This command downloads an existing SQL alert and creates bundle files
that you can use to deploy the alert to other environments or manage it as code.

Examples:
  # Generate alert configuration by ID
  databricks bundle generate alert --existing-id abc123

  # Specify custom directories for organization
  databricks bundle generate alert --existing-id abc123 \
    --key my_alert --config-dir resources --source-dir src

What gets generated:
- Alert configuration YAML file with settings and a reference to the alert definition
- Alert definition (.dbalert.json) file with the complete alert specification

After generation, you can deploy this alert to other targets using:
  databricks bundle deploy --target staging
  databricks bundle deploy --target prod`,
	}

	cmd.Flags().StringVar(&alertID, "existing-id", "", `ID of the alert to generate configuration for`)
	cmd.MarkFlagRequired("existing-id")

	addOutputDirFlag(cmd, &configDir, "config-dir", "d", "resources", `directory to write the configuration to`)
	addOutputDirFlag(cmd, &sourceDir, "source-dir", "s", "src", `directory to write the alert definition to`)
	addForceFlag(cmd, &force, `force overwrite existing files in the output directory`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, b, err := configureBundle(cmd)
		if err != nil {
			return err
		}

		w := b.WorkspaceClient(ctx)

		// Get alert from Databricks
		alert, err := w.AlertsV2.GetAlert(ctx, sql.GetAlertV2Request{Id: alertID})
		if err != nil {
			// Check if it's a not found error to provide a better message
			if apiErr, ok := errors.AsType[*apierr.APIError](err); ok && apiErr.StatusCode == http.StatusNotFound {
				return fmt.Errorf("alert with ID %s not found", alertID)
			}
			return err
		}

		// Calculate paths
		alertKey := selectedResourceKey(cmd, alert.DisplayName)

		// Make paths absolute if they aren't already
		configDir = absolutePath(b.BundleRootPath, configDir)
		sourceDir = absolutePath(b.BundleRootPath, sourceDir)

		// Calculate relative path from config dir to source dir
		relativeSourceDir, err := filepath.Rel(configDir, sourceDir)
		if err != nil {
			return err
		}
		relativeSourceDir = filepath.ToSlash(relativeSourceDir)

		// Save alert definition to source directory
		alertBasename := alertKey + ".dbalert.json"
		alertPath := filepath.Join(sourceDir, alertBasename)

		// remote alert path
		remoteAlertPath := path.Join(alert.ParentPath, alert.DisplayName+".dbalert.json")
		resp, err := w.Workspace.Export(ctx, workspace.ExportRequest{
			Path: remoteAlertPath,
		})
		if err != nil {
			return err
		}
		alertJSON, err := base64.StdEncoding.DecodeString(resp.Content)
		if err != nil {
			return err
		}

		// Create source directory if needed
		if err := os.MkdirAll(sourceDir, 0o755); err != nil {
			return err
		}

		// Check if file exists and force flag
		if _, err := os.Stat(alertPath); err == nil && !force {
			return fmt.Errorf("%s already exists. Use --force to overwrite", filepath.ToSlash(alertPath))
		}

		// Write alert definition file
		if err := os.WriteFile(alertPath, alertJSON, 0o644); err != nil {
			return err
		}

		// Convert alert to bundle configuration
		v, err := generate.ConvertAlertToValue(alert, path.Join(relativeSourceDir, alertBasename))
		if err != nil {
			return err
		}

		result := generatedResourceConfig("alerts", alertKey, v)

		// Create config directory if needed
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			return err
		}

		// Save configuration file
		configPath := filepath.Join(configDir, alertKey+".alert.yml")
		err = saveGeneratedResourceConfig(result, configPath, force, map[string]yaml.Style{
			"display_name": yaml.DoubleQuotedStyle,
		})
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, "Alert configuration successfully saved to "+filepath.ToSlash(configPath))
		cmdio.LogString(ctx, "Serialized alert definition to "+filepath.ToSlash(alertPath))

		warnIfNotIncluded(ctx, b, configPath)

		return nil
	}

	return cmd
}

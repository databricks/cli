package generate

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/textutil"
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
	var useYAML bool

	cmd := &cobra.Command{
		Use:   "alert",
		Short: "Generate configuration for an alert",
		Long: `Generate bundle configuration for an existing Databricks alert.

This command downloads an existing SQL alert and creates bundle files
that you can use to deploy the alert to other environments or manage it as code.

Examples:
  # Generate alert configuration by ID
  databricks bundle generate alert --existing-id abc123

  # Generate with inline YAML definition (no separate .dbalert.json file)
  databricks bundle generate alert --existing-id abc123 --yaml

  # Specify custom directories for organization
  databricks bundle generate alert --existing-id abc123 \
    --key my_alert --config-dir resources --source-dir src

What gets generated:
- Alert configuration YAML file with settings and a reference to the alert definition
- Alert definition (.dbalert.json) file with the complete alert specification (unless --yaml is used)

When using --yaml flag:
- The alert definition is embedded directly in the YAML configuration
- No separate .dbalert.json file is created
- All alert settings are in a single file for easier management

After generation, you can deploy this alert to other targets using:
  databricks bundle deploy --target staging
  databricks bundle deploy --target prod`,
	}

	cmd.Flags().StringVar(&alertID, "existing-id", "", `ID of the alert to generate configuration for`)
	cmd.MarkFlagRequired("existing-id")

	// Alias lookup flag that includes the resource type name.
	// Included for symmetry with the other generate commands, but we prefer the shorter flag.
	cmd.Flags().StringVar(&alertID, "existing-alert-id", "", `ID of the alert to generate configuration for`)
	cmd.Flags().MarkHidden("existing-alert-id")

	cmd.Flags().StringVarP(&configDir, "config-dir", "d", "resources", `directory to write the configuration to`)
	cmd.Flags().StringVarP(&sourceDir, "source-dir", "s", "src", `directory to write the alert definition to`)
	cmd.Flags().BoolVarP(&force, "force", "f", false, `force overwrite existing files in the output directory`)
	cmd.Flags().BoolVar(&useYAML, "yaml", false, `embed alert definition in YAML configuration instead of separate file`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := root.MustConfigureBundle(cmd)
		if b == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		w := b.WorkspaceClient()

		// Get alert from Databricks
		alert, err := w.AlertsV2.GetAlert(ctx, sql.GetAlertV2Request{Id: alertID})
		if err != nil {
			// Check if it's a not found error to provide a better message
			var apiErr *apierr.APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
				return fmt.Errorf("alert with ID %s not found", alertID)
			}
			return err
		}

		// Calculate paths
		alertKey := cmd.Flag("key").Value.String()
		if alertKey == "" {
			alertKey = textutil.NormalizeString(alert.DisplayName)
		}

		// Make paths absolute if they aren't already
		if !filepath.IsAbs(configDir) {
			configDir = filepath.Join(b.BundleRootPath, configDir)
		}
		if !filepath.IsAbs(sourceDir) {
			sourceDir = filepath.Join(b.BundleRootPath, sourceDir)
		}

		// Fetch alert definition from workspace
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

		var v dyn.Value
		var alertPath string

		if useYAML {
			// Embed definition directly in YAML
			v, err = generate.ConvertAlertToValueWithDefinition(alert, alertJSON)
			if err != nil {
				return err
			}
		} else {
			// Save alert definition to separate file
			// Calculate relative path from config dir to source dir
			relativeSourceDir, err := filepath.Rel(configDir, sourceDir)
			if err != nil {
				return err
			}
			relativeSourceDir = filepath.ToSlash(relativeSourceDir)

			alertBasename := alertKey + ".dbalert.json"
			alertPath = filepath.Join(sourceDir, alertBasename)

			// Create source directory if needed
			if err := os.MkdirAll(sourceDir, 0o755); err != nil {
				return err
			}

			// Check if file exists and force flag
			if _, err := os.Stat(alertPath); err == nil && !force {
				return fmt.Errorf("%s already exists. Use --force to overwrite", alertPath)
			}

			// Write alert definition file
			if err := os.WriteFile(alertPath, alertJSON, 0o644); err != nil {
				return err
			}

			// Convert alert to bundle configuration with file reference
			v, err = generate.ConvertAlertToValue(alert, path.Join(relativeSourceDir, alertBasename))
			if err != nil {
				return err
			}
		}

		result := map[string]dyn.Value{
			"resources": dyn.V(map[string]dyn.Value{
				"alerts": dyn.V(map[string]dyn.Value{
					alertKey: v,
				}),
			}),
		}

		// Create config directory if needed
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			return err
		}

		// Save configuration file
		configPath := filepath.Join(configDir, alertKey+".alert.yml")
		saver := yamlsaver.NewSaverWithStyle(map[string]yaml.Style{
			"display_name": yaml.DoubleQuotedStyle,
		})

		err = saver.SaveAsYAML(result, configPath, force)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, "Alert configuration successfully saved to "+configPath)
		if !useYAML && alertPath != "" {
			cmdio.LogString(ctx, "Serialized alert definition to "+alertPath)
		}

		return nil
	}

	return cmd
}

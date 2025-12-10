package generate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
)

type alert struct {
	// Lookup flags for one-time generate.
	existingID string

	// Lookup flag for existing bundle resource.
	resource string

	// Where to write the configuration and alert representation.
	resourceDir string
	alertDir    string

	// Force overwrite of existing files.
	force bool

	// Relative path from the resource directory to the alert directory.
	relativeAlertDir string

	// Command.
	cmd *cobra.Command

	// Output and error streams.
	out io.Writer
	err io.Writer
}

func (a *alert) resolveFromID(ctx context.Context, b *bundle.Bundle) string {
	if a.existingID == "" {
		logdiag.LogError(ctx, errors.New("expected --alert-id"))
		return ""
	}

	w := b.WorkspaceClient()
	obj, err := w.AlertsV2.GetAlert(ctx, sql.GetAlertV2Request{Id: a.existingID})
	if err != nil {
		if apierr.IsMissing(err) {
			logdiag.LogError(ctx, fmt.Errorf("alert with ID %s not found", a.existingID))
			return ""
		}
		logdiag.LogError(ctx, err)
		return ""
	}

	return obj.Id
}

func (a *alert) saveAlertDefinition(_ context.Context, b *bundle.Bundle, alert *sql.AlertV2, filename string) error {
	// Marshal the alert to JSON.
	data, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	// Unmarshal and remarshal to ensure it has a stable format.
	data, err = remarshalJSON(data)
	if err != nil {
		return err
	}

	// Make sure the output directory exists.
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}

	// Clean the filename to ensure it is a valid path (and can be used on this OS).
	filename = filepath.Clean(filename)

	// Attempt to make the path relative to the bundle root.
	rel, err := filepath.Rel(b.BundleRootPath, filename)
	if err != nil {
		rel = filename
	}

	// Verify that the file does not already exist.
	info, err := os.Stat(filename)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", rel)
		}
		if !a.force {
			return fmt.Errorf("%s already exists. Use --force to overwrite", rel)
		}
	}

	fmt.Fprintf(a.out, "Writing alert to %q\n", rel)
	return os.WriteFile(filename, data, 0o644)
}

func (a *alert) saveConfiguration(ctx context.Context, b *bundle.Bundle, alert *sql.AlertV2, key string) error {
	// Save alert definition to the alert directory.
	alertBasename := key + ".dbalert.json"
	alertPath := filepath.Join(a.alertDir, alertBasename)
	err := a.saveAlertDefinition(ctx, b, alert, alertPath)
	if err != nil {
		return err
	}

	// Synthesize resource configuration.
	v, err := generate.ConvertAlertToValue(alert, path.Join(a.relativeAlertDir, alertBasename))
	if err != nil {
		return err
	}

	result := map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			"alerts": dyn.V(map[string]dyn.Value{
				key: v,
			}),
		}),
	}

	// Make sure the output directory exists.
	if err := os.MkdirAll(a.resourceDir, 0o755); err != nil {
		return err
	}

	// Save the configuration to the resource directory.
	resourcePath := filepath.Join(a.resourceDir, key+".alert.yml")
	saver := yamlsaver.NewSaverWithStyle(map[string]yaml.Style{
		"display_name": yaml.DoubleQuotedStyle,
	})

	// Attempt to make the path relative to the bundle root.
	rel, err := filepath.Rel(b.BundleRootPath, resourcePath)
	if err != nil {
		rel = resourcePath
	}

	fmt.Fprintf(a.out, "Writing configuration to %q\n", rel)
	err = saver.SaveAsYAML(result, resourcePath, a.force)
	if err != nil {
		return err
	}

	return nil
}

func (a *alert) generateForExisting(ctx context.Context, b *bundle.Bundle, alertID string) {
	w := b.WorkspaceClient()
	alert, err := w.AlertsV2.GetAlert(ctx, sql.GetAlertV2Request{Id: alertID})
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	key := textutil.NormalizeString(alert.DisplayName)
	err = a.saveConfiguration(ctx, b, alert, key)
	if err != nil {
		logdiag.LogError(ctx, err)
	}
}

func (a *alert) initialize(ctx context.Context, b *bundle.Bundle) {
	// Make the paths absolute if they aren't already.
	if !filepath.IsAbs(a.resourceDir) {
		a.resourceDir = filepath.Join(b.BundleRootPath, a.resourceDir)
	}
	if !filepath.IsAbs(a.alertDir) {
		a.alertDir = filepath.Join(b.BundleRootPath, a.alertDir)
	}

	// Make sure we know how the alert path is relative to the resource path.
	rel, err := filepath.Rel(a.resourceDir, a.alertDir)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	a.relativeAlertDir = filepath.ToSlash(rel)
}

func (a *alert) runForResource(ctx context.Context, b *bundle.Bundle) {
	engine, err := engine.FromEnv(ctx)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	phases.Initialize(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	ctx, stateDesc := statemgmt.PullResourcesState(ctx, b, statemgmt.AlwaysPull(true), engine)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplySeqContext(ctx, b,
		statemgmt.Load(stateDesc.Engine),
	)
	if logdiag.HasError(ctx) {
		return
	}

	resource, ok := b.Config.Resources.Alerts[a.resource]
	if !ok {
		logdiag.LogError(ctx, fmt.Errorf("alert resource %q is not defined", a.resource))
		return
	}

	if resource.FilePath == "" {
		logdiag.LogError(ctx, fmt.Errorf("alert resource %q has no file path defined", a.resource))
		return
	}

	// Resolve the alert ID from the resource.
	alertID := resource.ID

	// Overwrite the alert at the path referenced from the resource.
	alertPath := resource.FilePath

	w := b.WorkspaceClient()
	alert, err := w.AlertsV2.GetAlert(ctx, sql.GetAlertV2Request{Id: alertID})
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	err = a.saveAlertDefinition(ctx, b, alert, alertPath)
	if err != nil {
		logdiag.LogError(ctx, err)
	}
}

func (a *alert) runForExisting(ctx context.Context, b *bundle.Bundle) {
	// Resolve the ID of the alert to generate configuration for.
	alertID := a.resolveFromID(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	a.generateForExisting(ctx, b, alertID)
}

func (a *alert) RunE(cmd *cobra.Command, args []string) error {
	ctx := logdiag.InitContext(cmd.Context())
	cmd.SetContext(ctx)

	b := root.MustConfigureBundle(cmd)
	if b == nil || logdiag.HasError(ctx) {
		return root.ErrAlreadyPrinted
	}

	a.initialize(ctx, b)
	if logdiag.HasError(ctx) {
		return root.ErrAlreadyPrinted
	}

	if a.resource != "" {
		a.runForResource(ctx, b)
	} else {
		a.runForExisting(ctx, b)
	}

	if logdiag.HasError(ctx) {
		return root.ErrAlreadyPrinted
	}

	return nil
}

// filterAlerts returns a filter that only includes alerts.
func filterAlerts(ref resources.Reference) bool {
	return ref.Description.SingularName == "alert"
}

// alertResourceCompletion executes to autocomplete the argument to the resource flag.
func alertResourceCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	b := root.MustConfigureBundle(cmd)
	if logdiag.HasError(cmd.Context()) {
		return nil, cobra.ShellCompDirectiveError
	}

	if b == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return maps.Keys(resources.Completions(b, filterAlerts)), cobra.ShellCompDirectiveNoFileComp
}

func NewGenerateAlertCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alert",
		Short: "Generate configuration for an alert",
		Long: `Generate bundle configuration for an existing Databricks alert.

This command downloads an existing SQL alert and creates bundle files
that you can use to deploy the alert to other environments or manage it as code.

Examples:
  # Generate alert configuration by ID
  databricks bundle generate alert --existing-id abc123

  # Update existing resource configuration
  databricks bundle generate alert --resource my_alert --force

What gets generated:
- Alert configuration YAML file with settings and a reference to the alert definition
- Alert definition (.dbalert.json) file with the complete alert specification

Sync workflow for alert development:
When developing alerts, you can modify them in the Databricks UI and sync
changes back to your bundle:

1. Make changes to alert in the Databricks UI
2. Run: databricks bundle generate alert --resource my_alert --force
3. Commit changes to version control
4. Deploy to other environments with: databricks bundle deploy --target prod`,
	}

	a := &alert{
		out: cmd.OutOrStdout(),
		err: cmd.ErrOrStderr(),
	}

	// Lookup flags.
	cmd.Flags().StringVar(&a.existingID, "existing-id", "", `ID of the alert to generate configuration for`)
	cmd.Flags().StringVar(&a.resource, "resource", "", `resource key of alert to update`)

	// Alias lookup flag that includes the resource type name.
	cmd.Flags().StringVar(&a.existingID, "existing-alert-id", "", `ID of the alert to generate configuration for`)
	cmd.Flags().MarkHidden("existing-alert-id")

	// Output flags.
	cmd.Flags().StringVarP(&a.resourceDir, "resource-dir", "d", "resources", `directory to write the configuration to`)
	cmd.Flags().StringVarP(&a.alertDir, "alert-dir", "s", "src", `directory to write the alert definition to`)
	cmd.Flags().BoolVarP(&a.force, "force", "f", false, `force overwrite existing files in the output directory`)

	// Exactly one of the lookup flags must be provided.
	cmd.MarkFlagsOneRequired(
		"existing-id",
		"resource",
	)

	// Completion for the resource flag.
	cmd.RegisterFlagCompletionFunc("resource", alertResourceCompletion)

	cmd.RunE = a.RunE
	a.cmd = cmd
	return cmd
}

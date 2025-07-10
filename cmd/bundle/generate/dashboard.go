package generate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
)

type dashboard struct {
	// Lookup flags for one-time generate.
	existingPath string
	existingID   string

	// Lookup flag for existing bundle resource.
	resource string

	// Where to write the configuration and dashboard representation.
	resourceDir  string
	dashboardDir string

	// Force overwrite of existing files.
	force bool

	// Watch for changes to the dashboard.
	watch bool

	// Relative path from the resource directory to the dashboard directory.
	relativeDashboardDir string
}

func (d *dashboard) resolveID(ctx context.Context, b *bundle.Bundle) string {
	switch {
	case d.existingPath != "":
		return d.resolveFromPath(ctx, b)
	case d.existingID != "":
		return d.resolveFromID(ctx, b)
	}

	logdiag.LogError(ctx, errors.New("expected one of --dashboard-path, --dashboard-id"))
	return ""
}

func (d *dashboard) resolveFromPath(ctx context.Context, b *bundle.Bundle) string {
	w := b.WorkspaceClient()
	obj, err := w.Workspace.GetStatusByPath(ctx, d.existingPath)
	if err != nil {
		if apierr.IsMissing(err) {
			logdiag.LogError(ctx, fmt.Errorf("dashboard %q not found", path.Base(d.existingPath)))
			return ""
		}

		// Emit a more descriptive error message for legacy dashboards.
		if errors.Is(err, apierr.ErrBadRequest) && strings.HasPrefix(err.Error(), "dbsqlDashboard ") {
			logdiag.LogDiag(ctx, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("dashboard %q is a legacy dashboard", path.Base(d.existingPath)),
				Detail: "" +
					"Databricks Asset Bundles work exclusively with AI/BI dashboards.\n" +
					"\n" +
					"Instructions on how to convert a legacy dashboard to an AI/BI dashboard\n" +
					"can be found at: https://docs.databricks.com/en/dashboards/clone-legacy-to-aibi.html.",
			})
			return ""
		}

		logdiag.LogError(ctx, err)
		return ""
	}

	if obj.ObjectType != workspace.ObjectTypeDashboard {
		found := strings.ToLower(obj.ObjectType.String())
		logdiag.LogDiag(ctx, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "expected a dashboard, found a " + found,
		})
		return ""
	}

	if obj.ResourceId == "" {
		logdiag.LogDiag(ctx, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "expected a non-empty dashboard resource ID",
		})
		return ""
	}

	return obj.ResourceId
}

func (d *dashboard) resolveFromID(ctx context.Context, b *bundle.Bundle) string {
	w := b.WorkspaceClient()
	obj, err := w.Lakeview.GetByDashboardId(ctx, d.existingID)
	if err != nil {
		if apierr.IsMissing(err) {
			logdiag.LogError(ctx, fmt.Errorf("dashboard with ID %s not found", d.existingID))
			return ""
		}
		logdiag.LogError(ctx, err)
		return ""
	}

	return obj.DashboardId
}

func remarshalJSON(data []byte) ([]byte, error) {
	var tmp any
	var err error
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		return nil, err
	}

	// Remarshal the data to ensure its formatting is stable.
	// The result will have alphabetically sorted keys and be indented.
	// HTML escaping is disabled to retain characters such as &, <, and >.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	err = enc.Encode(tmp)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (d *dashboard) saveSerializedDashboard(_ context.Context, b *bundle.Bundle, dashboard *dashboards.Dashboard, filename string) error {
	// Unmarshal and remarshal the serialized dashboard to ensure it is formatted correctly.
	// The result will have alphabetically sorted keys and be indented.
	data, err := remarshalJSON([]byte(dashboard.SerializedDashboard))
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
		if !d.force {
			return fmt.Errorf("%s already exists. Use --force to overwrite", rel)
		}
	}

	fmt.Printf("Writing dashboard to %q\n", rel)
	return os.WriteFile(filename, data, 0o644)
}

func (d *dashboard) saveConfiguration(ctx context.Context, b *bundle.Bundle, dashboard *dashboards.Dashboard, key string) error {
	// Save serialized dashboard definition to the dashboard directory.
	dashboardBasename := key + ".lvdash.json"
	dashboardPath := filepath.Join(d.dashboardDir, dashboardBasename)
	err := d.saveSerializedDashboard(ctx, b, dashboard, dashboardPath)
	if err != nil {
		return err
	}

	// Synthesize resource configuration.
	v, err := generate.ConvertDashboardToValue(dashboard, path.Join(d.relativeDashboardDir, dashboardBasename))
	if err != nil {
		return err
	}

	result := map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			"dashboards": dyn.V(map[string]dyn.Value{
				key: v,
			}),
		}),
	}

	// Make sure the output directory exists.
	if err := os.MkdirAll(d.resourceDir, 0o755); err != nil {
		return err
	}

	// Save the configuration to the resource directory.
	resourcePath := filepath.Join(d.resourceDir, key+".dashboard.yml")
	saver := yamlsaver.NewSaverWithStyle(map[string]yaml.Style{
		"display_name": yaml.DoubleQuotedStyle,
	})

	// Attempt to make the path relative to the bundle root.
	rel, err := filepath.Rel(b.BundleRootPath, resourcePath)
	if err != nil {
		rel = resourcePath
	}

	fmt.Printf("Writing configuration to %q\n", rel)
	err = saver.SaveAsYAML(result, resourcePath, d.force)
	if err != nil {
		return err
	}

	return nil
}

func waitForChanges(ctx context.Context, w *databricks.WorkspaceClient, dashboard *dashboards.Dashboard) diag.Diagnostics {
	// Compute [time.Time] for the most recent update.
	tref, err := time.Parse(time.RFC3339, dashboard.UpdateTime)
	if err != nil {
		return diag.FromErr(err)
	}

	for {
		obj, err := w.Workspace.GetStatusByPath(ctx, dashboard.Path)
		if err != nil {
			return diag.FromErr(err)
		}

		// Compute [time.Time] from timestamp in millis since epoch.
		tcur := time.Unix(0, obj.ModifiedAt*int64(time.Millisecond))
		if tcur.After(tref) {
			break
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

func (d *dashboard) updateDashboardForResource(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	resource, ok := b.Config.Resources.Dashboards[d.resource]
	if !ok {
		return diag.Errorf("dashboard resource %q is not defined", d.resource)
	}

	if resource.FilePath == "" {
		return diag.Errorf("dashboard resource %q has no file path defined", d.resource)
	}

	// Resolve the dashboard ID from the resource.
	dashboardID := resource.ID

	// Overwrite the dashboard at the path referenced from the resource.
	dashboardPath := resource.FilePath

	w := b.WorkspaceClient()

	// Start polling the underlying dashboard for changes.
	var etag string
	for {
		dashboard, err := w.Lakeview.GetByDashboardId(ctx, dashboardID)
		if err != nil {
			return diag.FromErr(err)
		}

		if etag != dashboard.Etag {
			err = d.saveSerializedDashboard(ctx, b, dashboard, dashboardPath)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		// Abort if we are not watching for changes.
		if !d.watch {
			return nil
		}

		// Update the etag for the next iteration.
		etag = dashboard.Etag

		// Now poll the workspace API for changes.
		// This is much more efficient than polling the dashboard API because it
		// includes the entire serialized dashboard whereas we're only interested
		// in the last modified time of the dashboard here.
		waitForChanges(ctx, w, dashboard)
	}
}

func (d *dashboard) generateForExisting(ctx context.Context, b *bundle.Bundle, dashboardID string) {
	w := b.WorkspaceClient()
	dashboard, err := w.Lakeview.GetByDashboardId(ctx, dashboardID)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	key := textutil.NormalizeString(dashboard.DisplayName)
	err = d.saveConfiguration(ctx, b, dashboard, key)
	if err != nil {
		logdiag.LogError(ctx, err)
	}
}

func (d *dashboard) initialize(ctx context.Context, b *bundle.Bundle) {
	// Make the paths absolute if they aren't already.
	if !filepath.IsAbs(d.resourceDir) {
		d.resourceDir = filepath.Join(b.BundleRootPath, d.resourceDir)
	}
	if !filepath.IsAbs(d.dashboardDir) {
		d.dashboardDir = filepath.Join(b.BundleRootPath, d.dashboardDir)
	}

	// Make sure we know how the dashboard path is relative to the resource path.
	rel, err := filepath.Rel(d.resourceDir, d.dashboardDir)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	d.relativeDashboardDir = filepath.ToSlash(rel)
}

func (d *dashboard) runForResource(ctx context.Context, b *bundle.Bundle) {
	phases.Initialize(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplySeqContext(ctx, b,
		terraform.Interpolate(),
		terraform.Write(),
		statemgmt.StatePull(),
		statemgmt.Load(),
	)
	if logdiag.HasError(ctx) {
		return
	}

	d.updateDashboardForResource(ctx, b)
}

func (d *dashboard) runForExisting(ctx context.Context, b *bundle.Bundle) {
	// Resolve the ID of the dashboard to generate configuration for.
	dashboardID := d.resolveID(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	d.generateForExisting(ctx, b, dashboardID)
}

func (d *dashboard) RunE(cmd *cobra.Command, args []string) error {
	ctx := logdiag.InitContext(cmd.Context())
	cmd.SetContext(ctx)

	b := root.MustConfigureBundle(cmd)
	if b == nil {
		return root.ErrAlreadyPrinted
	}

	d.initialize(ctx, b)
	if logdiag.HasError(ctx) {
		return root.ErrAlreadyPrinted
	}

	if d.resource != "" {
		d.runForResource(ctx, b)
	} else {
		d.runForExisting(ctx, b)
	}

	if logdiag.HasError(ctx) {
		return root.ErrAlreadyPrinted
	}

	return nil
}

// filterDashboards returns a filter that only includes dashboards.
func filterDashboards(ref resources.Reference) bool {
	return ref.Description.SingularName == "dashboard"
}

// dashboardResourceCompletion executes to autocomplete the argument to the resource flag.
func dashboardResourceCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	b := root.MustConfigureBundle(cmd)
	if logdiag.HasError(cmd.Context()) {
		return nil, cobra.ShellCompDirectiveError
	}

	if b == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return maps.Keys(resources.Completions(b, filterDashboards)), cobra.ShellCompDirectiveNoFileComp
}

func NewGenerateDashboardCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Generate configuration for a dashboard",
	}

	d := &dashboard{}

	// Lookup flags.
	cmd.Flags().StringVar(&d.existingPath, "existing-path", "", `workspace path of the dashboard to generate configuration for`)
	cmd.Flags().StringVar(&d.existingID, "existing-id", "", `ID of the dashboard to generate configuration for`)
	cmd.Flags().StringVar(&d.resource, "resource", "", `resource key of dashboard to watch for changes`)

	// Alias lookup flags that include the resource type name.
	// Included for symmetry with the other generate commands, but we prefer the shorter flags.
	cmd.Flags().StringVar(&d.existingPath, "existing-dashboard-path", "", `workspace path of the dashboard to generate configuration for`)
	cmd.Flags().StringVar(&d.existingID, "existing-dashboard-id", "", `ID of the dashboard to generate configuration for`)
	cmd.Flags().MarkHidden("existing-dashboard-path")
	cmd.Flags().MarkHidden("existing-dashboard-id")

	// Output flags.
	cmd.Flags().StringVarP(&d.resourceDir, "resource-dir", "d", "resources", `directory to write the configuration to`)
	cmd.Flags().StringVarP(&d.dashboardDir, "dashboard-dir", "s", "src", `directory to write the dashboard representation to`)
	cmd.Flags().BoolVarP(&d.force, "force", "f", false, `force overwrite existing files in the output directory`)

	// Exactly one of the lookup flags must be provided.
	cmd.MarkFlagsOneRequired(
		"existing-path",
		"existing-id",
		"resource",
	)

	// Watch flag. This is relevant only in combination with the resource flag.
	cmd.Flags().BoolVar(&d.watch, "watch", false, `watch for changes to the dashboard and update the configuration`)

	// Make sure the watch flag is only used with the existing-resource flag.
	cmd.MarkFlagsMutuallyExclusive("watch", "existing-path")
	cmd.MarkFlagsMutuallyExclusive("watch", "existing-id")

	// Completion for the resource flag.
	cmd.RegisterFlagCompletionFunc("resource", dashboardResourceCompletion)

	cmd.RunE = d.RunE
	return cmd
}

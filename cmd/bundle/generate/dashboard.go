package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/generate"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type dashboardLookup struct {
	dashboardPath string
	dashboardId   string
	resource      string
}

func (d *dashboardLookup) resolveID(ctx context.Context, b *bundle.Bundle) (string, diag.Diagnostics) {
	switch {
	case d.dashboardPath != "":
		return d.resolveFromPath(ctx, b)
	case d.dashboardId != "":
		return d.resolveFromID(ctx, b)
	case d.resource != "":
		return d.resolveFromResource(ctx, b)
	}

	return "", diag.Errorf("expected one of --dashboard-path, --dashboard-id, or --resource")
}

func (d *dashboardLookup) resolveFromPath(ctx context.Context, b *bundle.Bundle) (string, diag.Diagnostics) {
	w := b.WorkspaceClient()
	obj, err := w.Workspace.GetStatusByPath(ctx, d.dashboardPath)
	if err != nil {
		if apierr.IsMissing(err) {
			return "", diag.Errorf("dashboard at path %q not found", d.dashboardPath)
		}
		return "", diag.FromErr(err)
	}

	if obj.ObjectType != workspace.ObjectTypeDashboard {
		found := strings.ToLower(obj.ObjectType.String())
		return "", diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("expected a dashboard, found a %s", found),
			},
		}
	}

	if obj.ResourceId == "" {
		return "", diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "expected a non-empty dashboard resource ID",
			},
		}
	}

	return obj.ResourceId, nil
}

func (d *dashboardLookup) resolveFromID(ctx context.Context, b *bundle.Bundle) (string, diag.Diagnostics) {
	w := b.WorkspaceClient()
	obj, err := w.Lakeview.GetByDashboardId(ctx, d.dashboardId)
	if err != nil {
		if apierr.IsMissing(err) {
			return "", diag.Errorf("dashboard with ID %s not found", d.dashboardId)
		}
		return "", diag.FromErr(err)
	}

	return obj.DashboardId, nil
}

func (d *dashboardLookup) resolveFromResource(_ context.Context, b *bundle.Bundle) (string, diag.Diagnostics) {
	resource, ok := b.Config.Resources.Dashboards[d.resource]
	if !ok {
		return "", diag.Errorf("dashboard resource %q is not defined", d.resource)
	}

	if resource.ID == "" {
		return "", diag.Errorf("dashboard resource hasn't been deployed yet")
	}

	return resource.ID, nil
}

type dashboard struct {
	dashboardLookup

	resourceDir  string
	dashboardDir string
	force        bool
	watch        bool

	// Relative path from the resource directory to the dashboard directory.
	relativeDashboardDir string
}

func remarshalJSON(data []byte) ([]byte, error) {
	var tmp any
	var err error
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		return nil, err
	}
	out, err := json.MarshalIndent(tmp, "", "  ")
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (d *dashboard) saveSerializedDashboard(_ context.Context, b *bundle.Bundle, dashboard *dashboards.Dashboard, filename string) error {
	// Unmarshal and remarshal the serialized dashboard to ensure it is formatted correctly.
	// The result will have alphabetically sorted keys and be indented.
	data, err := remarshalJSON([]byte(dashboard.SerializedDashboard))
	if err != nil {
		return err
	}

	// Make sure the output directory exists.
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
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
	return os.WriteFile(filename, data, 0644)
}

func (d *dashboard) saveConfiguration(ctx context.Context, b *bundle.Bundle, dashboard *dashboards.Dashboard) error {
	key := d.dashboardLookup.resource
	if key == "" {
		key = textutil.NormalizeString(dashboard.DisplayName)
	}

	// Save serialized dashboard definition to the dashboard directory.
	dashboardBasename := fmt.Sprintf("%s.lvdash.json", key)
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
	if err := os.MkdirAll(d.resourceDir, 0755); err != nil {
		return err
	}

	// Save the configuration to the resource directory.
	resourcePath := filepath.Join(d.resourceDir, fmt.Sprintf("%s.yml", key))
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

func (d *dashboard) runWatch(ctx context.Context, b *bundle.Bundle, dashboardID string) diag.Diagnostics {
	resource, ok := b.Config.Resources.Dashboards[d.resource]
	if !ok {
		return diag.Errorf("dashboard resource %q is not defined", d.resource)
	}

	if resource.FilePath == "" {
		return diag.Errorf("dashboard resource %q has no file path defined", d.resource)
	}

	// Overwrite the dashboard at the path referenced from the resource.
	dashboardPath := resource.FilePath

	// Start polling the underlying dashboard for changes.
	var etag string
	for {
		w := b.WorkspaceClient()
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

		// Update the etag for the next iteration.
		etag = dashboard.Etag

		// Compute [time.Time] for the most recent update.
		tref, err := time.Parse(time.RFC3339, dashboard.UpdateTime)
		if err != nil {
			return diag.FromErr(err)
		}

		// Now poll the workspace API for changes.
		// This is much more efficient than polling the dashboard API.
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
	}
}

func (d *dashboard) runOnce(ctx context.Context, b *bundle.Bundle, dashboardID string) diag.Diagnostics {
	w := b.WorkspaceClient()
	dashboard, err := w.Lakeview.GetByDashboardId(ctx, dashboardID)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.saveConfiguration(ctx, b, dashboard)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func (d *dashboard) initialize(b *bundle.Bundle) diag.Diagnostics {
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
		return diag.FromErr(err)
	}

	d.relativeDashboardDir = filepath.ToSlash(rel)
	return nil
}

func (d *dashboard) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	b, diags := root.MustConfigureBundle(cmd)
	if diags.HasError() {
		return diags.Error()
	}

	diags = d.initialize(b)
	if diags.HasError() {
		return diags.Error()
	}

	diags = bundle.Apply(ctx, b, bundle.Seq(
		phases.Initialize(),
		terraform.Interpolate(),
		terraform.Write(),
		terraform.StatePull(),
		terraform.Load(),
	))
	if diags.HasError() {
		return diags.Error()
	}

	// Resolve the ID of the dashboard to generate configuration for.
	dashboardID, diags := d.resolveID(ctx, b)
	if diags.HasError() {
		return diags.Error()
	}

	if d.watch {
		diags = d.runWatch(ctx, b, dashboardID)
	} else {
		diags = d.runOnce(ctx, b, dashboardID)
	}

	return diags.Error()
}

func NewGenerateDashboardCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Generate configuration for a dashboard",
	}

	d := &dashboard{}

	// Lookup flags.
	cmd.Flags().StringVar(&d.dashboardPath, "existing-path", "", `workspace path of the dashboard to generate configuration for`)
	cmd.Flags().StringVar(&d.dashboardId, "existing-id", "", `ID of the dashboard to generate configuration for`)
	cmd.Flags().StringVar(&d.resource, "existing-resource", "", `resource key of dashboard to watch for changes`)

	// Output flags.
	cmd.Flags().StringVarP(&d.resourceDir, "resource-dir", "d", "./resources", `directory to write the configuration to`)
	cmd.Flags().StringVarP(&d.dashboardDir, "dashboard-dir", "s", "./src", `directory to write the dashboard representation to`)
	cmd.Flags().BoolVarP(&d.force, "force", "f", false, `force overwrite existing files in the output directory`)

	cmd.MarkFlagsOneRequired(
		"existing-path",
		"existing-id",
		"existing-resource",
	)

	// Watch flags.
	cmd.Flags().BoolVar(&d.watch, "watch", false, `watch for changes to the dashboard and update the configuration`)

	// Make sure the watch flag is only used with the existing-resource flag.
	cmd.MarkFlagsRequiredTogether("watch", "existing-resource")
	cmd.MarkFlagsOneRequired()

	cmd.RunE = d.RunE
	return cmd
}

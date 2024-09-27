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
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type dashboard struct {
	resourceDir  string
	dashboardDir string
	force        bool

	// Relative path from the resource directory to the dashboard directory.
	relativeDir string

	existingDashboardPath string
	existingDashboardId   string
	watch                 string

	key string
}

func (d *dashboard) resolveDashboardID(ctx context.Context, w *databricks.WorkspaceClient) diag.Diagnostics {
	if d.existingDashboardPath == "" {
		return nil
	}

	obj, err := w.Workspace.GetStatusByPath(ctx, d.existingDashboardPath)
	if err != nil {
		return diag.FromErr(err)
	}

	if obj.ObjectType != workspace.ObjectTypeDashboard {
		found := strings.ToLower(obj.ObjectType.String())
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("expected a dashboard, found a %s", found),
				Locations: []dyn.Location{{
					File: d.existingDashboardPath,
				}},
			},
		}
	}

	if obj.ResourceId == "" {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "expected resource ID to be set",
				Locations: []dyn.Location{{
					File: d.existingDashboardPath,
				}},
			},
		}
	}

	d.existingDashboardId = obj.ResourceId
	return nil
}

func (d *dashboard) saveConfiguration(ctx context.Context, dashboard *dashboards.Dashboard) error {
	// TODO: add flag
	key := d.key
	if key == "" {
		key = textutil.NormalizeString(dashboard.DisplayName)
	}

	dashboardFilePath := path.Join(d.relativeDir, fmt.Sprintf("%s.lvdash.json", key))
	v, err := generate.ConvertDashboardToValue(dashboard, dashboardFilePath)
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

	filename := filepath.Join(d.resourceDir, fmt.Sprintf("%s.yml", key))
	saver := yamlsaver.NewSaverWithStyle(map[string]yaml.Style{
		"display_name": yaml.DoubleQuotedStyle,
	})
	err = saver.SaveAsYAML(result, filename, false)
	if err != nil {
		return err
	}

	return nil
}

func (d *dashboard) remarshal(data []byte) ([]byte, error) {
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

func (d *dashboard) saveSerializedDashboard(ctx context.Context, dashboard *dashboards.Dashboard, dst string) error {
	// Unmarshal and remarshal the serialized dashboard to ensure it is formatted correctly.
	// The result will have alphabetically sorted keys and be indented.
	data, err := d.remarshal([]byte(dashboard.SerializedDashboard))
	if err != nil {
		return err
	}

	// Make sure the output directory exists.
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}

func (d *dashboard) watchForChanges(ctx context.Context, b *bundle.Bundle) error {
	diags := bundle.Apply(ctx, b, bundle.Seq(
		phases.Initialize(),
		terraform.Interpolate(),
		terraform.Write(),
		terraform.StatePull(),
		terraform.Load(terraform.ErrorOnEmptyState),
	))
	if err := diags.Error(); err != nil {
		return err
	}

	dash, ok := b.Config.Resources.Dashboards[d.watch]
	if !ok {
		return fmt.Errorf("dashboard %s not found", d.watch)
	}

	// fmt.Println(dash.DefinitionPath)

	w := b.WorkspaceClient()
	etag := ""

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(cwd, dash.FilePath)
	if err != nil {
		return err
	}

	for {
		dashboard, err := w.Lakeview.GetByDashboardId(ctx, dash.ID)
		if err != nil {
			return err
		}

		// fmt.Println(dashboard.Path)
		// fmt.Println(dashboard.Etag)
		// fmt.Println(dashboard.UpdateTime)

		// obj, err := w.Workspace.GetStatusByPath(ctx, "/Users/pieter.noordhuis@databricks.com/.bundle/dashboard-eng-work-generate/dev/files/[dev pieter_noordhuis] NYC Taxi Trip Analysis.lvdash.json")
		// if err != nil {
		// 	return err
		// }

		// fmt.Println(obj.ModifiedAt)

		if etag != dashboard.Etag {
			fmt.Printf("[%s]: Updating dashboard at %s\n", dashboard.UpdateTime, relPath)
			d.saveSerializedDashboard(ctx, dashboard, dash.FilePath)
		}

		etag = dashboard.Etag
		time.Sleep(1 * time.Second)
	}
}

func (d *dashboard) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	b, diags := root.MustConfigureBundle(cmd)
	if err := diags.Error(); err != nil {
		return diags.Error()
	}

	// Make sure we know how the dashboard path is relative to the resource path.
	rel, err := filepath.Rel(d.resourceDir, d.dashboardDir)
	if err != nil {
		return err
	}

	d.relativeDir = filepath.ToSlash(rel)

	w := b.WorkspaceClient()

	if d.watch != "" {
		return d.watchForChanges(ctx, b)
	}

	// Lookup the dashboard ID if the path is given
	diags = d.resolveDashboardID(ctx, w)
	if diags.HasError() {
		return diags.Error()
	}

	dashboard, err := w.Lakeview.GetByDashboardId(ctx, d.existingDashboardId)
	if err != nil {
		return err
	}

	d.saveConfiguration(ctx, dashboard)

	// TODO: add flag
	key := d.key
	if key == "" {
		key = textutil.NormalizeString(dashboard.DisplayName)
	}

	filename := filepath.Join(d.dashboardDir, fmt.Sprintf("%s.lvdash.json", key))
	d.saveSerializedDashboard(ctx, dashboard, filename)
	return nil
}

func NewGenerateDashboardCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Generate configuration for a dashboard",
	}

	d := &dashboard{}

	cmd.Flags().StringVarP(&d.resourceDir, "resource-dir", "d", "./resources", `directory to write the configuration to`)
	cmd.Flags().StringVarP(&d.dashboardDir, "dashboard-dir", "s", "./dashboards", `directory to write the dashboard representation to`)
	cmd.Flags().BoolVarP(&d.force, "force", "f", false, `force overwrite existing files in the output directory`)

	// Specify dashboard by workspace path

	cmd.Flags().StringVar(&d.existingDashboardPath, "existing-dashboard-path", "", `workspace path of the dashboard to generate configuration for`)
	cmd.Flags().StringVar(&d.existingDashboardId, "existing-dashboard-id", "", `ID of the dashboard to generate configuration for`)
	cmd.Flags().StringVar(&d.watch, "watch-resource", "", `resource key of dashboard to watch for changes`)

	cmd.MarkFlagsOneRequired(
		"existing-dashboard-path",
		"existing-dashboard-id",
		"watch-resource",
	)

	cmd.RunE = d.RunE
	return cmd
}

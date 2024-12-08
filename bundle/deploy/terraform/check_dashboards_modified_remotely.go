package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	tfjson "github.com/hashicorp/terraform-json"
)

type dashboardState struct {
	Name string
	ID   string
	ETag string
}

func collectDashboardsFromState(ctx context.Context, b *bundle.Bundle) ([]dashboardState, error) {
	state, err := ParseResourcesState(ctx, b)
	if err != nil && state == nil {
		return nil, err
	}

	var dashboards []dashboardState
	for _, resource := range state.Resources {
		if resource.Mode != tfjson.ManagedResourceMode {
			continue
		}
		for _, instance := range resource.Instances {
			switch resource.Type {
			case "databricks_dashboard":
				dashboards = append(dashboards, dashboardState{
					Name: resource.Name,
					ID:   instance.Attributes.ID,
					ETag: instance.Attributes.ETag,
				})
			}
		}
	}

	return dashboards, nil
}

type checkDashboardsModifiedRemotely struct {
}

func (l *checkDashboardsModifiedRemotely) Name() string {
	return "CheckDashboardsModifiedRemotely"
}

func (l *checkDashboardsModifiedRemotely) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// This mutator is relevant only if the bundle includes dashboards.
	if len(b.Config.Resources.Dashboards) == 0 {
		return nil
	}

	// If the user has forced the deployment, skip this check.
	if b.Config.Bundle.Force {
		return nil
	}

	dashboards, err := collectDashboardsFromState(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	for _, dashboard := range dashboards {
		// Skip dashboards that are not defined in the bundle.
		// These will be destroyed upon deployment.
		if _, ok := b.Config.Resources.Dashboards[dashboard.Name]; !ok {
			continue
		}

		path := dyn.MustPathFromString(fmt.Sprintf("resources.dashboards.%s", dashboard.Name))
		loc := b.Config.GetLocation(path.String())
		actual, err := b.WorkspaceClient().Lakeview.GetByDashboardId(ctx, dashboard.ID)
		if err != nil {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("failed to get dashboard %q", dashboard.Name),
				Detail:    err.Error(),
				Paths:     []dyn.Path{path},
				Locations: []dyn.Location{loc},
			})
			continue
		}

		// If the ETag is the same, the dashboard has not been modified.
		if actual.Etag == dashboard.ETag {
			continue
		}

		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("dashboard %q has been modified remotely", dashboard.Name),
			Detail: "" +
				"This dashboard has been modified remotely since the last bundle deployment.\n" +
				"These modifications are untracked and will be overwritten on deploy.\n" +
				"\n" +
				"Make sure that the local dashboard definition matches what you intend to deploy\n" +
				"before proceeding with the deployment.\n" +
				"\n" +
				"Run `databricks bundle deploy --force` to bypass this error." +
				"",
			Paths:     []dyn.Path{path},
			Locations: []dyn.Location{loc},
		})
	}

	return diags
}

func CheckDashboardsModifiedRemotely() *checkDashboardsModifiedRemotely {
	return &checkDashboardsModifiedRemotely{}
}

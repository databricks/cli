package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
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
	for resourceName, instance := range state["dashboards"] {
		dashboards = append(dashboards, dashboardState{
			Name: resourceName,
			ID:   instance.ID,
			ETag: instance.ETag,
		})
	}

	return dashboards, nil
}

type checkDashboardsModifiedRemotely struct{}

func (l *checkDashboardsModifiedRemotely) Name() string {
	return "CheckDashboardsModifiedRemotely"
}

func (l *checkDashboardsModifiedRemotely) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// This mutator is relevant only if the bundle includes dashboards.
	if len(b.Config.Resources.Dashboards) == 0 {
		return nil
	}

	if b.DirectDeployment {
		// TODO: not implemented yet
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

		path := dyn.MustPathFromString("resources.dashboards." + dashboard.Name)
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

package deploy

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structpath"
)

type dashboardState struct {
	Name string
	ID   string
	ETag string
}

func collectDashboardsFromState(ctx context.Context, b *bundle.Bundle) []dashboardState {
	state := b.DeploymentBundle.ExportState(ctx)

	var dashboards []dashboardState
	for resourceKey, instance := range state {
		// Check if this is a dashboard resource key
		if !strings.HasPrefix(resourceKey, "resources.dashboards.") {
			continue
		}
		// Extract dashboard name from "resources.dashboards.name"
		parts := strings.Split(resourceKey, ".")
		if len(parts) != 3 {
			continue
		}
		resourceName := parts[2]

		dashboards = append(dashboards, dashboardState{
			Name: resourceName,
			ID:   instance.ID,
			ETag: instance.ETag,
		})
	}

	return dashboards
}

type checkDashboardsModifiedRemotely struct {
	isPlan bool
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

	dashboards := collectDashboardsFromState(ctx, b)

	var diags diag.Diagnostics
	for _, dashboard := range dashboards {
		// Skip dashboards that are not defined in the bundle.
		// These will be destroyed upon deployment.
		if _, ok := b.Config.Resources.Dashboards[dashboard.Name]; !ok {
			continue
		}

		path := structpath.NewPath(nil, "resources", "dashboards", dashboard.Name)
		loc := b.Config.GetLocationOf(path)
		actual, err := b.WorkspaceClient(ctx).Lakeview.GetByDashboardId(ctx, dashboard.ID)
		if err != nil {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("failed to get dashboard %q", dashboard.Name),
				Detail:    err.Error(),
				Paths:     []*structpath.PathNode{path},
				Locations: []diag.Location{loc},
			})
			continue
		}

		// If the ETag is the same, the dashboard has not been modified.
		if actual.Etag == dashboard.ETag {
			continue
		}

		// Downgrade this to a warning in plan mode.
		severity := diag.Error
		if l.isPlan {
			severity = diag.Warning
		}

		diags = diags.Append(diag.Diagnostic{
			Severity: severity,
			Summary:  fmt.Sprintf("dashboard %q has been modified remotely", dashboard.Name),
			Detail: "" +
				"This dashboard has been modified remotely since the last bundle deployment.\n" +
				"These modifications are untracked and will be overwritten on deploy.\n" +
				"\n" +
				"Make sure that the local dashboard definition matches what you intend to deploy\n" +
				"before proceeding with the deployment.\n" +
				"\n" +
				"To overwrite the remote changes with your local version, use --force.\n" +
				"The remote modifications will be lost." + agent.AgentNotice(),
			Paths:     []*structpath.PathNode{path},
			Locations: []diag.Location{loc},
		})
	}

	return diags
}

func CheckDashboardsModifiedRemotely(isPlan bool) *checkDashboardsModifiedRemotely {
	return &checkDashboardsModifiedRemotely{isPlan: isPlan}
}

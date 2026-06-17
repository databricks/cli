package terraform

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
)

type dashboardState struct {
	Name string
	ID   string
	ETag string
}

func collectDashboardsFromState(ctx context.Context, b *bundle.Bundle, directDeployment bool) ([]dashboardState, error) {
	var state ExportedResourcesMap
	var err error
	if directDeployment {
		state = b.DeploymentBundle.ExportState(ctx)
	} else {
		state, err = ParseResourcesState(ctx, b)
		if err != nil {
			return nil, err
		}
	}

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

	return dashboards, nil
}

type checkDashboardsModifiedRemotely struct {
	isPlan bool
	engine engine.EngineType
}

func (l *checkDashboardsModifiedRemotely) Name() string {
	return "CheckDashboardsModifiedRemotely"
}

func (l *checkDashboardsModifiedRemotely) Apply(ctx context.Context, b *bundle.Bundle) error {
	// This mutator is relevant only if the bundle includes dashboards.
	if len(b.Config.Resources.Dashboards) == 0 {
		return nil
	}

	// If the user has forced the deployment, skip this check.
	if b.Config.Bundle.Force {
		return nil
	}

	dashboards, err := collectDashboardsFromState(ctx, b, l.engine.IsDirect())
	if err != nil {
		return err
	}

	for _, dashboard := range dashboards {
		// Skip dashboards that are not defined in the bundle.
		// These will be destroyed upon deployment.
		if _, ok := b.Config.Resources.Dashboards[dashboard.Name]; !ok {
			continue
		}

		path := dyn.MustPathFromString("resources.dashboards." + dashboard.Name)
		loc := b.Config.GetLocation(path.String())
		actual, err := b.WorkspaceClient(ctx).Lakeview.GetByDashboardId(ctx, dashboard.ID)
		if err != nil {
			return diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("failed to get dashboard %q", dashboard.Name),
				Detail:    err.Error(),
				Paths:     []dyn.Path{path},
				Locations: []dyn.Location{loc},
			}
		}

		// If the ETag is the same, the dashboard has not been modified.
		if actual.Etag == dashboard.ETag {
			continue
		}

		d := diag.Diagnostic{
			Severity: diag.Error,
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
			Paths:     []dyn.Path{path},
			Locations: []dyn.Location{loc},
		}

		// Downgrade this to a warning in plan mode, emitting it immediately;
		// in deploy mode it is a fatal error that aborts the pipeline.
		if l.isPlan {
			d.Severity = diag.Warning
			logdiag.LogDiag(ctx, d)
			continue
		}

		return d
	}

	return nil
}

func CheckDashboardsModifiedRemotely(isPlan bool, engine engine.EngineType) *checkDashboardsModifiedRemotely {
	return &checkDashboardsModifiedRemotely{isPlan: isPlan, engine: engine}
}

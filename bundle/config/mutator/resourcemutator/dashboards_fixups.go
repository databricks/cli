package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

func init() {
	// Reads (typed): b.Config.Resources.Dashboards (checks dashboard configurations)
	// Updates (typed): b.Config.Resources.Dashboards[].ParentPath (ensures /Workspace prefix is present)
	// Ensures dashboard parent paths have the required /Workspace prefix
	registerStageMutator(StageAfterDefaults, "dashboards", DashboardFixups)
}

type dashboardFixups struct{}

func DashboardFixups() bundle.Mutator {
	return &dashboardFixups{}
}

func (m *dashboardFixups) Name() string {
	return "DashboardFixups"
}

func (m *dashboardFixups) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, dashboard := range b.Config.Resources.Dashboards {
		if dashboard == nil {
			continue
		}

		dashboard.ParentPath = ensureWorkspacePrefix(dashboard.ParentPath)
	}

	return nil
}

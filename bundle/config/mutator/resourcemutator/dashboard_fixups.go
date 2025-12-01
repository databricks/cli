package resourcemutator

import (
	"context"
	"path"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type dashboardFixups struct{}

func DashboardFixups() bundle.Mutator {
	return &dashboardFixups{}
}

func (m *dashboardFixups) Name() string {
	return "DashboardFixups"
}

// The DoRead method for direct deployment adds the /Workspace prefix. Add it to the local
// configuration as well to avoid persistent recreates.
func ensureWorkspacePrefix(parentPath string) string {
	if parentPath == "/Workspace" || strings.HasPrefix(parentPath, "/Workspace/") {
		return parentPath
	}
	return path.Join("/Workspace", parentPath)
}

func (m *dashboardFixups) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, dashboard := range b.Config.Resources.Dashboards {
		if dashboard == nil {
			continue
		}

		dashboard.ParentPath = ensureWorkspacePrefix(dashboard.ParentPath)

		// // Serialized dashboard should always only be set in the [resources.DashboardConfig] struct
		// // not in the embedded [dashboards.Dashboard] struct.
		// embeddedDashboard := dashboard.Dashboard
		// if embeddedDashboard.SerializedDashboard != "" {
		// 	dashboard.DashboardConfig.SerializedDashboard = embeddedDashboard.SerializedDashboard
		// 	embeddedDashboard.SerializedDashboard = ""
		// }

		// forceSendFields := embeddedDashboard.ForceSendFields
		// for _, field := range forceSendFields {
		// 	if field == "SerializedDashboard" {
		// 		forceSendFields = append(forceSendFields, "SerializedDashboard")
		// 	}
		// }
		// embeddedDashboard.ForceSendFields = forceSendFields
		// dashboard.Dashboard = embeddedDashboard
	}

	return nil
}

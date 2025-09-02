package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

type resolveDashboard struct {
	name string
}

func (l resolveDashboard) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	// List dashboards and find the one with the given name
	// If there are multiple dashboards with the same name, return an error
	dashboards, err := w.Dashboards.ListAll(ctx, sql.ListDashboardsRequest{})
	if err != nil {
		return "", err
	}

	dashboardMap := make(map[string][]sql.Dashboard)
	for _, dashboard := range dashboards {
		dashboardMap[dashboard.Name] = append(dashboardMap[dashboard.Name], dashboard)
	}

	alternatives, ok := dashboardMap[l.name]
	if !ok || len(alternatives) == 0 {
		return "", fmt.Errorf("dashboard name '%s' does not exist", l.name)
	}
	if len(alternatives) > 1 {
		return "", fmt.Errorf("there are %d instances of dashboards named '%s'", len(alternatives), l.name)
	}
	return alternatives[0].Id, nil
}

func (l resolveDashboard) String() string {
	return "dashboard: " + l.name
}

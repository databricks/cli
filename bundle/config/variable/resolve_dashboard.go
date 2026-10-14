package variable

import (
	"context"
	"fmt"
	"net/http"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/sdkclients"
)

type resolveDashboard struct {
	name string
}

type legacyDashboard struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type listLegacyDashboardsResponse struct {
	Page    int               `json:"page"`
	Results []legacyDashboard `json:"results"`
}

// Lookups by dashboard name refer to legacy SQL dashboards, which have no
// sdk-go client, so the paginated list endpoint is called directly.
func (l resolveDashboard) Resolve(ctx context.Context, w *sdkclients.Clients) (string, error) {
	api, err := apiclient.New(w.Config)
	if err != nil {
		return "", err
	}

	// List dashboards and find the one with the given name
	// If there are multiple dashboards with the same name, return an error
	var dashboards []legacyDashboard
	for page := 1; ; {
		var resp listLegacyDashboardsResponse
		err := api.Do(ctx, http.MethodGet, "/api/2.0/preview/sql/dashboards", auth.WorkspaceIDHeaders(w.Config), map[string]any{"page": page}, nil, &resp)
		if err != nil {
			return "", err
		}
		if len(resp.Results) == 0 {
			break
		}
		dashboards = append(dashboards, resp.Results...)
		page = resp.Page + 1
	}

	dashboardMap := make(map[string][]legacyDashboard)
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

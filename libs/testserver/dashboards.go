package testserver

import (
	"encoding/json"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/google/uuid"
)

func (s *FakeWorkspace) DashboardCreate(req Request) Response {
	defer s.LockUnlock()()

	var dashboard dashboards.Dashboard
	if err := json.Unmarshal(req.Body, &dashboard); err != nil {
		return Response{
			StatusCode: 400,
		}
	}

	// Lakeview API strips hyphens from a uuid for dashboards
	dashboard.DashboardId = strings.ReplaceAll(uuid.New().String(), "-", "")

	// All dashboards are active by default:
	dashboard.LifecycleState = dashboards.LifecycleStateActive

	// Change path field if parent_path is provided
	if dashboard.ParentPath != "" {
		dashboard.Path = dashboard.ParentPath + "/" + dashboard.DisplayName + ".lvdash.json"
	}

	// Parse serializedDashboard into json and put it back as a string
	if dashboard.SerializedDashboard != "" {
		var dashboardContent map[string]any
		if err := json.Unmarshal([]byte(dashboard.SerializedDashboard), &dashboardContent); err == nil {
			// Add pageType to each page in the pages array (as of June 2025, this is an undocumented Lakeview API behaviour)
			if pages, ok := dashboardContent["pages"].([]any); ok {
				for _, page := range pages {
					if pageMap, ok := page.(map[string]any); ok {
						pageMap["pageType"] = "PAGE_TYPE_CANVAS"
					}
				}
			}
			if updatedContent, err := json.Marshal(dashboardContent); err == nil {
				dashboard.SerializedDashboard = string(updatedContent)
			}
		}
	}

	s.Dashboards[dashboard.DashboardId] = dashboard
	s.files[dashboard.Path] = FileEntry{
		Info: workspace.ObjectInfo{
			ObjectType: "DASHBOARD",
			Path:       dashboard.Path,
		},
		Data: []byte(dashboard.SerializedDashboard),
	}

	return Response{
		Body: dashboards.Dashboard{
			DashboardId: dashboard.DashboardId,
			Etag:        uuid.New().String(),
		},
	}
}

func (s *FakeWorkspace) DashboardUpdate(req Request) Response {
	defer s.LockUnlock()()

	var dashboard dashboards.Dashboard
	if err := json.Unmarshal(req.Body, &dashboard); err != nil {
		return Response{
			StatusCode: 400,
		}
	}

	// Update the etag for the dashboard.
	dashboard.Etag = uuid.New().String()

	// All dashboards are active by default:
	dashboard.LifecycleState = dashboards.LifecycleStateActive

	dashboardId := req.Vars["dashboard_id"]
	s.Dashboards[dashboardId] = dashboard

	return Response{
		Body: dashboard,
	}
}

func (s *FakeWorkspace) DashboardPublish(req Request) Response {
	defer s.LockUnlock()()

	var dashboard dashboards.Dashboard
	if err := json.Unmarshal(req.Body, &dashboard); err != nil {
		return Response{
			StatusCode: 400,
		}
	}

	return Response{
		Body: dashboards.PublishedDashboard{
			WarehouseId: dashboard.WarehouseId,
		},
	}
}

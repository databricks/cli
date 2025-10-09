package testserver

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func (s *FakeWorkspace) DashboardCreate(req Request) Response {
	defer s.LockUnlock()()

	var dashboard dashboards.Dashboard
	if err := json.Unmarshal(req.Body, &dashboard); err != nil {
		return Response{
			StatusCode: 400,
		}
	}

	if _, ok := s.directories[dashboard.ParentPath]; !ok {
		return Response{
			StatusCode: 404,
			Body: map[string]string{
				"message": fmt.Sprintf("Path (%s) doesn't exist.", dashboard.ParentPath),
			},
		}
	}

	// Use nextID() to generate deterministic dashboard IDs for testing
	dashboard.DashboardId = strconv.FormatInt(nextID(), 10)

	// All dashboards are active by default:
	dashboard.LifecycleState = dashboards.LifecycleStateActive

	// Remove /Workspace prefix from parent_path. This matches the remote behavior.
	if strings.HasPrefix(dashboard.ParentPath, "/Workspace/") {
		dashboard.ParentPath = strings.TrimPrefix(dashboard.ParentPath, "/Workspace")
	}

	// Change path field if parent_path is provided
	if dashboard.ParentPath != "" {
		dashboard.Path = dashboard.ParentPath + "/" + dashboard.DisplayName + ".lvdash.json"
	}

	dashboard.CreateTime = strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339), "Z")
	dashboard.UpdateTime = dashboard.CreateTime

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
				dashboard.SerializedDashboard = string(updatedContent) + "\n"
			}
		}
	}
	dashboard.Etag = "80611980"

	s.Dashboards[dashboard.DashboardId] = dashboard
	workspacePath := path.Join("/Workspace", dashboard.Path)
	s.files[workspacePath] = FileEntry{
		Info: workspace.ObjectInfo{
			ObjectType: "DASHBOARD",
			// Include the /Workspace prefix for workspace get-status API.
			Path:       workspacePath,
			ResourceId: dashboard.DashboardId,
		},
		Data: []byte(dashboard.SerializedDashboard),
	}

	return Response{
		Body: dashboard,
	}
}

func (s *FakeWorkspace) DashboardUpdate(req Request) Response {
	defer s.LockUnlock()()

	var updateReq dashboards.Dashboard
	if err := json.Unmarshal(req.Body, &updateReq); err != nil {
		return Response{
			StatusCode: 400,
		}
	}

	dashboardId := req.Vars["dashboard_id"]
	dashboard, ok := s.Dashboards[dashboardId]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	// Update etag.
	prevEtag, err := strconv.Atoi(dashboard.Etag)
	if err != nil {
		return Response{
			Body: map[string]string{
				"message": "Invalid etag: " + dashboard.Etag,
			},
			StatusCode: 400,
		}
	}
	nextEtag := prevEtag + 1
	dashboard.Etag = strconv.Itoa(nextEtag)

	// Update the dashboard.
	dashboard.LifecycleState = dashboards.LifecycleStateActive
	if updateReq.DisplayName != "" {
		dashboard.DisplayName = updateReq.DisplayName
		dir := filepath.Dir(dashboard.Path)
		base := updateReq.DisplayName + ".lvdash.json"
		dashboard.Path = filepath.Join(dir, base)
	}
	if updateReq.SerializedDashboard != "" {
		dashboard.SerializedDashboard = updateReq.SerializedDashboard
	}
	if updateReq.WarehouseId != "" {
		dashboard.WarehouseId = updateReq.WarehouseId
	}
	dashboard.UpdateTime = time.Now().UTC().Format(time.RFC3339)

	s.Dashboards[dashboardId] = dashboard

	return Response{
		Body: dashboard,
	}
}

func (s *FakeWorkspace) DashboardPublish(req Request) Response {
	defer s.LockUnlock()()

	var publishReq dashboards.PublishRequest
	if err := json.Unmarshal(req.Body, &publishReq); err != nil {
		return Response{
			StatusCode: 400,
		}
	}

	dashboardId := req.Vars["dashboard_id"]
	dashboard, ok := s.Dashboards[dashboardId]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	publishedDashboard := dashboards.PublishedDashboard{
		WarehouseId:      dashboard.WarehouseId,
		DisplayName:      dashboard.DisplayName,
		EmbedCredentials: publishReq.EmbedCredentials,
	}

	if publishReq.WarehouseId != "" {
		publishedDashboard.WarehouseId = publishReq.WarehouseId
	}
	if publishReq.EmbedCredentials {
		publishedDashboard.EmbedCredentials = publishReq.EmbedCredentials
	}

	s.PublishedDashboards[dashboardId] = publishedDashboard

	return Response{
		Body: dashboards.PublishedDashboard{
			WarehouseId:      publishedDashboard.WarehouseId,
			DisplayName:      publishedDashboard.DisplayName,
			EmbedCredentials: publishedDashboard.EmbedCredentials,
		},
	}
}

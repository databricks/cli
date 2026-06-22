package testserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// Generate 32 character hex string for dashboard ID
func generateDashboardId() (string, error) {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(randomBytes), nil
}

// Transform the serialized dashboard to mimic remote behavior.
func transformSerializedDashboard(serializedDashboard, datasetCatalog, datasetSchema string) string {
	var dashboardContent map[string]any
	err := json.Unmarshal([]byte(serializedDashboard), &dashboardContent)
	if err != nil {
		return serializedDashboard
	}

	// The real backend stores the serialized dashboard verbatim unless it
	// modifies the content (the mutations below), in which case it re-serializes
	// it. The one exception is an empty object, which it canonicalizes from
	// "{ }" to "{}" (verified against cloud). Track whether we mutate so we can
	// pick the right behavior.
	mutated := false

	// Add pageType to each page in the pages array (as of June 2025, this is an undocumented Lakeview API behaviour)
	if pages, ok := dashboardContent["pages"].([]any); ok {
		for _, page := range pages {
			if pageMap, ok := page.(map[string]any); ok {
				pageMap["pageType"] = "PAGE_TYPE_CANVAS"
				mutated = true
			}
		}
	}

	// Apply dataset_catalog and dataset_schema overrides to all datasets
	if datasets, ok := dashboardContent["datasets"].([]any); ok {
		for _, dataset := range datasets {
			if datasetMap, ok := dataset.(map[string]any); ok {
				if datasetCatalog != "" {
					datasetMap["catalog"] = datasetCatalog
					mutated = true
				}
				if datasetSchema != "" {
					datasetMap["schema"] = datasetSchema
					mutated = true
				}
			}
		}
	}

	// Re-marshal when we mutated the parsed content; canonicalize an empty
	// object to "{}"; otherwise preserve the caller's input verbatim.
	var result string
	switch {
	case mutated:
		updatedContent, err := json.Marshal(dashboardContent)
		if err != nil {
			return serializedDashboard
		}
		result = string(updatedContent)
	case len(dashboardContent) == 0:
		result = "{}"
	default:
		result = serializedDashboard
	}

	// Cloud always terminates the stored dashboard with a single trailing newline.
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

func (s *FakeWorkspace) DashboardCreate(req Request) Response {
	defer s.LockUnlock()()

	var dashboard dashboards.Dashboard
	if err := json.Unmarshal(req.Body, &dashboard); err != nil {
		return Response{
			StatusCode: 400,
		}
	}

	// Default to user's home directory if parent_path is not provided (matches cloud behavior)
	if dashboard.ParentPath == "" {
		dashboard.ParentPath = "/Users/" + s.CurrentUser().UserName
	}

	if _, ok := s.directories[dashboard.ParentPath]; !ok {
		return Response{
			StatusCode: 404,
			Body: map[string]string{
				"message": fmt.Sprintf("Path (%s) doesn't exist.", dashboard.ParentPath),
			},
		}
	}

	var err error
	dashboard.DashboardId, err = generateDashboardId()
	if err != nil {
		return Response{
			StatusCode: 500,
			Body: map[string]string{
				"message": "Failed to generate dashboard ID",
			},
		}
	}

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

	inputSerializedDashboard := dashboard.SerializedDashboard

	// Extract dataset_catalog and dataset_schema from query parameters
	datasetCatalog := req.URL.Query().Get("dataset_catalog")
	datasetSchema := req.URL.Query().Get("dataset_schema")

	// Parse serializedDashboard into json and put it back as a string
	if dashboard.SerializedDashboard != "" {
		dashboard.SerializedDashboard = transformSerializedDashboard(dashboard.SerializedDashboard, datasetCatalog, datasetSchema)
	}
	dashboard.Etag = "80611980"

	s.Dashboards[dashboard.DashboardId] = fakeDashboard{
		Dashboard:                dashboard,
		InputSerializedDashboard: inputSerializedDashboard,
	}

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

	// Bump etag on every write, matching cloud behavior.
	prevEtag, err := strconv.Atoi(dashboard.Etag)
	if err != nil {
		return Response{
			Body: map[string]string{
				"message": "Invalid etag: " + dashboard.Etag,
			},
			StatusCode: 400,
		}
	}
	dashboard.Etag = strconv.Itoa(prevEtag + 1)

	if updateReq.SerializedDashboard != dashboard.InputSerializedDashboard {
		dashboard.InputSerializedDashboard = updateReq.SerializedDashboard
	}

	// Update the dashboard.
	dashboard.LifecycleState = dashboards.LifecycleStateActive
	if updateReq.DisplayName != "" {
		dashboard.DisplayName = updateReq.DisplayName
		dir := path.Dir(dashboard.Path)
		base := updateReq.DisplayName + ".lvdash.json"
		dashboard.Path = path.Join(dir, base)
	}
	if updateReq.SerializedDashboard != "" {
		// Extract dataset_catalog and dataset_schema from query parameters
		datasetCatalog := req.URL.Query().Get("dataset_catalog")
		datasetSchema := req.URL.Query().Get("dataset_schema")
		dashboard.SerializedDashboard = transformSerializedDashboard(updateReq.SerializedDashboard, datasetCatalog, datasetSchema)
	}
	dashboard.WarehouseId = updateReq.WarehouseId
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
		WarehouseId:        dashboard.WarehouseId,
		DisplayName:        dashboard.DisplayName,
		EmbedCredentials:   publishReq.EmbedCredentials,
		RevisionCreateTime: time.Now().UTC().Format(time.RFC3339),
		ForceSendFields:    []string{"EmbedCredentials"},
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
			WarehouseId:        publishedDashboard.WarehouseId,
			DisplayName:        publishedDashboard.DisplayName,
			EmbedCredentials:   publishedDashboard.EmbedCredentials,
			RevisionCreateTime: publishedDashboard.RevisionCreateTime,
			ForceSendFields:    []string{"EmbedCredentials"},
		},
	}
}

func (s *FakeWorkspace) DashboardTrash(req Request) Response {
	defer s.LockUnlock()()

	dashboardId := req.Vars["dashboard_id"]
	dashboard, ok := s.Dashboards[dashboardId]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	// The dashboard is marked as trashed and moved to the trash.
	s.Dashboards[dashboardId] = fakeDashboard{
		Dashboard: dashboards.Dashboard{
			Etag:           dashboard.Etag,
			DashboardId:    dashboardId,
			LifecycleState: dashboards.LifecycleStateTrashed,
			ParentPath:     path.Join("/Users", s.CurrentUser().UserName, "Trash"),
		},
	}

	// The published dashboard is deleted.
	delete(s.PublishedDashboards, dashboardId)

	return Response{
		Body: dashboard,
	}
}

func (s *FakeWorkspace) DashboardUnpublish(req Request) Response {
	defer s.LockUnlock()()

	dashboardId := req.Vars["dashboard_id"]
	_, ok := s.Dashboards[dashboardId]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	// Delete the published dashboard entry.
	delete(s.PublishedDashboards, dashboardId)

	return Response{
		Body: "",
	}
}

func (s *FakeWorkspace) DashboardGetPublished(req Request) Response {
	defer s.LockUnlock()()

	dashboardId := req.Vars["dashboard_id"]
	publishedDashboard, ok := s.PublishedDashboards[dashboardId]
	if !ok {
		return Response{
			StatusCode: 404,
			Body: map[string]string{
				"message": fmt.Sprintf("Unable to find published dashboard [%s]", dashboardId),
			},
		}
	}

	return Response{
		Body: publishedDashboard,
	}
}

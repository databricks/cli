package testserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

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

	// Track whether we modify the parsed content below.
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

	// Cloud returns the serialized dashboard verbatim, except it re-serializes
	// the content when it injects the fields above and canonicalizes an empty
	// object to "{}" (both verified against cloud). Marshaling the parsed content
	// covers both: a mutated object re-serializes and an empty object yields "{}".
	result := serializedDashboard
	if mutated || len(dashboardContent) == 0 {
		updatedContent, err := json.Marshal(dashboardContent)
		if err != nil {
			return serializedDashboard
		}
		result = string(updatedContent)
	}

	// Cloud always terminates the stored dashboard with a single trailing newline.
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

func (s *FakeWorkspace) DashboardGet(req Request) Response {
	defer s.LockUnlock()()

	dashboardId := req.Vars["dashboard_id"]
	// Read applies eventual consistency: the first GET after a create returns nil
	// (404) to simulate propagation delay. Updates are immediately visible.
	ptr, ok := s.Dashboards.Read(dashboardId)
	if !ok || ptr == nil {
		return Response{StatusCode: 404}
	}
	return Response{Body: *ptr}
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

	workspacePath := path.Join("/Workspace", dashboard.Path)
	if _, exists := s.files[workspacePath]; exists {
		return Response{
			StatusCode: http.StatusConflict,
			Body: map[string]string{
				"error_code": "ALREADY_EXISTS",
				"message":    fmt.Sprintf("A node with name '%s' already exists under the specified parent", dashboard.DisplayName),
			},
		}
	}

	dashboard.CreateTime = nextTimestamp()
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

	// Write so that, when eventual consistency is enabled, the first GET after this
	// create returns 404.
	s.Dashboards.Write(dashboard.DashboardId, &fakeDashboard{
		Dashboard:                dashboard,
		InputSerializedDashboard: inputSerializedDashboard,
	})

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
	dashboard, ok := s.Dashboards.ReadStrong(dashboardId)
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}
	updated := *dashboard
	if resp, ok := updated.bumpEtag(); !ok {
		return resp
	}

	if updateReq.SerializedDashboard != updated.InputSerializedDashboard {
		updated.InputSerializedDashboard = updateReq.SerializedDashboard
	}

	// Update the dashboard.
	updated.LifecycleState = dashboards.LifecycleStateActive
	if updateReq.DisplayName != "" {
		updated.DisplayName = updateReq.DisplayName
		dir := path.Dir(updated.Path)
		base := updateReq.DisplayName + ".lvdash.json"
		s.moveDashboardFile(updated.Path, path.Join(dir, base), dashboardId)
		updated.Path = path.Join(dir, base)
	}
	if updateReq.SerializedDashboard != "" {
		// Extract dataset_catalog and dataset_schema from query parameters
		datasetCatalog := req.URL.Query().Get("dataset_catalog")
		datasetSchema := req.URL.Query().Get("dataset_schema")
		updated.SerializedDashboard = transformSerializedDashboard(updateReq.SerializedDashboard, datasetCatalog, datasetSchema)
	}
	updated.WarehouseId = updateReq.WarehouseId
	updated.UpdateTime = nextTimestamp()

	// Write stages a stale value: like a real backend, the first read after an update
	// returns the pre-update value. Tests that read right after an update wait for the
	// new value with a content-aware retry (retry --until / --until-not).
	s.Dashboards.Write(dashboardId, &updated)

	return Response{
		Body: updated,
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
	dashboard, ok := s.Dashboards.ReadStrong(dashboardId)
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	publishedDashboard := dashboards.PublishedDashboard{
		WarehouseId:        dashboard.WarehouseId,
		DisplayName:        dashboard.DisplayName,
		EmbedCredentials:   publishReq.EmbedCredentials,
		RevisionCreateTime: nextTimestamp(),
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
	dashboard, ok := s.Dashboards.ReadStrong(dashboardId)
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	// The dashboard is marked as trashed and moved to the trash, which frees its path.
	s.moveDashboardFile(dashboard.Path, "", dashboardId)
	s.Dashboards.Put(dashboardId, &fakeDashboard{
		Etag:           dashboard.Etag,
		DashboardId:    dashboardId,
		LifecycleState: dashboards.LifecycleStateTrashed,
		ParentPath:     path.Join("/Users", s.CurrentUser().UserName, "Trash"),
	})

	// The published dashboard is deleted.
	delete(s.PublishedDashboards, dashboardId)

	return Response{
		Body: *dashboard,
	}
}

func (s *FakeWorkspace) DashboardUnpublish(req Request) Response {
	defer s.LockUnlock()()

	dashboardId := req.Vars["dashboard_id"]
	if _, ok := s.Dashboards.ReadStrong(dashboardId); !ok {
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

// moveDashboardFile moves the workspace file of dashboard dashboardId from oldPath to newPath, or
// removes it if newPath is empty. Paths are as in Dashboard.Path, without the /Workspace prefix.
func (s *FakeWorkspace) moveDashboardFile(oldPath, newPath, dashboardId string) {
	oldWorkspacePath := path.Join("/Workspace", oldPath)
	entry, ok := s.files[oldWorkspacePath]
	if !ok || entry.Info.ResourceId != dashboardId {
		return
	}
	delete(s.files, oldWorkspacePath)
	if newPath != "" {
		entry.Info.Path = path.Join("/Workspace", newPath)
		s.files[entry.Info.Path] = entry
	}
}

// dashboardEntry finds the workspace object of a dashboard by the path Lakeview returns for it
// (Dashboard.Path, without the /Workspace prefix), which the real Workspace API accepts as well.
// Only dashboards are resolved this way. The caller must hold the lock.
func (s *FakeWorkspace) dashboardEntry(filePath string) (FileEntry, bool) {
	if strings.HasPrefix(filePath, "/Workspace/") {
		return FileEntry{}, false
	}
	entry, ok := s.files[path.Join("/Workspace", filePath)]
	if !ok || entry.Info.ObjectType != workspace.ObjectTypeDashboard {
		return FileEntry{}, false
	}
	return entry, true
}

// bumpEtag advances the etag, as the cloud does on every write. ok is false, with the response to
// return, when the stored etag is not a number.
func (d *fakeDashboard) bumpEtag() (Response, bool) {
	prevEtag, err := strconv.Atoi(d.Etag)
	if err != nil {
		return Response{
			Body: map[string]string{
				"message": "Invalid etag: " + d.Etag,
			},
			StatusCode: 400,
		}, false
	}
	d.Etag = strconv.Itoa(prevEtag + 1)
	return Response{}, true
}

// WorkspaceImportOverDashboard serves an AUTO-format /workspace/import with overwrite onto the
// .lvdash.json path of an existing dashboard. The real API updates that dashboard in place: the
// path keeps the dashboard's identity (get-status resource_id), and the import sets its
// serialized_dashboard, etag and update_time. ok is false when the request is not such an import.
func (s *FakeWorkspace) WorkspaceImportOverDashboard(filePath string, body []byte, overwrite bool) (Response, bool) {
	if !overwrite {
		return Response{}, false
	}
	filePath = path.Clean("/" + filePath)

	defer s.LockUnlock()()

	entry, ok := s.files[filePath]
	if !ok {
		entry, ok = s.dashboardEntry(filePath)
		filePath = entry.Info.Path
	}
	if !ok || entry.Info.ObjectType != workspace.ObjectTypeDashboard {
		return Response{}, false
	}
	dashboard, ok := s.Dashboards.ReadStrong(entry.Info.ResourceId)
	if !ok {
		return Response{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("internal error: no dashboard %s for workspace object %s", entry.Info.ResourceId, filePath),
		}, true
	}
	updated := *dashboard
	if resp, ok := updated.bumpEtag(); !ok {
		return resp, true
	}
	updated.InputSerializedDashboard = string(body)
	updated.SerializedDashboard = transformSerializedDashboard(string(body), "", "")
	updated.UpdateTime = nextTimestamp()
	s.Dashboards.Write(updated.DashboardId, &updated)

	entry.Data = []byte(updated.SerializedDashboard)
	s.files[filePath] = entry
	return Response{}, true
}

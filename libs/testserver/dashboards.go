package testserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	lakeview "github.com/databricks/cli/libs/sdktypes/lakeview/v1"
	"github.com/databricks/cli/libs/utils"
	workspaceapi "github.com/databricks/cli/libs/workspaceapi"
	"path"
	"strconv"
	"strings"
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

	var dashboard lakeview.Dashboard
	if err := json.Unmarshal(req.Body, &dashboard); err != nil {
		return Response{
			StatusCode: 400,
		}
	}

	// Default to user's home directory if parent_path is not provided (matches cloud behavior)
	if utils.Deref(dashboard.ParentPath) == "" {
		dashboard.ParentPath = utils.NonZero("/Users/" + utils.Deref(s.CurrentUser().UserName))
	}

	if _, ok := s.directories[utils.Deref(dashboard.ParentPath)]; !ok {
		return Response{
			StatusCode: 404,
			Body: map[string]string{
				"message": fmt.Sprintf("Path (%s) doesn't exist.", utils.Deref(dashboard.ParentPath)),
			},
		}
	}

	var err error
	var dashboardId string
	dashboardId, err = generateDashboardId()
	dashboard.DashboardId = utils.NonZero(dashboardId)
	if err != nil {
		return Response{
			StatusCode: 500,
			Body: map[string]string{
				"message": "Failed to generate dashboard ID",
			},
		}
	}

	// All dashboards are active by default:
	dashboard.LifecycleState = lakeview.LifecycleState_Active

	// Remove /Workspace prefix from parent_path. This matches the remote behavior.
	if strings.HasPrefix(utils.Deref(dashboard.ParentPath), "/Workspace/") {
		dashboard.ParentPath = utils.NonZero(strings.TrimPrefix(utils.Deref(dashboard.ParentPath), "/Workspace"))
	}

	// Change path field if parent_path is provided
	if utils.Deref(dashboard.ParentPath) != "" {
		dashboard.Path = utils.NonZero(utils.Deref(dashboard.ParentPath) + "/" + utils.Deref(dashboard.DisplayName) + ".lvdash.json")
	}

	dashboard.CreateTime = nextTime()
	dashboard.UpdateTime = dashboard.CreateTime

	inputSerializedDashboard := dashboard.SerializedDashboard

	// Extract dataset_catalog and dataset_schema from query parameters
	datasetCatalog := req.URL.Query().Get("dataset_catalog")
	datasetSchema := req.URL.Query().Get("dataset_schema")

	// Parse serializedDashboard into json and put it back as a string
	if utils.Deref(dashboard.SerializedDashboard) != "" {
		dashboard.SerializedDashboard = utils.NonZero(transformSerializedDashboard(utils.Deref(dashboard.SerializedDashboard), datasetCatalog, datasetSchema))
	}
	dashboard.Etag = utils.NonZero("80611980")

	// Write so that, when eventual consistency is enabled, the first GET after this
	// create returns 404.
	s.Dashboards.Write(utils.Deref(dashboard.DashboardId), &fakeDashboard{
		Dashboard:                dashboard,
		InputSerializedDashboard: utils.Deref(inputSerializedDashboard),
	})

	workspacePath := path.Join("/Workspace", utils.Deref(dashboard.Path))
	s.files[workspacePath] = FileEntry{
		Info: workspaceapi.ObjectInfo{
			ObjectType: "DASHBOARD",
			// Include the /Workspace prefix for workspace get-status API.
			Path:       workspacePath,
			ResourceId: utils.Deref(dashboard.DashboardId),
		},
		Data: []byte(utils.Deref(dashboard.SerializedDashboard)),
	}

	return Response{
		Body: dashboard,
	}
}

func (s *FakeWorkspace) DashboardUpdate(req Request) Response {
	defer s.LockUnlock()()

	var updateReq lakeview.Dashboard
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

	// Bump etag on every write, matching cloud behavior.
	prevEtag, err := strconv.Atoi(utils.Deref(updated.Etag))
	if err != nil {
		return Response{
			Body: map[string]string{
				"message": "Invalid etag: " + utils.Deref(updated.Etag),
			},
			StatusCode: 400,
		}
	}
	updated.Etag = utils.NonZero(strconv.Itoa(prevEtag + 1))

	if utils.Deref(updateReq.SerializedDashboard) != updated.InputSerializedDashboard {
		updated.InputSerializedDashboard = utils.Deref(updateReq.SerializedDashboard)
	}

	// Update the dashboard.
	updated.LifecycleState = lakeview.LifecycleState_Active
	if utils.Deref(updateReq.DisplayName) != "" {
		updated.DisplayName = updateReq.DisplayName
		dir := path.Dir(utils.Deref(updated.Path))
		base := utils.Deref(updateReq.DisplayName) + ".lvdash.json"
		updated.Path = utils.NonZero(path.Join(dir, base))
	}
	if utils.Deref(updateReq.SerializedDashboard) != "" {
		// Extract dataset_catalog and dataset_schema from query parameters
		datasetCatalog := req.URL.Query().Get("dataset_catalog")
		datasetSchema := req.URL.Query().Get("dataset_schema")
		updated.SerializedDashboard = utils.NonZero(transformSerializedDashboard(utils.Deref(updateReq.SerializedDashboard), datasetCatalog, datasetSchema))
	}
	updated.WarehouseId = updateReq.WarehouseId
	updated.UpdateTime = nextTime()

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

	var publishReq lakeview.PublishDashboardRequest
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

	publishedDashboard := lakeview.PublishedDashboard{
		WarehouseId:        dashboard.WarehouseId,
		DisplayName:        dashboard.DisplayName,
		EmbedCredentials:   publishReq.EmbedCredentials,
		RevisionCreateTime: nextTime(),
	}

	if utils.Deref(publishReq.WarehouseId) != "" {
		publishedDashboard.WarehouseId = publishReq.WarehouseId
	}
	// embed_credentials is always reported, false when the request omitted it.
	publishedDashboard.EmbedCredentials = new(utils.Deref(publishReq.EmbedCredentials))

	s.PublishedDashboards[dashboardId] = publishedDashboard

	return Response{
		Body: lakeview.PublishedDashboard{
			WarehouseId:        publishedDashboard.WarehouseId,
			DisplayName:        publishedDashboard.DisplayName,
			EmbedCredentials:   publishedDashboard.EmbedCredentials,
			RevisionCreateTime: publishedDashboard.RevisionCreateTime,
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

	// The dashboard is marked as trashed and moved to the trash.
	s.Dashboards.Put(dashboardId, &fakeDashboard{
		Etag:           dashboard.Etag,
		DashboardId:    utils.NonZero(dashboardId),
		LifecycleState: lakeview.LifecycleState_Trashed,
		ParentPath:     utils.NonZero(path.Join("/Users", utils.Deref(s.CurrentUser().UserName), "Trash")),
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

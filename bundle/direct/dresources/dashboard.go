package dresources

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/cli/libs/apierrs"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/lakeview/v1"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/core/types"
	sdklakeview "github.com/databricks/sdk-go/lakeview/v1"
	"golang.org/x/sync/errgroup"
)

// Terraform implementation reference: https://github.com/databricks/terraform-provider-databricks/blob/main/dashboards/resource_dashboard.go
type ResourceDashboard struct {
	client *sdkclients.Clients
}

type DashboardState struct {
	resources.DashboardConfig
	// Published reports whether the current draft content is published. DoRead sets it
	// to true only when GetPublished succeeds AND the published revision is at least as
	// new as the draft (revision_create_time >= update_time). A draft update without a
	// republish leaves the previously-published content in place, but the API keeps
	// returning it from GetPublished, so a naive "does GetPublished succeed" cannot
	// detect stale content; the timestamp comparison can.
	//
	// The desired value (PrepareState) is always true, so when the published content is
	// stale, remote Published=false differs from desired and the planner runs DoUpdate,
	// which republishes.
	Published bool `json:"published"`
}

// ensureWorkspacePrefix adds the /Workspace prefix to the parent path if it's not already present.
// The backend removes this prefix from parent path, and thus it needs to be added back
// to match the local configuration. The default parent_path (i.e. ${workspace.resource_path})
// includes the /Workspace prefix, that's why we need to add it back here.
//
// If in the future `parent_path` from GET includes the /Workspace prefix, this logic
// will still be correct because creating "/Workspace/Workspace" is not allowed.
func ensureWorkspacePrefix(parentPath string) string {
	if parentPath == "/Workspace" || strings.HasPrefix(parentPath, "/Workspace/") {
		return parentPath
	}
	return path.Join("/Workspace", parentPath)
}

func (*ResourceDashboard) New(client *sdkclients.Clients) *ResourceDashboard {
	return &ResourceDashboard{client: client}
}

func (*ResourceDashboard) PrepareState(input *resources.Dashboard) *DashboardState {
	return &DashboardState{
		DashboardConfig: input.DashboardConfig,
		Published:       true,
	}
}

func (r *ResourceDashboard) DoRead(ctx context.Context, id string) (*DashboardState, error) {
	var dashboardResp *sdklakeview.Dashboard
	var publishedResp *sdklakeview.PublishedDashboard
	var publishedErr error

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		dashboardResp, err = r.client.Lakeview.GetDashboard(ctx, sdklakeview.GetDashboardRequest{
			DashboardId: new(id),
		})
		return err
	})

	g.Go(func() error {
		var err error
		publishedResp, err = r.client.Lakeview.GetPublishedDashboard(ctx, sdklakeview.GetPublishedDashboardRequest{
			DashboardId: new(id),
		})
		// Store error but don't fail - dashboard might be unpublished
		publishedErr = err
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	dashboard, err := lakeview.DashboardFromSDK(dashboardResp)
	if err != nil {
		return nil, err
	}
	var publishedDashboard *lakeview.PublishedDashboard
	if publishedErr == nil {
		publishedDashboard, err = lakeview.PublishedDashboardFromSDK(publishedResp)
		if err != nil {
			return nil, err
		}
	}

	// A trashed dashboard is considered deleted - return not found error
	if dashboard.LifecycleState == lakeview.LifecycleState_Trashed {
		return nil, errResourceDoesNotExist("dashboard " + id + " is trashed")
	}

	// A dashboard is published (in the sense that matters for planning: the current
	// draft content is what's published) only when GetPublished succeeds AND the
	// published revision is at least as new as the draft. A draft update without a
	// republish leaves stale content published, but GetPublished keeps returning it,
	// so publishedErr == nil alone cannot detect staleness — the timestamp comparison
	// can. If either timestamp is missing, fall back to publish-existence so we
	// don't spuriously republish. See DashboardState.Published.
	published := publishedErr == nil && publishedIsCurrent(dashboard, publishedDashboard)

	state := dashboardToState(dashboard, publishedDashboard, utils.Deref(dashboard.SerializedDashboard), published)
	return state, nil
}

// publishedIsCurrent reports whether the published revision reflects the current
// draft, i.e. the dashboard was published after its last draft update. The published
// API exposes no etag, only revision_create_time, so we compare it against the draft's
// update_time. Both are set by the backend on a shared clock (verified on
// AWS/GCP/Azure/dogfood: a publish always leaves revision_create_time >= update_time,
// and a draft update pushes update_time ahead until the next publish). If either
// timestamp is missing, return true so we do not force a spurious republish.
func publishedIsCurrent(dashboard *lakeview.Dashboard, published *lakeview.PublishedDashboard) bool {
	if published == nil {
		return false
	}
	if dashboard.UpdateTime == nil || published.RevisionCreateTime == nil {
		return true
	}
	return !published.RevisionCreateTime.AsTime().Before(dashboard.UpdateTime.AsTime())
}

func prepareDashboardRequest(config *DashboardState) (*lakeview.Dashboard, error) {
	// Fields like "embed_credentials" are part of the bundle configuration but not the
	// create/update request here.
	dashboard := &lakeview.Dashboard{
		DisplayName:         nonEmpty(config.DisplayName),
		ParentPath:          nonEmpty(config.ParentPath),
		WarehouseId:         nonEmpty(config.WarehouseId),
		Etag:                nonEmpty(config.Etag),
		CreateTime:          nil, // Not set by DABs
		DashboardId:         nil, // Not set by DABs
		LifecycleState:      "",  // Not set by DABs
		Path:                nil, // Not set by DABs
		SerializedDashboard: nil, // Not set by DABs
		UpdateTime:          nil, // Not set by DABs
	}
	switch v := config.SerializedDashboard.(type) {
	case nil:
	case string:
		dashboard.SerializedDashboard = nonEmpty(v)
	default:
		return nil, fmt.Errorf("internal error: serialized_dashboard should have been normalized to a string, got %T", v)
	}
	return dashboard, nil
}

func (r *ResourceDashboard) publishDashboard(ctx context.Context, id string, config *DashboardState) (*lakeview.PublishedDashboard, error) {
	// embed_credentials has a zero valued default in resourcemutator/resource_mutator.go,
	// so it is always sent, including when false.
	resp, err := r.client.Lakeview.PublishDashboard(ctx, sdklakeview.PublishDashboardRequest{
		DashboardId:      new(id),
		EmbedCredentials: new(config.EmbedCredentials),
		WarehouseId:      nonEmpty(config.WarehouseId),
	})
	if err != nil {
		return nil, err
	}
	return lakeview.PublishedDashboardFromSDK(resp)
}

func timeString(t *types.Time) string {
	if t == nil {
		return ""
	}
	return t.String()
}

func dashboardToState(dashboard *lakeview.Dashboard, publishResp *lakeview.PublishedDashboard, serializedDashboard string, published bool) *DashboardState {
	var embedCredentials bool
	if publishResp != nil {
		embedCredentials = utils.Deref(publishResp.EmbedCredentials)
	}

	return &DashboardState{
		DisplayName:         utils.Deref(dashboard.DisplayName),
		Etag:                utils.Deref(dashboard.Etag),
		WarehouseId:         utils.Deref(dashboard.WarehouseId),
		SerializedDashboard: serializedDashboard,
		ParentPath:          ensureWorkspacePrefix(utils.Deref(dashboard.ParentPath)),
		// diffs are detected via etags, which will change if dataset_catalog/dataset_schema is updated.
		DatasetCatalog: "",
		DatasetSchema:  "",

		// Output only fields
		CreateTime:     timeString(dashboard.CreateTime),
		DashboardId:    utils.Deref(dashboard.DashboardId),
		LifecycleState: dashboard.LifecycleState,
		Path:           utils.Deref(dashboard.Path),
		UpdateTime:     timeString(dashboard.UpdateTime),

		EmbedCredentials: embedCredentials,
		Published:        published,
	}
}

func (r *ResourceDashboard) createDashboard(ctx context.Context, dashboard *lakeview.Dashboard, config *DashboardState) (*lakeview.Dashboard, error) {
	create := &lakeview.DashboardCreate{
		DisplayName:         dashboard.DisplayName,
		ParentPath:          dashboard.ParentPath,
		WarehouseId:         dashboard.WarehouseId,
		SerializedDashboard: dashboard.SerializedDashboard,
		CreateTime:          nil, // Not set by DABs
		DashboardId:         nil, // Not set by DABs
		LifecycleState:      "",  // Not set by DABs
		Path:                nil, // Not set by DABs
		UpdateTime:          nil, // Not set by DABs
	}
	request, err := lakeview.CreateDashboardRequestToSDK(&lakeview.CreateDashboardRequest{
		Dashboard:      create,
		DatasetCatalog: nonEmpty(config.DatasetCatalog),
		DatasetSchema:  nonEmpty(config.DatasetSchema),
	})
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Lakeview.CreateDashboard(ctx, *request)
	if err != nil {
		return nil, err
	}
	return lakeview.DashboardFromSDK(resp)
}

func (r *ResourceDashboard) trashDashboard(ctx context.Context, id string) error {
	_, err := r.client.Lakeview.TrashDashboard(ctx, sdklakeview.TrashDashboardRequest{DashboardId: new(id)})
	return err
}

func (r *ResourceDashboard) DoCreate(ctx context.Context, config *DashboardState) (string, *DashboardState, error) {
	dashboard, err := prepareDashboardRequest(config)
	if err != nil {
		return "", nil, err
	}

	createResp, err := r.createDashboard(ctx, dashboard, config)

	// The API returns 404 if the parent directory doesn't exist.
	// If the parent directory doesn't exist, create it and try again.
	if err != nil && apierrs.IsMissing(err) {
		err = r.client.Workspace.MkdirsByPath(ctx, config.ParentPath)
		if err != nil {
			return "", nil, fmt.Errorf("failed to create parent directory: %w", err)
		}
		createResp, err = r.createDashboard(ctx, dashboard, config)
	}
	if err != nil {
		return "", nil, err
	}

	dashboardID := utils.Deref(createResp.DashboardId)

	// Persist the etag in state.
	config.Etag = utils.Deref(createResp.Etag)

	var publishResp *lakeview.PublishedDashboard
	// Note, today config.Published is always true (we do not have this field in input config).
	if config.Published {
		publishResp, err = r.publishDashboard(ctx, dashboardID, config)
		if err != nil {
			// If the publish fails, we should delete the dashboard to avoid leaving it in a bad state.
			deleteErr := r.trashDashboard(ctx, dashboardID)
			if deleteErr != nil {
				log.Warnf(ctx, "failed to delete draft dashboard %s after publish failed: %v", dashboardID, deleteErr)
				return "", nil, deleteErr
			}
			return "", nil, err
			// QQQ: instead, we could store partial state with published=false
		}
	}

	return dashboardID, dashboardToState(createResp, publishResp, utils.Deref(dashboard.SerializedDashboard), config.Published), nil
}

func (r *ResourceDashboard) DoUpdate(ctx context.Context, id string, config *DashboardState, _ *PlanEntry) (*DashboardState, error) {
	dashboard, err := prepareDashboardRequest(config)
	if err != nil {
		return nil, err
	}
	dashboard.DashboardId = new(id)

	// TODO: if the only thing that changed was "published", we can skip Update()?

	request, err := lakeview.UpdateDashboardRequestToSDK(&lakeview.UpdateDashboardRequest{
		Dashboard:      dashboard,
		DatasetCatalog: nonEmpty(config.DatasetCatalog),
		DatasetSchema:  nonEmpty(config.DatasetSchema),
	})
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Lakeview.UpdateDashboard(ctx, *request)
	if err != nil {
		return nil, err
	}
	updateResp, err := lakeview.DashboardFromSDK(resp)
	if err != nil {
		return nil, err
	}

	// Persist the etag in state.
	config.Etag = utils.Deref(updateResp.Etag)

	var publishResp *lakeview.PublishedDashboard
	// Note, today config.Published is always true (we do not have this field in input config).
	if config.Published {
		publishResp, err = r.publishDashboard(ctx, id, config)
		if err != nil {
			// TODO: store partial state with published=false?
			return nil, err
		}
	}

	return dashboardToState(updateResp, publishResp, utils.Deref(dashboard.SerializedDashboard), config.Published), nil
}

func (r *ResourceDashboard) DoDelete(ctx context.Context, id string, _ *DashboardState) error {
	return r.trashDashboard(ctx, id)
}

func (r *ResourceDashboard) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, _ *DashboardState) error {
	switch path.String() {
	case "etag":
		// change.New is always nil for etag because it's not present in the config
		// We compare stored etag with remote one.
		if change.Old == change.Remote {
			change.Action = deployplan.Skip
		} else {
			change.Action = deployplan.Update
		}
	}

	return nil
}

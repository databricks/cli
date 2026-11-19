package dresources

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structcopy"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"golang.org/x/sync/errgroup"
)

// Terraform implementation reference: https://github.com/databricks/terraform-provider-databricks/blob/main/dashboards/resource_dashboard.go
type ResourceDashboard struct {
	client *databricks.WorkspaceClient
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

// We need these dummy methiods otherwise custom marshaller from DashboardConfig takes over and ignores Published field
func (s *DashboardState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s DashboardState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
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

func (*ResourceDashboard) New(client *databricks.WorkspaceClient) *ResourceDashboard {
	return &ResourceDashboard{client: client}
}

func (*ResourceDashboard) PrepareState(input *resources.Dashboard) *DashboardState {
	return &DashboardState{
		DashboardConfig: input.DashboardConfig,
		Published:       true,
	}
}

func (r *ResourceDashboard) DoRead(ctx context.Context, id string) (*DashboardState, error) {
	var dashboard *dashboards.Dashboard
	var publishedDashboard *dashboards.PublishedDashboard
	var publishedErr error

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		dashboard, err = r.client.Lakeview.Get(ctx, dashboards.GetDashboardRequest{
			DashboardId: id,
		})
		return err
	})

	g.Go(func() error {
		var err error
		publishedDashboard, err = r.client.Lakeview.GetPublished(ctx, dashboards.GetPublishedDashboardRequest{
			DashboardId: id,
		})
		// Store error but don't fail - dashboard might be unpublished
		publishedErr = err
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// A trashed dashboard is considered deleted - return not found error
	if dashboard.LifecycleState == dashboards.LifecycleStateTrashed {
		return nil, apierr.ErrNotFound
	}

	// A dashboard is published (in the sense that matters for planning: the current
	// draft content is what's published) only when GetPublished succeeds AND the
	// published revision is at least as new as the draft. A draft update without a
	// republish leaves stale content published, but GetPublished keeps returning it,
	// so publishedErr == nil alone cannot detect staleness — the timestamp comparison
	// can. If either timestamp is unparseable, fall back to publish-existence so we
	// don't spuriously republish. See DashboardState.Published.
	published := publishedErr == nil && publishedIsCurrent(dashboard, publishedDashboard)

	return responseToState(dashboard, publishedDashboard, dashboard.SerializedDashboard, published), nil
}

// publishedIsCurrent reports whether the published revision reflects the current
// draft, i.e. the dashboard was published after its last draft update. The published
// API exposes no etag, only revision_create_time, so we compare it against the draft's
// update_time. Both are RFC3339 timestamps set by the backend on a shared clock
// (verified on AWS/GCP/Azure/dogfood: a publish always leaves revision_create_time >=
// update_time, and a draft update pushes update_time ahead until the next publish).
// If either timestamp is missing or unparseable, return true so we do not force a
// spurious republish.
func publishedIsCurrent(dashboard *dashboards.Dashboard, published *dashboards.PublishedDashboard) bool {
	if published == nil {
		return false
	}
	updateTime, err := time.Parse(time.RFC3339, dashboard.UpdateTime)
	if err != nil {
		return true
	}
	revisionTime, err := time.Parse(time.RFC3339, published.RevisionCreateTime)
	if err != nil {
		return true
	}
	return !revisionTime.Before(updateTime)
}

// embed_credentials as a zero valued default in resourcemutator/resource_mutator.go.
// Thus we always need to include it in the ForceSendFields list to ensure that it is sent to
// the server, and that it's serialized in the state even when false (zero value).
var dashboardForceSend = []string{"EmbedCredentials"}

// serialized_dashboard is a string in the API and any in the config, so each copier skips it
// and the caller sets it.
var (
	// The output-only fields are not sent. Fields like "embed_credentials" are part of the
	// bundle configuration but not the create request here, so they are left out.
	toDashboardRequest = newCopier[DashboardState, dashboards.Dashboard](
		structcopy.Skip("CreateTime", "DashboardId", "LifecycleState", "Path", "UpdateTime", "SerializedDashboard"),
	)

	// DashboardId addresses the dashboard, so it is the id.
	toPublishRequest = newCopier[DashboardState, dashboards.PublishRequest](
		structcopy.Skip("DashboardId"),
		structcopy.ForceSend(dashboardForceSend...),
	)

	// The output-only fields (create_time, path, ...) are copied as well. dataset_catalog and
	// dataset_schema are not reported: diffs are detected via etags, which will change if
	// dataset_catalog/dataset_schema is updated.
	toDashboardState = newCopier[dashboards.Dashboard, DashboardState](
		structcopy.Skip("SerializedDashboard"),
		structcopy.ForceSend(dashboardForceSend...),
	)
)

func prepareDashboardRequest(config *DashboardState) (dashboards.Dashboard, error) {
	dashboard := toDashboardRequest.Copy(config)
	switch v := config.SerializedDashboard.(type) {
	case nil:
	case string:
		dashboard.SerializedDashboard = v
	default:
		return dashboards.Dashboard{}, fmt.Errorf("internal error: serialized_dashboard should have been normalized to a string, got %T", v)
	}
	return *dashboard, nil
}

func (r *ResourceDashboard) publishDashboard(ctx context.Context, id string, config *DashboardState) (*dashboards.PublishedDashboard, error) {
	request := toPublishRequest.Copy(config)
	request.DashboardId = id
	return r.client.Lakeview.Publish(ctx, *request)
}

// responseToState returns the state of a dashboard the API reported. publishResp is nil for a
// dashboard that is not published.
func responseToState(createOrUpdateResp *dashboards.Dashboard, publishResp *dashboards.PublishedDashboard, serializedDashboard string, published bool) *DashboardState {
	state := toDashboardState.Copy(createOrUpdateResp)
	state.SerializedDashboard = serializedDashboard
	state.ParentPath = ensureWorkspacePrefix(createOrUpdateResp.ParentPath)
	if publishResp != nil {
		state.EmbedCredentials = publishResp.EmbedCredentials
	}
	state.Published = published
	return state
}

func (r *ResourceDashboard) DoCreate(ctx context.Context, config *DashboardState) (string, *DashboardState, error) {
	dashboard, err := prepareDashboardRequest(config)
	if err != nil {
		return "", nil, err
	}

	create := func(ctx context.Context) (*dashboards.Dashboard, error) {
		return r.client.Lakeview.Create(ctx, dashboards.CreateDashboardRequest{
			Dashboard:      dashboard,
			DatasetCatalog: config.DatasetCatalog,
			DatasetSchema:  config.DatasetSchema,
		})
	}
	start := time.Now()
	createResp, err := create(ctx)

	// The API returns 404 if the parent directory doesn't exist.
	// If the parent directory doesn't exist, create it and try again.
	if err != nil && apierr.IsMissing(err) {
		err = r.client.Workspace.MkdirsByPath(ctx, config.ParentPath)
		if err != nil {
			return "", nil, fmt.Errorf("failed to create parent directory: %w", err)
		}
		createResp, err = create(ctx)
	}
	if err != nil && config.ParentPath != "" {
		createResp, err = r.recoverFailedCreate(ctx, config, start, create, err)
	}
	if err != nil {
		return "", nil, err
	}

	// Persist the etag in state.
	config.Etag = createResp.Etag

	var publishResp *dashboards.PublishedDashboard
	// Note, today config.Published is always true (we do not have this field in input config).
	if config.Published {
		publishResp, err = r.publishDashboard(ctx, createResp.DashboardId, config)
		if err != nil {
			// If the publish fails, we should delete the dashboard to avoid leaving it in a bad state.
			deleteErr := r.client.Lakeview.Trash(ctx, dashboards.TrashDashboardRequest{
				DashboardId: createResp.DashboardId,
			})
			if deleteErr != nil {
				log.Warnf(ctx, "failed to delete draft dashboard %s after publish failed: %v", createResp.DashboardId, deleteErr)
				return "", nil, deleteErr
			}
			return "", nil, err
			// QQQ: instead, we could store partial state with published=false
		}
	}

	return createResp.DashboardId, responseToState(createResp, publishResp, dashboard.SerializedDashboard, config.Published), nil
}

// recoverFailedCreate handles a create request that failed; see recoverCreate. A dashboard is
// stored as the workspace file <parent_path>/<display_name>.lvdash.json, so the API rejects a
// second dashboard with the same display name in the same folder, and the file names the
// dashboard to look up.
func (r *ResourceDashboard) recoverFailedCreate(ctx context.Context, config *DashboardState, start time.Time, create func(context.Context) (*dashboards.Dashboard, error), createErr error) (*dashboards.Dashboard, error) {
	filePath := path.Join(ensureWorkspacePrefix(config.ParentPath), config.DisplayName+".lvdash.json")
	return recoverCreate(ctx, createRecovery[dashboards.Dashboard]{
		name:  "dashboard " + filePath,
		start: start,
		get: func(ctx context.Context) (*dashboards.Dashboard, error) {
			info, err := r.client.Workspace.GetStatusByPath(ctx, filePath)
			if err != nil {
				return nil, err
			}
			if info.ObjectType != workspace.ObjectTypeDashboard || info.ResourceId == "" {
				return nil, fmt.Errorf("%s is a %s, not a dashboard", filePath, info.ObjectType)
			}
			return r.client.Lakeview.Get(ctx, dashboards.GetDashboardRequest{DashboardId: info.ResourceId})
		},
		createTime: func(found *dashboards.Dashboard) (time.Time, bool) {
			created, err := time.Parse(time.RFC3339, found.CreateTime)
			return created, err == nil
		},
		create: create,
	}, createErr)
}

func (r *ResourceDashboard) DoUpdate(ctx context.Context, id string, config *DashboardState, _ *PlanEntry) (*DashboardState, error) {
	dashboard, err := prepareDashboardRequest(config)
	if err != nil {
		return nil, err
	}

	// TODO: if the only thing that changed was "published", we can skip Update()?

	updateResp, err := r.client.Lakeview.Update(ctx, dashboards.UpdateDashboardRequest{
		DashboardId:    id,
		Dashboard:      dashboard,
		DatasetCatalog: config.DatasetCatalog,
		DatasetSchema:  config.DatasetSchema,
	})
	if err != nil {
		return nil, err
	}

	// Persist the etag in state.
	config.Etag = updateResp.Etag

	var publishResp *dashboards.PublishedDashboard
	// Note, today config.Published is always true (we do not have this field in input config).
	if config.Published {
		publishResp, err = r.publishDashboard(ctx, id, config)
		if err != nil {
			// TODO: store partial state with published=false?
			return nil, err
		}
	}

	return responseToState(updateResp, publishResp, dashboard.SerializedDashboard, config.Published), nil
}

func (r *ResourceDashboard) DoDelete(ctx context.Context, id string, _ *DashboardState) error {
	return r.client.Lakeview.Trash(ctx, dashboards.TrashDashboardRequest{
		DashboardId: id,
	})
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

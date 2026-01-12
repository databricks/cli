package dresources

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"golang.org/x/sync/errgroup"
)

// Terraform implementation reference: https://github.com/databricks/terraform-provider-databricks/blob/main/dashboards/resource_dashboard.go
type ResourceDashboard struct {
	client *databricks.WorkspaceClient
}

type DashboardState struct {
	resources.DashboardConfig
	Published bool `json:"published"`
}

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

func (r *ResourceDashboard) RemapState(state *DashboardState) *DashboardState {
	forceSendFields := utils.FilterFields[DashboardState](state.ForceSendFields, []string{
		"CreateTime",
		"DashboardId",
		"LifecycleState",
		"Path",
		"UpdateTime",
		"SerializedDashboard",
		"DatasetCatalog",
		"DatasetSchema",
	}...)

	// EmbedCredentials must always be included in ForceSendFields to ensure it's serialized
	// even when false (zero value).
	if !slices.Contains(forceSendFields, "EmbedCredentials") {
		forceSendFields = append(forceSendFields, "EmbedCredentials")
	}

	return &DashboardState{
		DashboardConfig: resources.DashboardConfig{
			DisplayName:         state.DisplayName,
			Etag:                state.Etag,
			ParentPath:          state.ParentPath,
			WarehouseId:         state.WarehouseId,
			SerializedDashboard: state.SerializedDashboard,
			EmbedCredentials:    state.EmbedCredentials,
			DatasetCatalog:      state.DatasetCatalog,
			DatasetSchema:       state.DatasetSchema,

			ForceSendFields: forceSendFields,

			// Clear output only fields. They should not show up on remote diff computation.
			CreateTime:     "",
			DashboardId:    "",
			LifecycleState: dashboards.LifecycleState(""),
			Path:           "",
			UpdateTime:     "",
		},
		Published: state.Published,
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

	// Determine if the dashboard is published
	published := publishedErr == nil

	forceSendFields := utils.FilterFields[DashboardState](dashboard.ForceSendFields)
	// EmbedCredentials must always be included in ForceSendFields to ensure it's serialized
	// even when false (zero value).
	if !slices.Contains(forceSendFields, "EmbedCredentials") {
		forceSendFields = append(forceSendFields, "EmbedCredentials")
	}

	var embedCredentials bool
	if publishedDashboard != nil {
		embedCredentials = publishedDashboard.EmbedCredentials
	}

	return &DashboardState{
		DashboardConfig: resources.DashboardConfig{
			DisplayName:         dashboard.DisplayName,
			Etag:                dashboard.Etag,
			WarehouseId:         dashboard.WarehouseId,
			SerializedDashboard: dashboard.SerializedDashboard,
			ParentPath:          ensureWorkspacePrefix(dashboard.ParentPath),
			// diffs are detected via etags, which will change if dataset_catalog/dataset_schema is updated.
			DatasetCatalog: "",
			DatasetSchema:  "",

			// Output only fields.
			CreateTime:      dashboard.CreateTime,
			DashboardId:     dashboard.DashboardId,
			LifecycleState:  dashboard.LifecycleState,
			Path:            dashboard.Path,
			UpdateTime:      dashboard.UpdateTime,
			ForceSendFields: forceSendFields,

			EmbedCredentials: embedCredentials,
		},
		Published: published,
	}, nil
}

func prepareDashboardRequest(config *DashboardState) (dashboards.Dashboard, error) {
	dashboard := dashboards.Dashboard{
		DisplayName:         config.DisplayName,
		ParentPath:          config.ParentPath,
		WarehouseId:         config.WarehouseId,
		Etag:                config.Etag,
		CreateTime:          "",
		DashboardId:         "",
		LifecycleState:      "",
		Path:                "",
		SerializedDashboard: "",
		UpdateTime:          "",
		// Fields like "embed_credentials" are part of the bundle configuration but not the create request here.
		// Thus we need to filter such fields out.
		ForceSendFields: utils.FilterFields[dashboards.Dashboard](config.ForceSendFields),
	}
	v := config.SerializedDashboard
	if serializedDashboard, ok := v.(string); ok {
		// If serialized dashboard is already a string, we can use it directly.
		dashboard.SerializedDashboard = serializedDashboard
	} else if v != nil {
		// If it's inlined in the bundle config as a map, we need to marshal it to a string.
		b, err := json.Marshal(v)
		if err != nil {
			return dashboards.Dashboard{}, fmt.Errorf("failed to marshal serialized dashboard: %w", err)
		}
		dashboard.SerializedDashboard = string(b)
	}
	return dashboard, nil
}

func (r *ResourceDashboard) publishDashboard(ctx context.Context, id string, config *DashboardState) (*dashboards.PublishedDashboard, error) {
	// embed_credentials as a zero valued default in resourcemutator/resource_mutator.go.
	// Thus we always need to include it in the ForceSendFields list to ensure that it is sent to the server.
	forceSendFields := utils.FilterFields[dashboards.PublishRequest](config.ForceSendFields)
	if !slices.Contains(forceSendFields, "EmbedCredentials") {
		forceSendFields = append(forceSendFields, "EmbedCredentials")
	}

	return r.client.Lakeview.Publish(ctx, dashboards.PublishRequest{
		DashboardId:      id,
		EmbedCredentials: config.EmbedCredentials,
		WarehouseId:      config.WarehouseId,
		ForceSendFields:  forceSendFields,
	})
}

func responseToState(createOrUpdateResp *dashboards.Dashboard, publishResp *dashboards.PublishedDashboard, serializedDashboard string, published bool) *DashboardState {
	forceSendFields := utils.FilterFields[DashboardState](createOrUpdateResp.ForceSendFields)
	// EmbedCredentials must always be included in ForceSendFields to ensure it's serialized
	// even when false (zero value).
	if !slices.Contains(forceSendFields, "EmbedCredentials") {
		forceSendFields = append(forceSendFields, "EmbedCredentials")
	}

	var embedCredentials bool
	if publishResp != nil {
		embedCredentials = publishResp.EmbedCredentials
	}

	return &DashboardState{
		DashboardConfig: resources.DashboardConfig{
			DisplayName:         createOrUpdateResp.DisplayName,
			Etag:                createOrUpdateResp.Etag,
			WarehouseId:         createOrUpdateResp.WarehouseId,
			SerializedDashboard: serializedDashboard,
			ParentPath:          ensureWorkspacePrefix(createOrUpdateResp.ParentPath),
			DatasetCatalog:      "",
			DatasetSchema:       "",

			// Output only fields
			CreateTime:      createOrUpdateResp.CreateTime,
			DashboardId:     createOrUpdateResp.DashboardId,
			LifecycleState:  createOrUpdateResp.LifecycleState,
			Path:            createOrUpdateResp.Path,
			UpdateTime:      createOrUpdateResp.UpdateTime,
			ForceSendFields: forceSendFields,

			EmbedCredentials: embedCredentials,
		},
		Published: published,
	}
}

func (r *ResourceDashboard) DoCreate(ctx context.Context, config *DashboardState) (string, *DashboardState, error) {
	dashboard, err := prepareDashboardRequest(config)
	if err != nil {
		return "", nil, err
	}

	createResp, err := r.client.Lakeview.Create(ctx, dashboards.CreateDashboardRequest{
		Dashboard:      dashboard,
		DatasetCatalog: config.DatasetCatalog,
		DatasetSchema:  config.DatasetSchema,

		ForceSendFields: nil,
	})

	// The API returns 404 if the parent directory doesn't exist.
	// If the parent directory doesn't exist, create it and try again.
	if err != nil && apierr.IsMissing(err) {
		err = r.client.Workspace.MkdirsByPath(ctx, config.ParentPath)
		if err != nil {
			return "", nil, fmt.Errorf("failed to create parent directory: %w", err)
		}
		createResp, err = r.client.Lakeview.Create(ctx, dashboards.CreateDashboardRequest{
			Dashboard:      dashboard,
			DatasetCatalog: config.DatasetCatalog,
			DatasetSchema:  config.DatasetSchema,

			ForceSendFields: nil,
		})
	}
	if err != nil {
		return "", nil, err
	}

	// Persist the etag in state.
	config.Etag = createResp.Etag

	// Only publish if config.Published is true
	var publishResp *dashboards.PublishedDashboard
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
		}
	}

	return createResp.DashboardId, responseToState(createResp, publishResp, dashboard.SerializedDashboard, config.Published), nil
}

func (r *ResourceDashboard) DoUpdate(ctx context.Context, id string, config *DashboardState, _ Changes) (*DashboardState, error) {
	dashboard, err := prepareDashboardRequest(config)
	if err != nil {
		return nil, err
	}

	updateResp, err := r.client.Lakeview.Update(ctx, dashboards.UpdateDashboardRequest{
		DashboardId:    id,
		Dashboard:      dashboard,
		DatasetCatalog: config.DatasetCatalog,
		DatasetSchema:  config.DatasetSchema,

		ForceSendFields: nil,
	})
	if err != nil {
		return nil, err
	}

	// Persist the etag in state.
	config.Etag = updateResp.Etag

	// Only publish if config.Published is true
	var publishResp *dashboards.PublishedDashboard
	if config.Published {
		publishResp, err = r.publishDashboard(ctx, id, config)
		if err != nil {
			return nil, err
		}
	}

	return responseToState(updateResp, publishResp, dashboard.SerializedDashboard, config.Published), nil
}

func (r *ResourceDashboard) DoDelete(ctx context.Context, id string) error {
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
	case "parent_path":
		if change.Action == deployplan.Update {
			change.Action = deployplan.Recreate
		}
	case "serialized_dashboard", "dataset_catalog", "dataset_schema":
		// "serialized_dashboard" locally and remotely will have different diffs.
		// We only need to rely on etag here, and can skip this field for diff computation.

		// "dataset_catalog" and "dataset_schema" are write-only fields that are not returned by the server.
		// They will always differ between local config (which has values) and remote state (which has empty strings),
		// so we skip them for remote diff computation to avoid false positives.
		if change.Old == change.New {
			change.Action = deployplan.Skip
		}
	}

	return nil
}

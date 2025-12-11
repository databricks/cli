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
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"golang.org/x/sync/errgroup"
)

// Terraform implementation reference: https://github.com/databricks/terraform-provider-databricks/blob/main/dashboards/resource_dashboard.go
type ResourceDashboard struct {
	client *databricks.WorkspaceClient
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

func (*ResourceDashboard) PrepareState(input *resources.Dashboard) *resources.DashboardConfig {
	return &input.DashboardConfig
}

func (r *ResourceDashboard) RemapState(state *resources.DashboardConfig) *resources.DashboardConfig {
	forceSendFields := utils.FilterFields[resources.DashboardConfig](state.ForceSendFields, []string{
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

	return &resources.DashboardConfig{
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
	}
}

func (r *ResourceDashboard) DoRead(ctx context.Context, id string) (*resources.DashboardConfig, error) {
	var dashboard *dashboards.Dashboard
	var publishedDashboard *dashboards.PublishedDashboard

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
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	forceSendFields := utils.FilterFields[resources.DashboardConfig](dashboard.ForceSendFields)
	// EmbedCredentials must always be included in ForceSendFields to ensure it's serialized
	// even when false (zero value).
	if !slices.Contains(forceSendFields, "EmbedCredentials") {
		forceSendFields = append(forceSendFields, "EmbedCredentials")
	}

	return &resources.DashboardConfig{
		DisplayName:         dashboard.DisplayName,
		Etag:                dashboard.Etag,
		WarehouseId:         dashboard.WarehouseId,
		SerializedDashboard: dashboard.SerializedDashboard,
		ParentPath:          ensureWorkspacePrefix(dashboard.ParentPath),
		// diffs are detected via etags, which will change if dataset_catalog/dataset_schema is updated.
		DatasetCatalog:      "",
		DatasetSchema:       "",

		// Output only fields.
		CreateTime:      dashboard.CreateTime,
		DashboardId:     dashboard.DashboardId,
		LifecycleState:  dashboard.LifecycleState,
		Path:            dashboard.Path,
		UpdateTime:      dashboard.UpdateTime,
		ForceSendFields: forceSendFields,

		EmbedCredentials: publishedDashboard.EmbedCredentials,
	}, nil
}

func prepareDashboardRequest(config *resources.DashboardConfig) (dashboards.Dashboard, error) {
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

func (r *ResourceDashboard) publishDashboard(ctx context.Context, id string, config *resources.DashboardConfig) (*dashboards.PublishedDashboard, error) {
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

func responseToState(createOrUpdateResp *dashboards.Dashboard, publishResp *dashboards.PublishedDashboard, serializedDashboard string) *resources.DashboardConfig {
	forceSendFields := utils.FilterFields[resources.DashboardConfig](createOrUpdateResp.ForceSendFields)
	// EmbedCredentials must always be included in ForceSendFields to ensure it's serialized
	// even when false (zero value).
	if !slices.Contains(forceSendFields, "EmbedCredentials") {
		forceSendFields = append(forceSendFields, "EmbedCredentials")
	}

	return &resources.DashboardConfig{
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

		EmbedCredentials: publishResp.EmbedCredentials,
	}
}

func (r *ResourceDashboard) DoCreate(ctx context.Context, config *resources.DashboardConfig) (string, *resources.DashboardConfig, error) {
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
	publishResp, err := r.publishDashboard(ctx, createResp.DashboardId, config)
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

	return createResp.DashboardId, responseToState(createResp, publishResp, dashboard.SerializedDashboard), nil
}

func (r *ResourceDashboard) DoUpdate(ctx context.Context, id string, config *resources.DashboardConfig, _ *Changes) (*resources.DashboardConfig, error) {
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

	publishResp, err := r.publishDashboard(ctx, id, config)
	if err != nil {
		return nil, err
	}

	return responseToState(updateResp, publishResp, dashboard.SerializedDashboard), nil
}

func (r *ResourceDashboard) DoDelete(ctx context.Context, id string) error {
	return r.client.Lakeview.Trash(ctx, dashboards.TrashDashboardRequest{
		DashboardId: id,
	})
}

func (*ResourceDashboard) FieldTriggers(isLocal bool) map[string]deployplan.ActionType {
	// Common triggers for both local and remote.
	triggers := map[string]deployplan.ActionType{
		// change in parent_path should trigger a recreate
		"parent_path": deployplan.ActionTypeRecreate,
	}

	if isLocal {
		// Etags should only be compared for remote diffs, not local diffs.
		triggers["etag"] = deployplan.ActionTypeSkip
	} else {
		// Note: If the etag changes remotely, it means the dashboard has been modified remotely
		// and needs to be updated to match with the config. Since update is the default action type,
		// we don't need to explicitly specify it here.
		//
		// Note: etags cannot be specified in the bundle config since we error if they are set.

		// "serialized_dashboard" locally and remotely will have different diffs.
		// We only need to rely on etag here, and can skip this field for diff computation.
		triggers["serialized_dashboard"] = deployplan.ActionTypeSkip
	}

	return triggers
}

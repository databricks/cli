package dresources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
)

// Terraform implementation reference: https://github.com/databricks/terraform-provider-databricks/blob/main/dashboards/resource_dashboard.go
type ResourceDashboard struct {
	client *databricks.WorkspaceClient
}

func (*ResourceDashboard) New(client *databricks.WorkspaceClient) *ResourceDashboard {
	return &ResourceDashboard{client: client}
}

func (*ResourceDashboard) PrepareState(input *resources.Dashboard) *resources.DashboardConfig {
	// Unset serialized dashboard in the [dashboards.Dashboard] struct.
	// Only the serialized_dashboard field in the [dashboard.DashboardConfig] struct should be used.
	dashboard := input.Dashboard
	dashboard.SerializedDashboard = ""
	dashboard.ForceSendFields = filterFields[dashboards.Dashboard](dashboard.ForceSendFields, "SerializedDashboard")

	input.Dashboard = dashboard
	return &input.DashboardConfig
}

// TODO(followup): do this for all resources automatically to avoid boilerplate code.
func (r *ResourceDashboard) RemapState(state *resources.DashboardConfig) *resources.DashboardConfig {
	dashboard := &resources.DashboardConfig{
		Dashboard: dashboards.Dashboard{
			DisplayName: state.DisplayName,
			Etag:        state.Etag,
			ParentPath:  state.ParentPath,
			WarehouseId: state.WarehouseId,
			ForceSendFields: filterFields[dashboards.Dashboard](state.ForceSendFields, []string{
				"CreateTime",
				"DashboardId",
				"LifecycleState",
				"Path",
				"UpdateTime",
				"SerializedDashboard",
			}...),

			// Clear output only fields. They should not show up on remote diff computation.
			CreateTime:     "",
			DashboardId:    "",
			LifecycleState: dashboards.LifecycleState(""),
			Path:           "",
			UpdateTime:     "",

			// Serialized dashboard is ignored for remote diff changes.
			// They are only relevant for local diff changes.
			SerializedDashboard: "",
		},

		EmbedCredentials: state.EmbedCredentials,
		ForceSendFields: filterFields[resources.DashboardConfig](state.ForceSendFields, []string{
			"SerializedDashboard",
		}...),

		// Serialized dashboard is ignored for remote diff changes.
		// They are only relevant for local diff changes.
		SerializedDashboard: "",
	}

	return dashboard
}

func (r *ResourceDashboard) DoRefresh(ctx context.Context, id string) (*resources.DashboardConfig, error) {
	wg := sync.WaitGroup{}
	var dashboard *dashboards.Dashboard
	var publishedDashboard *dashboards.PublishedDashboard
	var getErr, getPublishedErr error

	wg.Go(func() {
		dashboard, getErr = r.client.Lakeview.Get(ctx, dashboards.GetDashboardRequest{
			DashboardId: id,
		})
	})
	wg.Go(func() {
		publishedDashboard, getPublishedErr = r.client.Lakeview.GetPublished(ctx, dashboards.GetPublishedDashboardRequest{
			DashboardId: id,
		})
	})

	// Wait for both GET call to complete
	wg.Wait()
	if getErr != nil {
		return nil, fmt.Errorf("failed to get draft dashboard: %w", getErr)
	}
	if getPublishedErr != nil {
		return nil, fmt.Errorf("failed to get published dashboard: %w", getPublishedErr)
	}

	return &resources.DashboardConfig{
		Dashboard: dashboards.Dashboard{
			DisplayName:         dashboard.DisplayName,
			Etag:                dashboard.Etag,
			WarehouseId:         dashboard.WarehouseId,
			SerializedDashboard: dashboard.SerializedDashboard,

			// Add the /Workspace prefix to the parent path. The backend removes this prefix from parent
			// path, and thus it needs to be added back in to match the local configuration.
			ParentPath: path.Join("/Workspace", dashboard.ParentPath),

			// Output only fields.
			CreateTime:      dashboard.CreateTime,
			DashboardId:     dashboard.DashboardId,
			LifecycleState:  dashboard.LifecycleState,
			Path:            dashboard.Path,
			UpdateTime:      dashboard.UpdateTime,
			ForceSendFields: filterFields[dashboards.Dashboard](dashboard.ForceSendFields),
		},
		SerializedDashboard: dashboard.SerializedDashboard,
		EmbedCredentials:    publishedDashboard.EmbedCredentials,
		ForceSendFields:     filterFields[dashboards.PublishedDashboard](publishedDashboard.ForceSendFields),
	}, nil
}

func isParentDoesntExistError(err error) bool {
	errStr := err.Error()
	return strings.HasPrefix(errStr, "Path (") && strings.HasSuffix(errStr, ") doesn't exist.")
}

func prepareDashboardRequest(config *resources.DashboardConfig) (dashboards.Dashboard, error) {
	// Fields like "embed_credentials" are part of the bundle configuration but not the create request here.
	// Thus we need to filter such fields out.
	config.Dashboard.ForceSendFields = filterFields[dashboards.Dashboard](config.Dashboard.ForceSendFields)

	dashboard := config.Dashboard
	v := config.SerializedDashboard
	if _, ok := v.(string); ok {
		// If serialized dashboard is already a string, we can use it directly.
		dashboard.SerializedDashboard = v.(string)
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
	// TODO(followup): A more general solution that does not require this special casing.
	forceSendFields := filterFields[dashboards.PublishRequest](config.ForceSendFields)
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

func responseToState(createOrUpdateResp *dashboards.Dashboard, publishResp *dashboards.PublishedDashboard) *resources.DashboardConfig {
	return &resources.DashboardConfig{
		Dashboard: dashboards.Dashboard{
			DisplayName:         createOrUpdateResp.DisplayName,
			Etag:                createOrUpdateResp.Etag,
			WarehouseId:         createOrUpdateResp.WarehouseId,
			SerializedDashboard: createOrUpdateResp.SerializedDashboard,

			// Add the /Workspace prefix to the parent path. The backend removes this prefix from parent
			// path, and thus it needs to be added back in to match the local configuration.
			ParentPath: path.Join("/Workspace", createOrUpdateResp.ParentPath),

			// Output only fields
			CreateTime:      createOrUpdateResp.CreateTime,
			DashboardId:     createOrUpdateResp.DashboardId,
			LifecycleState:  createOrUpdateResp.LifecycleState,
			Path:            createOrUpdateResp.Path,
			UpdateTime:      createOrUpdateResp.UpdateTime,
			ForceSendFields: filterFields[dashboards.Dashboard](createOrUpdateResp.ForceSendFields),
		},
		SerializedDashboard: createOrUpdateResp.SerializedDashboard,
		EmbedCredentials:    publishResp.EmbedCredentials,
		ForceSendFields:     filterFields[dashboards.PublishedDashboard](publishResp.ForceSendFields),
	}
}

func (r *ResourceDashboard) DoCreate(ctx context.Context, config *resources.DashboardConfig) (string, *resources.DashboardConfig, error) {
	dashboard, err := prepareDashboardRequest(config)
	if err != nil {
		return "", nil, err
	}

	createResp, err := r.client.Lakeview.Create(ctx, dashboards.CreateDashboardRequest{
		Dashboard: dashboard,
	})

	// If the parent directory doesn't exist, create it and try again.
	if err != nil && isParentDoesntExistError(err) {
		err = r.client.Workspace.MkdirsByPath(ctx, config.ParentPath)
		if err != nil {
			return "", nil, fmt.Errorf("failed to create parent directory: %w", err)
		}
		createResp, err = r.client.Lakeview.Create(ctx, dashboards.CreateDashboardRequest{Dashboard: dashboard})
	}
	if err != nil {
		return "", nil, err
	}

	// Persist the etag in state.
	config.Etag = createResp.Etag

	publishResp, err := r.publishDashboard(ctx, createResp.DashboardId, config)
	if err != nil {
		return "", nil, err
	}

	return createResp.DashboardId, responseToState(createResp, publishResp), nil
}

func (r *ResourceDashboard) DoUpdate(ctx context.Context, id string, config *resources.DashboardConfig) (*resources.DashboardConfig, error) {
	dashboard, err := prepareDashboardRequest(config)
	if err != nil {
		return nil, err
	}

	updateResp, err := r.client.Lakeview.Update(ctx, dashboards.UpdateDashboardRequest{
		DashboardId: id,
		Dashboard:   dashboard,
	})
	if err != nil {
		return nil, err
	}

	publishResp, err := r.publishDashboard(ctx, id, config)
	if err != nil {
		return nil, err
	}

	return responseToState(updateResp, publishResp), nil
}

func (r *ResourceDashboard) DoDelete(ctx context.Context, id string) error {
	trashErr := r.client.Lakeview.Trash(ctx, dashboards.TrashDashboardRequest{
		DashboardId: id,
	})

	// Successfully deleted the dashboard. Return nil.
	if trashErr == nil {
		return nil
	}

	// If the dashboard was already trashed, we'll get a 403 (Permission Denied) error.
	// There may be other cases where we get a 403, so we first confirm that the
	// dashboard state is actually trashed, and if so, return success.
	if !errors.Is(trashErr, apierr.ErrPermissionDenied) {
		return trashErr
	}

	dashboard, err := r.client.Lakeview.Get(ctx, dashboards.GetDashboardRequest{
		DashboardId: id,
	})

	// If we can't get the dashboard state or the dashboard is not trashed, return the original error.
	if err != nil || dashboard.LifecycleState != dashboards.LifecycleStateTrashed {
		return trashErr
	}

	// Confirmed that the dashboard is indeed trashed. Return success.
	return nil
}

func (*ResourceDashboard) FieldTriggers(isLocal bool) map[string]deployplan.ActionType {
	// Common triggers for both local and remote.
	triggers := map[string]deployplan.ActionType{
		// change in parent_path should trigger a recreate
		"parent_path": deployplan.ActionTypeRecreate,
	}

	if isLocal {
		// Etags are not relevant to determine if the local configuration changed.
		triggers["etag"] = deployplan.ActionTypeSkip
	} else {
		// If the etag changes remotely, it means the dashboard has been modified remotely
		// and needs to be updated to match with the config.
		// Even though "update" action type is the default, we explicitly specify it here
		// to make this relationship clear.
		triggers["etag"] = deployplan.ActionTypeUpdate

		// "serialized_dashboard" locally and remotely will have different diffs. =
		// We only need to rely on etag here, and can skip this field for diff computation.
		triggers["serialized_dashboard"] = deployplan.ActionTypeSkip
	}

	return triggers
}

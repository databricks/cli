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
	"unicode"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
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

// TOOD: Serialize the dashboard from a JSON object to a string.
func (*ResourceDashboard) PrepareState(input *resources.Dashboard) *resources.DashboardConfig {
	// Unset serialized dashboard in the [dashboards.Dashboard] struct.
	// Only the serialized_dashboard field in the [dashboard.DashboardConfig] struct should be used.
	dashboard := input.DashboardConfig.Dashboard
	dashboard.SerializedDashboard = ""
	dashboard.ForceSendFields = filterFields[dashboards.Dashboard](dashboard.ForceSendFields, "SerializedDashboard")

	input.DashboardConfig.Dashboard = dashboard
	return &input.DashboardConfig
}

func snakeToTitle(snake string) string {
	out := strings.Builder{}
	makeUpper := true
	for _, r := range snake {
		if makeUpper {
			out.WriteRune(unicode.ToUpper(r))
			makeUpper = false
		} else if r == '_' {
			makeUpper = true
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// TODO(followup): do this for all resources automatically to avoid boilerplate code.
func (s *ResourceDashboard) RemapState(state *resources.DashboardConfig) *resources.DashboardConfig {
	// Output only fields are marked as skip in dashboards. They need to be cleaned up
	// before comparing with local configuration.
	fieldTriggersRemote := s.FieldTriggers(false)
	configForceSendFields := []string{}
	dashboardForceSendFields := []string{}
	for k, v := range fieldTriggersRemote {
		path, err := structpath.Parse(k)
		if err != nil {
			continue
		}
		if v == deployplan.ActionTypeSkip {
			// Remove the field from the state.
			structaccess.Set(state, path, nil)
			continue
		}
		titleField := snakeToTitle(k)
		if state.ForceSendFields != nil && slices.Contains(state.ForceSendFields, titleField) {
			configForceSendFields = append(configForceSendFields, titleField)
		}
		if state.Dashboard.ForceSendFields != nil && slices.Contains(state.Dashboard.ForceSendFields, titleField) {
			dashboardForceSendFields = append(dashboardForceSendFields, titleField)
		}
	}

	state.ForceSendFields = configForceSendFields
	state.Dashboard.ForceSendFields = dashboardForceSendFields

	// Set SerializedDashboard to nil. It's ignored for remote diff computation and thus
	// should always be nil here. Because it's overridden in [resources.DashboardConfig]
	// it needs to be set nil explicitly.
	state.SerializedDashboard = nil
	return state
}

// TODO: Add test ensuring that the detection of serialized dashboard local changes works properly.
func (r *ResourceDashboard) DoRefresh(ctx context.Context, id string) (*resources.DashboardConfig, error) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	var dashboard *dashboards.Dashboard
	var publishedDashboard *dashboards.PublishedDashboard
	var getErr, getPublishedErr error
	go func() {
		defer wg.Done()
		dashboard, getErr = r.client.Lakeview.Get(ctx, dashboards.GetDashboardRequest{
			DashboardId: id,
		})
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		// We need a separate GET call to get the embed_credentials field. This edge case is not
		// handled by TF and does mean that the embed_credentials diffs are not detected in TF.
		publishedDashboard, getPublishedErr = r.client.Lakeview.GetPublished(ctx, dashboards.GetPublishedDashboardRequest{
			DashboardId: id,
		})
	}()

	// Wait for both GET call to complete
	wg.Wait()
	if getErr != nil {
		return nil, fmt.Errorf("failed to get draft dashboard: %w", getErr)
	}
	if getPublishedErr != nil {
		return nil, fmt.Errorf("failed to get published dashboard: %w", getPublishedErr)
	}

	// Add the /Workspace prefix to the parent path. The backend removes this prefix from parent
	// path, and thus it needs to be added back in to match the local configuration.
	dashboard.ParentPath = path.Join("/Workspace", dashboard.ParentPath)

	// Unset serialized dashboard in the [dashboards.Dashboard] struct.
	// Only the serialized_dashboard field in the [dashboard.DashboardConfig] struct is should be used.
	dashboard.SerializedDashboard = ""
	getDashboardForceSendFields := filterFields[dashboards.Dashboard](dashboard.ForceSendFields, "SerializedDashboard")

	getPublishedForceSendFields := filterFields[dashboards.PublishedDashboard](publishedDashboard.ForceSendFields)

	return &resources.DashboardConfig{
		Dashboard:           *dashboard,
		EmbedCredentials:    publishedDashboard.EmbedCredentials,
		SerializedDashboard: dashboard.SerializedDashboard,
		ForceSendFields:     append(getDashboardForceSendFields, getPublishedForceSendFields...),
	}, nil
}

func isParentDoesntExistError(err error) bool {
	errStr := err.Error()
	return strings.HasPrefix(errStr, "Path (") && strings.HasSuffix(errStr, ") doesn't exist.")
}

func (r *ResourceDashboard) DoCreate(ctx context.Context, config *resources.DashboardConfig) (string, *resources.DashboardConfig, error) {
	// Fields like "embed_credentials" are part of the bundle configuration but not the create request here.
	// Thus we need to filter such fields out.
	config.Dashboard.ForceSendFields = filterFields[dashboards.Dashboard](config.Dashboard.ForceSendFields)

	// Set serialized dashboard in the create body request.
	createReq := config.Dashboard
	v := config.SerializedDashboard
	if _, ok := v.(string); ok {
		// If serialized dashboard is already a string, we can use it directly.
		createReq.SerializedDashboard = v.(string)
	} else if v != nil {
		// If it's inlined in the bundle config as a map, we need to marshal it to a string.
		b, err := json.Marshal(v)
		if err != nil {
			return "", nil, fmt.Errorf("failed to marshal serialized dashboard: %w", err)
		}
		createReq.SerializedDashboard = string(b)
	}

	createResp, err := r.client.Lakeview.Create(ctx, dashboards.CreateDashboardRequest{
		Dashboard: createReq,
	})
	// If the parent directory doesn't exist, create it and try again.
	if err != nil && isParentDoesntExistError(err) {
		err = r.client.Workspace.MkdirsByPath(ctx, config.ParentPath)
		if err != nil {
			return "", nil, fmt.Errorf("failed to create parent directory: %w", err)
		}
		createResp, err = r.client.Lakeview.Create(ctx, dashboards.CreateDashboardRequest{Dashboard: config.Dashboard})
	}
	if err != nil {
		return "", nil, err
	}

	// Persist the etag in state.
	config.Etag = createResp.Etag

	// embed_credentials as a zero valued default in resourcemutator/resource_mutator.go.
	// Thus we always need to include it in the ForceSendFields list to ensure that it is sent to the server.
	// TODO(followup): A more general solution that does not require this special casing.
	publishForceSendFields := filterFields[dashboards.PublishRequest](config.ForceSendFields)
	if !slices.Contains(publishForceSendFields, "EmbedCredentials") {
		publishForceSendFields = append(publishForceSendFields, "EmbedCredentials")
	}

	publishResp, err := r.client.Lakeview.Publish(ctx, dashboards.PublishRequest{
		DashboardId:      createResp.DashboardId,
		EmbedCredentials: config.EmbedCredentials,
		WarehouseId:      config.WarehouseId,
		ForceSendFields:  publishForceSendFields,
	})
	if err != nil {
		return "", nil, err
	}

	// Add the /Workspace prefix to the parent path. The backend removes this prefix from parent
	// path, and thus it needs to be added back in to match the local configuration.
	createResp.ParentPath = path.Join("/Workspace", createResp.ParentPath)

	// Set serialized dashboard to empty string in the [dashboards.Dashboard] struct.
	// Only the dashboard field in the [dashboard.DashboardConfig] struct is should be used.
	createResp.SerializedDashboard = ""
	createForceSendFields := filterFields[dashboards.Dashboard](createResp.ForceSendFields, "SerializedDashboard")

	// TODO: Add test for inline serialized dashboard. Is there a persistent drift?
	remoteState := &resources.DashboardConfig{
		Dashboard:        *createResp,
		EmbedCredentials: publishResp.EmbedCredentials,

		// Always store input serialized dashboard in state.
		// The serialized_dashboard field is only used for local diff computation
		// and thus should always be the local value.
		SerializedDashboard: config.SerializedDashboard,

		ForceSendFields: append(createForceSendFields, publishForceSendFields...),
	}

	return createResp.DashboardId, remoteState, nil
}

func (r *ResourceDashboard) DoUpdate(ctx context.Context, id string, config *resources.DashboardConfig) (*resources.DashboardConfig, error) {
	// Fields like "embed_credentials" are part of the bundle configuration but not the create request here.
	// Thus we need to filter such fields out.
	config.Dashboard.ForceSendFields = filterFields[dashboards.Dashboard](config.Dashboard.ForceSendFields)

	// Set serialized dashboard in the update body request.
	updateReqDashboard := config.Dashboard
	v := config.SerializedDashboard
	if _, ok := v.(string); ok {
		updateReqDashboard.SerializedDashboard = v.(string)
	} else if v != nil {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal serialized dashboard: %w", err)
		}
		updateReqDashboard.SerializedDashboard = string(b)
	}

	// Update the dashboard itself
	updateReq := dashboards.UpdateDashboardRequest{
		DashboardId: id,
		Dashboard:   updateReqDashboard,
	}

	updateResp, err := r.client.Lakeview.Update(ctx, updateReq)
	if err != nil {
		return nil, err
	}

	// embed_credentials as a zero valued default in resourcemutator/resource_mutator.go.
	// Thus we always need to include it in the ForceSendFields list to ensure that it is sent to the server.
	publishForceSendFields := filterFields[dashboards.PublishRequest](config.ForceSendFields)
	if !slices.Contains(publishForceSendFields, "EmbedCredentials") {
		publishForceSendFields = append(publishForceSendFields, "EmbedCredentials")
	}

	// Republish with potentially updated settings
	publishReq := dashboards.PublishRequest{
		DashboardId:      id,
		EmbedCredentials: config.EmbedCredentials,
		WarehouseId:      config.WarehouseId,
		ForceSendFields:  publishForceSendFields,
	}

	publishResp, err := r.client.Lakeview.Publish(ctx, publishReq)
	if err != nil {
		return nil, err
	}

	// Add the /Workspace prefix to the parent path. The backend removes this prefix from parent
	// path, and thus it needs to be added back in to match the local configuration.
	updateResp.ParentPath = path.Join("/Workspace", updateResp.ParentPath)

	// Unset serialized dashboard in the [dashboards.Dashboard] struct.
	// Only the serialized_dashboard field in the [dashboard.DashboardConfig] struct is should be used.
	updateResp.SerializedDashboard = ""
	updateForceSendFields := filterFields[dashboards.Dashboard](updateResp.ForceSendFields, "SerializedDashboard")

	remoteState := &resources.DashboardConfig{
		Dashboard:        *updateResp,
		EmbedCredentials: publishResp.EmbedCredentials,

		// Always store input serialized dashboard in state.
		// The serialized_dashboard field is only used for local diff computation
		// and thus should always be the local value.
		SerializedDashboard: config.SerializedDashboard,

		ForceSendFields: append(updateForceSendFields, publishForceSendFields...),
	}
	return remoteState, nil
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

		// Output only fields that should be ignored for diff computation.
		"create_time":     deployplan.ActionTypeSkip,
		"dashboard_id":    deployplan.ActionTypeSkip,
		"lifecycle_state": deployplan.ActionTypeSkip,
		"path":            deployplan.ActionTypeSkip,
		"update_time":     deployplan.ActionTypeSkip,
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

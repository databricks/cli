package dresources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	return &input.DashboardConfig
}

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

	return &resources.DashboardConfig{
		Dashboard:        *dashboard,
		EmbedCredentials: publishedDashboard.EmbedCredentials,
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
	v := config.SerializedDashboard
	if _, ok := v.(string); ok {
		// If serialized dashboard is already a string, we can use it directly.
		config.Dashboard.SerializedDashboard = v.(string)
	} else {
		// If it's inlined in the bundle config as a map, we need to marshal it to a string.
		b, err := json.Marshal(v)
		if err != nil {
			return "", nil, fmt.Errorf("failed to marshal serialized dashboard: %w", err)
		}
		config.Dashboard.SerializedDashboard = string(b)
	}

	createResp, err := r.client.Lakeview.Create(ctx, dashboards.CreateDashboardRequest{
		Dashboard: config.Dashboard,
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

	// TODO: Solve this more generally by adding all fields that have a default
	// value configured to the ForceSendFields list (see: resourcemutator/resource_mutator.go)
	// embed_credentials has a client side default value of false. We need to add it to the ForceSendFields list
	// to ensure that it is sent to the server.
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

	remoteState := &resources.DashboardConfig{
		Dashboard:        *createResp,
		EmbedCredentials: publishResp.EmbedCredentials,
	}
	return createResp.DashboardId, remoteState, nil
}

func (r *ResourceDashboard) DoUpdate(ctx context.Context, id string, config *resources.DashboardConfig) (*resources.DashboardConfig, error) {
	// Fields like "embed_credentials" are part of the bundle configuration but not the create request here.
	// Thus we need to filter such fields out.
	config.Dashboard.ForceSendFields = filterFields[dashboards.Dashboard](config.Dashboard.ForceSendFields)

	// Set serialized dashboard in the update body request. We
	v := config.SerializedDashboard
	if _, ok := v.(string); ok {
		config.Dashboard.SerializedDashboard = v.(string)
	} else {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal serialized dashboard: %w", err)
		}
		config.Dashboard.SerializedDashboard = string(b)
	}

	// Update the dashboard itself
	updateReq := dashboards.UpdateDashboardRequest{
		DashboardId: id,
		Dashboard:   config.Dashboard,
	}

	updateResp, err := r.client.Lakeview.Update(ctx, updateReq)
	if err != nil {
		return nil, err
	}

	// TODO: Solve this more generally by adding all fields that have a default
	// value configured to the ForceSendFields list (see: resourcemutator/resource_mutator.go)
	// embed_credentials has a client side default value of false. We need to add it to the ForceSendFields list
	// to ensure that it is sent to the server.
	publishForceSendFields := filterFields[dashboards.PublishRequest](config.ForceSendFields)
	if !slices.Contains(publishForceSendFields, "EmbedCredentials") {
		publishForceSendFields = append(publishForceSendFields, "EmbedCredentials")
	}

	// Republish with potentially updated settings
	publishReq := dashboards.PublishRequest{
		DashboardId:      id,
		EmbedCredentials: config.EmbedCredentials,
		WarehouseId:      config.WarehouseId,
		ForceSendFields:  filterFields[dashboards.PublishRequest](config.ForceSendFields),
	}

	publishResp, err := r.client.Lakeview.Publish(ctx, publishReq)
	if err != nil {
		return nil, err
	}

	remoteState := &resources.DashboardConfig{
		Dashboard:        *updateResp,
		EmbedCredentials: publishResp.EmbedCredentials,
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
	if isLocal {
		return map[string]deployplan.ActionType{
			// Etags are not relevant to determine if the local configuration changed.
			"etag": deployplan.ActionTypeSkip,

			"parent_path": deployplan.ActionTypeRecreate,

			// Output only fields that should be ignored for diff computation.
			"create_time":     deployplan.ActionTypeSkip,
			"dashboard_id":    deployplan.ActionTypeSkip,
			"lifecycle_state": deployplan.ActionTypeSkip,
			"path":            deployplan.ActionTypeSkip,
			"update_time":     deployplan.ActionTypeSkip,
		}
	}

	return map[string]deployplan.ActionType{
		// If the etag changes remotely, it means the dashboard has been modified remotely
		// and needs to be updated to match with the config.
		// Even though "update" action type is the default, we explicitly specify it here
		// to make this relationship clear.
		"etag": deployplan.ActionTypeUpdate,

		// "serialized_dashboard" locally and remotely will have different diffs. =
		// We only need to rely on etag here, and can skip this field for diff computation.
		"serialized_dashboard": deployplan.ActionTypeSkip,

		// Output only fields that should be ignored for diff computation.
		"create_time":     deployplan.ActionTypeSkip,
		"dashboard_id":    deployplan.ActionTypeSkip,
		"lifecycle_state": deployplan.ActionTypeSkip,
		"path":            deployplan.ActionTypeSkip,
		"update_time":     deployplan.ActionTypeSkip,
	}
}

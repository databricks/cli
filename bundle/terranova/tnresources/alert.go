package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

type ResourceAlert struct {
	client *databricks.WorkspaceClient
}

// New initializes a ResourceAlert with the given client.
func (*ResourceAlert) New(client *databricks.WorkspaceClient) *ResourceAlert {
	return &ResourceAlert{client: client}
}

// PrepareConfig converts bundle config to the SDK type.
func (*ResourceAlert) PrepareConfig(input *resources.Alert) *sql.AlertV2 {
	return &input.AlertV2
}

// DoRefresh reads the alert by id.
func (r *ResourceAlert) DoRefresh(ctx context.Context, id string) (*sql.AlertV2, error) {
	return r.client.AlertsV2.GetAlertById(ctx, id)
}

// DoCreate creates the alert and returns its id.
func (r *ResourceAlert) DoCreate(ctx context.Context, config *sql.AlertV2) (string, *sql.AlertV2, error) {
	request := sql.CreateAlertV2Request{
		Alert: *config,
	}
	response, err := r.client.AlertsV2.CreateAlert(ctx, request)
	if err != nil {
		return "", nil, err
	}
	return response.Id, response, nil
}

// DoUpdate updates the alert in place.
func (r *ResourceAlert) DoUpdate(ctx context.Context, id string, config *sql.AlertV2) (*sql.AlertV2, error) {
	request := sql.UpdateAlertV2Request{
		Id:         id,
		Alert:      *config,
		UpdateMask: "*",
	}
	response, err := r.client.AlertsV2.UpdateAlert(ctx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// DoDelete deletes the alert by id.
func (r *ResourceAlert) DoDelete(ctx context.Context, id string) error {
	return r.client.AlertsV2.TrashAlertById(ctx, id)
}

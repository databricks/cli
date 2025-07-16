package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

type ResourceAlert struct {
	client *databricks.WorkspaceClient
	config sql.AlertV2
}

func NewResourceAlert(client *databricks.WorkspaceClient, resource *resources.Alert) (*ResourceAlert, error) {
	return &ResourceAlert{
		client: client,
		config: resource.AlertV2,
	}, nil
}

func (r *ResourceAlert) Config() any {
	return r.config
}

func (r *ResourceAlert) DoCreate(ctx context.Context) (string, error) {
	request := sql.CreateAlertV2Request{
		Alert: r.config,
	}
	response, err := r.client.AlertsV2.CreateAlert(ctx, request)
	if err != nil {
		return "", SDKError{Method: "AlertsV2.CreateAlert", Err: err}
	}

	return response.Id, nil
}

func (r *ResourceAlert) DoUpdate(ctx context.Context, oldID string) (string, error) {
	request := sql.UpdateAlertV2Request{
		Id:         oldID,
		Alert:      r.config,
		UpdateMask: "*",
	}
	response, err := r.client.AlertsV2.UpdateAlert(ctx, request)
	if err != nil {
		return "", SDKError{Method: "AlertsV2.UpdateAlert", Err: err}
	}
	return response.Id, nil
}

func (r *ResourceAlert) WaitAfterCreate(ctx context.Context) error {
	// Alerts do not have a live status to wait for.
	return nil
}

func (r *ResourceAlert) WaitAfterUpdate(ctx context.Context) error {
	// Alerts do not have a live status to wait for.
	return nil
}

func (r *ResourceAlert) ClassifyChanges(changes []structdiff.Change) deployplan.ActionType {
	return deployplan.ActionTypeUpdate
}

func DeleteAlert(ctx context.Context, client *databricks.WorkspaceClient, oldID string) error {
	err := client.AlertsV2.TrashAlertById(ctx, oldID)
	if err != nil {
		return SDKError{Method: "AlertsV2.TrashAlertById", Err: err}
	}
	return nil
}

package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type ResourceVolume struct {
	client *databricks.WorkspaceClient
	config catalog.CreateVolumeRequestContent
}

func NewResourceVolume(client *databricks.WorkspaceClient, schema *resources.Volume) (*ResourceVolume, error) {
	return &ResourceVolume{
		client: client,
		config: schema.CreateVolumeRequestContent,
	}, nil
}

func (r *ResourceVolume) Config() any {
	return r.config
}

func (r *ResourceVolume) DoCreate(ctx context.Context) (string, error) {
	response, err := r.client.Volumes.Create(ctx, r.config)
	if err != nil {
		return "", SDKError{Method: "Volumes.Create", Err: err}
	}
	return response.FullName, nil
}

func (r *ResourceVolume) DoUpdate(ctx context.Context, id string) (string, error) {
	updateRequest := catalog.UpdateVolumeRequestContent{}
	err := copyViaJSON(&updateRequest, r.config)
	if err != nil {
		return "", err
	}

	updateRequest.Name = id

	response, err := r.client.Volumes.Update(ctx, updateRequest)
	if err != nil {
		return "", SDKError{Method: "Volumes.Update", Err: err}
	}

	return response.FullName, nil
}

func DeleteVolume(ctx context.Context, client *databricks.WorkspaceClient, id string) error {
	err := client.Volumes.DeleteByName(ctx, id)
	if err != nil {
		return SDKError{Method: "Volumes.Delete", Err: err}
	}
	return nil
}

func (r *ResourceVolume) WaitAfterCreate(ctx context.Context) error {
	// Intentional no-op
	return nil
}

func (r *ResourceVolume) WaitAfterUpdate(ctx context.Context) error {
	// Intentional no-op
	return nil
}

func (r *ResourceVolume) ClassifyChanges(changes []structdiff.Change) deployplan.ActionType {
	// TODO: Name, SchemaName changes should result in re-create
	return deployplan.ActionTypeUpdate
}

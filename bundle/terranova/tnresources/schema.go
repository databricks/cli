package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type ResourceSchema struct {
	client *databricks.WorkspaceClient
	config catalog.CreateSchema
}

func NewResourceSchema(client *databricks.WorkspaceClient, schema *resources.Schema) (*ResourceSchema, error) {
	return &ResourceSchema{
		client: client,
		config: schema.CreateSchema,
	}, nil
}

func (r *ResourceSchema) Config() any {
	return r.config
}

func (r *ResourceSchema) DoCreate(ctx context.Context) (string, error) {
	response, err := r.client.Schemas.Create(ctx, r.config)
	if err != nil {
		return "", SDKError{Method: "Schemas.Create", Err: err}
	}
	return response.FullName, nil
}

func (r *ResourceSchema) DoUpdate(ctx context.Context, id string) (string, error) {
	updateRequest := catalog.UpdateSchema{}
	err := copyViaJSON(&updateRequest, r.config)
	if err != nil {
		return "", err
	}

	updateRequest.FullName = id

	response, err := r.client.Schemas.Update(ctx, updateRequest)
	if err != nil {
		return "", SDKError{Method: "Schemas.Update", Err: err}
	}

	return response.FullName, nil
}

func DeleteSchema(ctx context.Context, client *databricks.WorkspaceClient, id string) error {
	// TODO: implement schema deletion
	return nil
}

func (r *ResourceSchema) WaitAfterCreate(ctx context.Context) error {
	// Intentional no-op
	return nil
}

func (r *ResourceSchema) WaitAfterUpdate(ctx context.Context) error {
	// Intentional no-op
	return nil
}

func (r *ResourceSchema) ClassifyChanges(changes []structdiff.Change) deployplan.ActionType {
	return deployplan.ActionTypeUpdate
}

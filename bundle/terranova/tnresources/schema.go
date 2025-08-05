package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
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

func (r *ResourceSchema) DoUpdate(ctx context.Context, id string) error {
	updateRequest := catalog.UpdateSchema{
		Comment:                      r.config.Comment,
		EnablePredictiveOptimization: "", // Not suppotred by DABs
		FullName:                     id,
		NewName:                      "", // We recreate schemas on name change intentionally.
		Owner:                        "", // Not supported by DABs
		Properties:                   r.config.Properties,
		ForceSendFields:              filterFields[catalog.UpdateSchema](r.config.ForceSendFields),
	}

	response, err := r.client.Schemas.Update(ctx, updateRequest)
	if err != nil {
		return SDKError{Method: "Schemas.Update", Err: err}
	}

	if response.FullName != id {
		log.Warnf(ctx, "schemas: response contains unexpected full_name=%#v (expected %#v)", response.FullName, id)
	}

	return nil
}

func DeleteSchema(ctx context.Context, client *databricks.WorkspaceClient, id string) error {
	err := client.Schemas.Delete(ctx, catalog.DeleteSchemaRequest{
		FullName:        id,
		Force:           true,
		ForceSendFields: nil, // Zero value for slice
	})
	if err != nil {
		return SDKError{Method: "Schemas.Delete", Err: err}
	}
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

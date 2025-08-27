package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type ResourceSchema struct {
	client      *databricks.WorkspaceClient
	config      catalog.CreateSchema
	remoteState *catalog.SchemaInfo
}

func NewResourceSchema(client *databricks.WorkspaceClient, schema *resources.Schema) (*ResourceSchema, error) {
	return &ResourceSchema{
		client:      client,
		config:      schema.CreateSchema,
		remoteState: nil,
	}, nil
}

func (r *ResourceSchema) Config() any {
	return r.config
}

func (r *ResourceSchema) RemoteState() any {
	return r.remoteState
}

func (r *ResourceSchema) RemoteStateAsConfig() any {
	if r.remoteState == nil {
		return nil
	}
	return catalog.CreateSchema{
		CatalogName: r.remoteState.CatalogName,
		Comment:     r.remoteState.Comment,
		Name:        r.remoteState.Name,
		Properties:  r.remoteState.Properties,
		StorageRoot: r.remoteState.StorageRoot,
	}
}

func (r *ResourceSchema) DoRefresh(ctx context.Context, id string) error {
	response, err := r.client.Schemas.GetByFullName(ctx, id)
	if err != nil {
		return err
	}
	r.remoteState = response
	return nil
}

func (r *ResourceSchema) DoCreateWithRefresh(ctx context.Context) (string, error) {
	response, err := r.client.Schemas.Create(ctx, r.config)
	if err != nil {
		return "", err
	}
	r.remoteState = response
	return response.FullName, nil
}

func (r *ResourceSchema) DoUpdateWithRefresh(ctx context.Context, id string) error {
	updateRequest := catalog.UpdateSchema{
		Comment:                      r.config.Comment,
		EnablePredictiveOptimization: "", // Not supported by DABs
		FullName:                     id,
		NewName:                      "", // We recreate schemas on name change intentionally.
		Owner:                        "", // Not supported by DABs
		Properties:                   r.config.Properties,
		ForceSendFields:              filterFields[catalog.UpdateSchema](r.config.ForceSendFields),
	}

	response, err := r.client.Schemas.Update(ctx, updateRequest)
	if err != nil {
		return err
	}

	if response.FullName != id {
		log.Warnf(ctx, "schemas: response contains unexpected full_name=%#v (expected %#v)", response.FullName, id)
	}

	r.remoteState = response
	return nil
}

func DeleteSchema(ctx context.Context, client *databricks.WorkspaceClient, id string) error {
	return client.Schemas.Delete(ctx, catalog.DeleteSchemaRequest{
		FullName:        id,
		Force:           true,
		ForceSendFields: nil,
	})
}

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
}

func (*ResourceSchema) New(client *databricks.WorkspaceClient) *ResourceSchema {
	return &ResourceSchema{client: client}
}

func (*ResourceSchema) PrepareConfig(input *resources.Schema) *catalog.CreateSchema {
	return &input.CreateSchema
}

func (r *ResourceSchema) DoCreate(ctx context.Context, config *catalog.CreateSchema) (string, error) {
	response, err := r.client.Schemas.Create(ctx, *config)
	if err != nil || response == nil {
		return "", err
	}
	return response.FullName, nil
}

func (r *ResourceSchema) DoUpdate(ctx context.Context, id string, config *catalog.CreateSchema) error {
	updateRequest := catalog.UpdateSchema{
		Comment:                      config.Comment,
		EnablePredictiveOptimization: "", // Not supported by DABs
		FullName:                     id,
		NewName:                      "", // We recreate schemas on name change intentionally.
		Owner:                        "", // Not supported by DABs
		Properties:                   config.Properties,
		ForceSendFields:              filterFields[catalog.UpdateSchema](config.ForceSendFields),
	}

	response, err := r.client.Schemas.Update(ctx, updateRequest)
	if err != nil {
		return err
	}

	if response != nil && response.FullName != id {
		log.Warnf(ctx, "schemas: response contains unexpected full_name=%#v (expected %#v)", response.FullName, id)
	}

	return nil
}

func (r *ResourceSchema) DoDelete(ctx context.Context, id string) error {
	return r.client.Schemas.Delete(ctx, catalog.DeleteSchemaRequest{
		FullName:        id,
		Force:           true,
		ForceSendFields: nil,
	})
}

func (*ResourceSchema) RecreateFields() []string {
	return []string{
		".name",
		".catalog_name",
		".storage_root",
	}
}

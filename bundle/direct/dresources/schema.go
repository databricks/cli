package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
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

func (*ResourceSchema) PrepareState(input *resources.Schema) *catalog.CreateSchema {
	return &input.CreateSchema
}

func (*ResourceSchema) RemapState(info *catalog.SchemaInfo) *catalog.CreateSchema {
	return &catalog.CreateSchema{
		CatalogName:     info.CatalogName,
		Comment:         info.Comment,
		Name:            info.Name,
		Properties:      info.Properties,
		StorageRoot:     info.StorageRoot,
		ForceSendFields: filterFields[catalog.CreateSchema](info.ForceSendFields),
	}
}

func (r *ResourceSchema) DoRefresh(ctx context.Context, id string) (*catalog.SchemaInfo, error) {
	return r.client.Schemas.GetByFullName(ctx, id)
}

func (r *ResourceSchema) DoCreate(ctx context.Context, config *catalog.CreateSchema) (string, *catalog.SchemaInfo, error) {
	response, err := r.client.Schemas.Create(ctx, *config)
	if err != nil || response == nil {
		return "", nil, err
	}
	return response.FullName, response, nil
}

// DoUpdate updates the schema in place and returns remote state.
func (r *ResourceSchema) DoUpdate(ctx context.Context, id string, config *catalog.CreateSchema) (*catalog.SchemaInfo, error) {
	updateRequest := catalog.UpdateSchema{
		Comment:                      config.Comment,
		EnablePredictiveOptimization: "", // Not supported by DABs
		FullName:                     id,
		NewName:                      "", // We recreate schemas on name change intentionally.
		Owner:                        "", // Not supported by DABs
		Properties:                   config.Properties,
		ForceSendFields:              filterFields[catalog.UpdateSchema](config.ForceSendFields, "EnablePredictiveOptimization", "NewName", "Owner"),
	}

	response, err := r.client.Schemas.Update(ctx, updateRequest)
	if err != nil {
		return nil, err
	}

	if response != nil && response.FullName != id {
		log.Warnf(ctx, "schemas: response contains unexpected full_name=%#v (expected %#v)", response.FullName, id)
	}

	return response, nil
}

func (r *ResourceSchema) DoDelete(ctx context.Context, id string) error {
	return r.client.Schemas.Delete(ctx, catalog.DeleteSchemaRequest{
		FullName:        id,
		Force:           true,
		ForceSendFields: nil,
	})
}

func (*ResourceSchema) FieldTriggers() map[string]deployplan.ActionType {
	return map[string]deployplan.ActionType{
		"name":         deployplan.ActionTypeRecreate,
		"catalog_name": deployplan.ActionTypeRecreate,
		"storage_root": deployplan.ActionTypeRecreate,
	}
}

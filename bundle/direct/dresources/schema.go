package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sdkclients"
	schemas "github.com/databricks/cli/libs/sdktypes/uc/schemas/v1"
	"github.com/databricks/cli/libs/utils"
	sdkschemas "github.com/databricks/sdk-go/uc/schemas/v1"
)

type ResourceSchema struct {
	client *sdkclients.Clients
}

func (*ResourceSchema) New(client *sdkclients.Clients) *ResourceSchema {
	return &ResourceSchema{client: client}
}

func (*ResourceSchema) PrepareState(input *resources.Schema) *resources.CreateSchema {
	return &input.CreateSchema
}

func (*ResourceSchema) RemapState(info *schemas.SchemaInfo) *resources.CreateSchema {
	return &resources.CreateSchema{
		CatalogName:             info.CatalogName,
		Comment:                 info.Comment,
		CustomMaxRetentionHours: info.CustomMaxRetentionHours,
		Name:                    info.Name,
		Properties:              info.Properties,
		StorageRoot:             info.StorageRoot,
	}
}

func (r *ResourceSchema) DoRead(ctx context.Context, id string) (*schemas.SchemaInfo, error) {
	info, err := r.client.Schemas.GetSchema(ctx, sdkschemas.GetSchemaRequest{
		FullNameArg:   new(id),
		IncludeBrowse: nil, // Not set by DABs
	})
	if err != nil {
		return nil, err
	}
	return schemas.SchemaInfoFromSDK(info)
}

func (r *ResourceSchema) DoCreate(ctx context.Context, config *resources.CreateSchema) (string, *schemas.SchemaInfo, error) {
	request, err := toSDK[sdkschemas.CreateSchemaRequest](config.AsRequest())
	if err != nil {
		return "", nil, err
	}
	response, err := r.client.Schemas.CreateSchema(ctx, request)
	if err != nil || response == nil {
		return "", nil, err
	}
	info, err := schemas.SchemaInfoFromSDK(response)
	if err != nil {
		return "", nil, err
	}
	return utils.Deref(info.FullName), info, nil
}

// UC drops a field the payload omits, so a value the config stops declaring never
// reaches the backend and every later plan reports the same pending change. Worse, when
// nothing else is in the payload UC rejects it outright with "Nothing to update" (400),
// which made a schema undeployable once someone set a description on it outside the
// bundle. Sending these explicitly is the only way to express "the bundle owns this".
//
// Only fields verified against a real workspace are always sent: {"comment": ""} and
// {"custom_max_retention_hours": 0} both clear. Do not add a field without probing it -- UC rejects
// an empty value on some (owner: "" is "Could not find principal with name .", new_name: "" is "not
// a valid name").
//
// custom_max_retention_hours is always sent so the direct engine can clear it. The terraform
// databricks_schema resource cannot set this field at all, so a schema config that uses it deploys
// differently under terraform (which drops it) than under direct; such configs are kept out of the
// migrate invariant (schema.yml is excluded there, and schema_tf_compat.yml covers migrate instead).
// DoUpdate updates the schema in place and returns remote state.
func (r *ResourceSchema) DoUpdate(ctx context.Context, id string, config *resources.CreateSchema, _ *PlanEntry) (*schemas.SchemaInfo, error) {
	// new_name is not set: schemas are recreated on name change intentionally.
	// owner and enable_predictive_optimization are not supported by DABs.
	request, err := toSDK[sdkschemas.UpdateSchemaRequest](&schemas.UpdateSchemaRequest{
		FullNameArg:                         new(id),
		Comment:                             orZero(config.Comment),
		CustomMaxRetentionHours:             orZero(config.CustomMaxRetentionHours),
		Properties:                          config.Properties,
		BrowseOnly:                          nil, // Not set by DABs
		CatalogName:                         nil, // Not set by DABs
		CatalogType:                         "",  // Not set by DABs
		CreatedAt:                           nil, // Not set by DABs
		CreatedBy:                           nil, // Not set by DABs
		EffectivePredictiveOptimizationFlag: nil, // Not set by DABs
		EnablePredictiveOptimization:        nil, // Not set by DABs
		FullName:                            nil, // Not set by DABs
		MetastoreId:                         nil, // Not set by DABs
		Name:                                nil, // Not set by DABs
		NewName:                             nil, // Not set by DABs
		Options:                             nil, // Not set by DABs
		Owner:                               nil, // Not set by DABs
		SchemaId:                            nil, // Not set by DABs
		StorageLocation:                     nil, // Not set by DABs
		StorageRoot:                         nil, // Not set by DABs
		UpdatedAt:                           nil, // Not set by DABs
		UpdatedBy:                           nil, // Not set by DABs
	})
	if err != nil {
		return nil, err
	}
	response, err := r.client.Schemas.UpdateSchema(ctx, request)
	if err != nil {
		return nil, err
	}

	if response != nil && utils.Deref(response.FullName) != id {
		log.Warnf(ctx, "schemas: response contains unexpected full_name=%#v (expected %#v)", utils.Deref(response.FullName), id)
	}

	return schemas.SchemaInfoFromSDK(response)
}

func (r *ResourceSchema) DoDelete(ctx context.Context, id string, _ *resources.CreateSchema) error {
	_, err := r.client.Schemas.DeleteSchema(ctx, sdkschemas.DeleteSchemaRequest{
		FullNameArg: new(id),
		Force:       new(true),
	})
	return err
}

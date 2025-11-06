package dresources

import (
	"context"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

type ResourceModelServingEndpoint struct {
	client *databricks.WorkspaceClient
}

func (*ResourceModelServingEndpoint) New(client *databricks.WorkspaceClient) *ResourceModelServingEndpoint {
	return &ResourceModelServingEndpoint{
		client: client,
	}
}

func (*ResourceModelServingEndpoint) PrepareState(input *resources.ModelServingEndpoint) *serving.CreateServingEndpoint {
	return &input.CreateServingEndpoint
}

func autoCaptureConfigOutputToInput(output *serving.AutoCaptureConfigOutput) *serving.AutoCaptureConfigInput {
	if output == nil {
		return nil
	}
	return &serving.AutoCaptureConfigInput{
		CatalogName:     output.CatalogName,
		SchemaName:      output.SchemaName,
		TableNamePrefix: output.TableNamePrefix,
		Enabled:         output.Enabled,
		ForceSendFields: filterFields[serving.AutoCaptureConfigInput](output.ForceSendFields),
	}
}

func servedEntitiesOutputToInput(output []serving.ServedEntityOutput) []serving.ServedEntityInput {
	if len(output) == 0 {
		return nil
	}
	entities := make([]serving.ServedEntityInput, len(output))
	for i, entity := range output {
		entities[i] = serving.ServedEntityInput{
			EntityName:                entity.EntityName,
			EntityVersion:             entity.EntityVersion,
			EnvironmentVars:           entity.EnvironmentVars,
			ExternalModel:             entity.ExternalModel,
			InstanceProfileArn:        entity.InstanceProfileArn,
			MaxProvisionedConcurrency: entity.MaxProvisionedConcurrency,
			MaxProvisionedThroughput:  entity.MaxProvisionedThroughput,
			MinProvisionedConcurrency: entity.MinProvisionedConcurrency,
			MinProvisionedThroughput:  entity.MinProvisionedThroughput,
			Name:                      entity.Name,
			ProvisionedModelUnits:     entity.ProvisionedModelUnits,
			ScaleToZeroEnabled:        entity.ScaleToZeroEnabled,
			WorkloadSize:              entity.WorkloadSize,
			WorkloadType:              entity.WorkloadType,
			ForceSendFields:           filterFields[serving.ServedEntityInput](entity.ForceSendFields),
		}
	}

	return entities
}

func configOutputToInput(output *serving.EndpointCoreConfigOutput) *serving.EndpointCoreConfigInput {
	if output == nil {
		return nil
	}
	return &serving.EndpointCoreConfigInput{
		AutoCaptureConfig: autoCaptureConfigOutputToInput(output.AutoCaptureConfig),
		ServedEntities:    servedEntitiesOutputToInput(output.ServedEntities),
	}
}

// TODO: Remap served_models to served_entities.
func (*ResourceModelServingEndpoint) RemapState(endpoint *serving.ServingEndpointDetailed) *serving.CreateServingEndpoint {
	// Map the remote state (ServingEndpointDetailed) to the local state (CreateServingEndpoint)
	// for proper comparison during diff calculation
	return &serving.CreateServingEndpoint{
		AiGateway:          endpoint.AiGateway,
		BudgetPolicyId:     endpoint.BudgetPolicyId,
		Config:             configOutputToInput(endpoint.Config),
		Description:        endpoint.Description,
		EmailNotifications: endpoint.EmailNotifications,
		Name:               endpoint.Name,
		RouteOptimized:     endpoint.RouteOptimized,
		Tags:               endpoint.Tags,
		ForceSendFields:    filterFields[serving.CreateServingEndpoint](endpoint.ForceSendFields),
	}
}

func (r *ResourceModelServingEndpoint) DoRefresh(ctx context.Context, id string) (*serving.ServingEndpointDetailed, error) {
	return r.client.ServingEndpoints.GetByName(ctx, id)
}

func (r *ResourceModelServingEndpoint) DoCreate(ctx context.Context, config *serving.CreateServingEndpoint) (string, error) {
	waiter, err := r.client.ServingEndpoints.Create(ctx, *config)
	if err != nil {
		return "", err
	}

	return waiter.Response.Name, nil
}

// waitForEndpointReady waits for the serving endpoint to be ready (not updating)
func (r *ResourceModelServingEndpoint) waitForEndpointReady(ctx context.Context, name string) (*serving.ServingEndpointDetailed, error) {
	waiter := &serving.WaitGetServingEndpointNotUpdating[serving.ServingEndpointDetailed]{
		Response: &serving.ServingEndpointDetailed{Name: name},
		Name:     name,
		Poll: func(timeout time.Duration, callback func(*serving.ServingEndpointDetailed)) (*serving.ServingEndpointDetailed, error) {
			return r.client.ServingEndpoints.WaitGetServingEndpointNotUpdating(ctx, name, timeout, callback)
		},
	}

	// Model serving endpoints can take a long time to spin up. We match the timeout from TF here (35 minutes).
	return waiter.GetWithTimeout(35 * time.Minute)
}

func (r *ResourceModelServingEndpoint) WaitAfterCreate(ctx context.Context, config *serving.CreateServingEndpoint) (*serving.ServingEndpointDetailed, error) {
	return r.waitForEndpointReady(ctx, config.Name)
}

func (r *ResourceModelServingEndpoint) DoUpdate(ctx context.Context, id string, config *serving.CreateServingEndpoint) error {
	if config.Config == nil {
		return nil
	}

	// UpdateConfig expects an EndpointCoreConfigInput with the Name field set
	updateConfig := *config.Config
	updateConfig.Name = id

	_, err := r.client.ServingEndpoints.UpdateConfig(ctx, updateConfig)
	return err
}

func (r *ResourceModelServingEndpoint) WaitAfterUpdate(ctx context.Context, config *serving.CreateServingEndpoint) (*serving.ServingEndpointDetailed, error) {
	return r.waitForEndpointReady(ctx, config.Name)
}

func (r *ResourceModelServingEndpoint) DoDelete(ctx context.Context, id string) error {
	return r.client.ServingEndpoints.DeleteByName(ctx, id)
}

func (*ResourceModelServingEndpoint) FieldTriggers(_ bool) map[string]deployplan.ActionType {
	// TF implementation: https://github.com/databricks/terraform-provider-databricks/blob/6c106e8e7052bb2726148d66309fd460ed444236/mlflow/resource_mlflow_experiment.go#L22
	return map[string]deployplan.ActionType{
		"name": deployplan.ActionTypeRecreate,
		"config.auto_capture_config.catalog_name":      deployplan.ActionTypeRecreate,
		"config.auto_capture_config.schema_name":       deployplan.ActionTypeRecreate,
		"config.auto_capture_config.table_name_prefix": deployplan.ActionTypeRecreate,
		"route_optimized":                              deployplan.ActionTypeRecreate,
	}
}

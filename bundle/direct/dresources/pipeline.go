package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type ResourcePipeline struct {
	client *databricks.WorkspaceClient
}

func (*ResourcePipeline) New(client *databricks.WorkspaceClient) *ResourcePipeline {
	return &ResourcePipeline{
		client: client,
	}
}

func (*ResourcePipeline) PrepareState(input *resources.Pipeline) *pipelines.CreatePipeline {
	return &input.CreatePipeline
}

func (*ResourcePipeline) RemapState(p *pipelines.GetPipelineResponse) *pipelines.CreatePipeline {
	spec := p.Spec
	return &pipelines.CreatePipeline{
		// TODO: Fields that are not available in GetPipelineResponse (like AllowDuplicateNames) should be added to resource's ignore_remote_changes list so that they never produce a call to action
		AllowDuplicateNames: false,
		BudgetPolicyId:      spec.BudgetPolicyId,
		Catalog:             spec.Catalog,
		Channel:             spec.Channel,
		Clusters:            spec.Clusters,
		Configuration:       spec.Configuration,
		Continuous:          spec.Continuous,
		Deployment:          spec.Deployment,
		Development:         spec.Development,
		DryRun:              false,
		Edition:             spec.Edition,
		Environment:         spec.Environment,
		EventLog:            spec.EventLog,
		Filters:             spec.Filters,
		GatewayDefinition:   spec.GatewayDefinition,
		// Clear "id" field, otherwise it shows up in changes.remote
		Id:                  "",
		IngestionDefinition: spec.IngestionDefinition,
		Libraries:           spec.Libraries,
		Name:                spec.Name,
		Notifications:       spec.Notifications,
		Photon:              spec.Photon,
		RestartWindow:       spec.RestartWindow,
		RootPath:            spec.RootPath,
		RunAs:               p.RunAs,
		Schema:              spec.Schema,
		Serverless:          spec.Serverless,
		Storage:             spec.Storage,
		Tags:                spec.Tags,
		Target:              spec.Target,
		Trigger:             spec.Trigger,
		UsagePolicyId:       spec.UsagePolicyId,
		ForceSendFields:     utils.FilterFields[pipelines.CreatePipeline](spec.ForceSendFields, "AllowDuplicateNames", "DryRun", "RunAs", "Id"),
	}
}

func (r *ResourcePipeline) DoRead(ctx context.Context, id string) (*pipelines.GetPipelineResponse, error) {
	return r.client.Pipelines.GetByPipelineId(ctx, id)
}

func (r *ResourcePipeline) DoCreate(ctx context.Context, config *pipelines.CreatePipeline) (string, *pipelines.GetPipelineResponse, error) {
	response, err := r.client.Pipelines.Create(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	return response.PipelineId, nil, nil
}

func (r *ResourcePipeline) DoUpdate(ctx context.Context, id string, config *pipelines.CreatePipeline, _ Changes) (*pipelines.GetPipelineResponse, error) {
	request := pipelines.EditPipeline{
		AllowDuplicateNames:  config.AllowDuplicateNames,
		BudgetPolicyId:       config.BudgetPolicyId,
		Catalog:              config.Catalog,
		Channel:              config.Channel,
		Clusters:             config.Clusters,
		Configuration:        config.Configuration,
		Continuous:           config.Continuous,
		Deployment:           config.Deployment,
		Development:          config.Development,
		Edition:              config.Edition,
		Environment:          config.Environment,
		EventLog:             config.EventLog,
		ExpectedLastModified: 0,
		Filters:              config.Filters,
		GatewayDefinition:    config.GatewayDefinition,
		Id:                   config.Id,
		IngestionDefinition:  config.IngestionDefinition,
		Libraries:            config.Libraries,
		Name:                 config.Name,
		Notifications:        config.Notifications,
		Photon:               config.Photon,
		RestartWindow:        config.RestartWindow,
		RootPath:             config.RootPath,
		RunAs:                config.RunAs,
		Schema:               config.Schema,
		Serverless:           config.Serverless,
		Storage:              config.Storage,
		Tags:                 config.Tags,
		Target:               config.Target,
		Trigger:              config.Trigger,
		UsagePolicyId:        config.UsagePolicyId,
		PipelineId:           id,
		ForceSendFields:      utils.FilterFields[pipelines.EditPipeline](config.ForceSendFields),
	}

	return nil, r.client.Pipelines.Update(ctx, request)
}

func (r *ResourcePipeline) DoDelete(ctx context.Context, id string) error {
	return r.client.Pipelines.DeleteByPipelineId(ctx, id)
}

func (*ResourcePipeline) FieldTriggers() map[string]deployplan.ActionType {
	result := map[string]deployplan.ActionType{
		"storage":                                   deployplan.Recreate,
		"ingestion_definition.connection_name":      deployplan.Recreate,
		"ingestion_definition.ingestion_gateway_id": deployplan.Recreate,
	}

	return result
}

func (*ResourcePipeline) OverrideChangeDesc(ctx context.Context, path *structpath.PathNode, ch *ChangeDesc, _ *pipelines.GetPipelineResponse) error {
	if path.String() == "run_as" {
		if structdiff.IsEqual(ch.Old, ch.New) {
			ch.Action = deployplan.Skip
			ch.Reason = "override"
		}
	}

	return nil
}

// Note, terraform provider either
// a) reads back state at least once and fails create if state is "failed"
// b) repeatededly reads state until state is "running" (if spec.Contionous is set).
// TODO: investigate if we need to mimic this behaviour or can rely on Create status code.

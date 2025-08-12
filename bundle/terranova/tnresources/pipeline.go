package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type ResourcePipeline struct {
	client *databricks.WorkspaceClient
	config pipelines.CreatePipeline
}

func NewResourcePipeline(client *databricks.WorkspaceClient, resource *resources.Pipeline) (*ResourcePipeline, error) {
	return &ResourcePipeline{
		client: client,
		config: resource.CreatePipeline,
	}, nil
}

func (r *ResourcePipeline) Config() any {
	return r.config
}

func (r *ResourcePipeline) DoCreate(ctx context.Context) (string, error) {
	response, err := r.client.Pipelines.Create(ctx, r.config)
	if err != nil {
		return "", SDKError{Method: "Pipelines.Create", Err: err}
	}
	return response.PipelineId, nil
}

func (r *ResourcePipeline) DoUpdate(ctx context.Context, id string) error {
	request := pipelines.EditPipeline{
		AllowDuplicateNames:  r.config.AllowDuplicateNames,
		BudgetPolicyId:       r.config.BudgetPolicyId,
		Catalog:              r.config.Catalog,
		Channel:              r.config.Channel,
		Clusters:             r.config.Clusters,
		Configuration:        r.config.Configuration,
		Continuous:           r.config.Continuous,
		Deployment:           r.config.Deployment,
		Development:          r.config.Development,
		Edition:              r.config.Edition,
		Environment:          r.config.Environment,
		EventLog:             r.config.EventLog,
		ExpectedLastModified: 0,
		Filters:              r.config.Filters,
		GatewayDefinition:    r.config.GatewayDefinition,
		Id:                   r.config.Id,
		IngestionDefinition:  r.config.IngestionDefinition,
		Libraries:            r.config.Libraries,
		Name:                 r.config.Name,
		Notifications:        r.config.Notifications,
		Photon:               r.config.Photon,
		RestartWindow:        r.config.RestartWindow,
		RootPath:             r.config.RootPath,
		RunAs:                r.config.RunAs,
		Schema:               r.config.Schema,
		Serverless:           r.config.Serverless,
		Storage:              r.config.Storage,
		Tags:                 r.config.Tags,
		Target:               r.config.Target,
		Trigger:              r.config.Trigger,
		PipelineId:           id,
		ForceSendFields:      filterFields[pipelines.EditPipeline](r.config.ForceSendFields),
	}

	err := r.client.Pipelines.Update(ctx, request)
	if err != nil {
		return SDKError{Method: "Pipelines.Update", Err: err}
	}
	return nil
}

func DeletePipeline(ctx context.Context, client *databricks.WorkspaceClient, id string) error {
	err := client.Pipelines.DeleteByPipelineId(ctx, id)
	if err != nil {
		return SDKError{Method: "Pipelines.DeleteByPipelineId", Err: err}
	}
	return nil
}

func (r *ResourcePipeline) WaitAfterCreate(ctx context.Context) error {
	// Note, terraform provider either
	// a) reads back state at least once and fails create if state is "failed"
	// b) repeatededly reads state until state is "running" (if spec.Contionous is set).
	// TODO: investigate if we need to mimic this behaviour or can rely on Create status code.
	return nil
}

func (r *ResourcePipeline) WaitAfterUpdate(ctx context.Context) error {
	// TODO: investigate if we need to mimic waiting behaviour in TF or can rely on Update status code.
	return nil
}

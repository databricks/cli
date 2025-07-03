package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structdiff"
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

func (r *ResourcePipeline) DoUpdate(ctx context.Context, id string) (string, error) {
	request := pipelines.EditPipeline{}
	err := copyViaJSON(&request, r.config)
	if err != nil {
		return "", err
	}
	request.PipelineId = id

	err = r.client.Pipelines.Update(ctx, request)
	if err != nil {
		return "", SDKError{Method: "Pipelines.Update", Err: err}
	}
	return id, nil
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

func (r *ResourcePipeline) ClassifyChanges(changes []structdiff.Change) deployplan.ActionType {
	return deployplan.ActionTypeUpdate
}

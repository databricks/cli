package terranova_resources

import (
	"context"
	"reflect"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type ResourcePipeline struct {
	client *databricks.WorkspaceClient
	config pipelines.CreatePipeline
}

func NewResourcePipeline(client *databricks.WorkspaceClient, resource resources.Pipeline) (ResourcePipeline, error) {
	return ResourcePipeline{
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
		return "", SDKError{Method: "Pipelines.Updater", Err: err}
	}
	return id, nil
}

func (r *ResourcePipeline) DoDelete(ctx context.Context, id string) error {
	return nil
}

func (r *ResourcePipeline) WaitAfterCreate(ctx context.Context) error {
	return nil
}

func (r *ResourcePipeline) WaitAfterUpdate(ctx context.Context) error {
	return nil
}

func (r *ResourcePipeline) ClassifyChanges(changes []structdiff.Change) ChangeType {
	return ChangeTypeUpdate
}

var pipelineType = reflect.TypeOf(ResourcePipeline{}.config)

func (r *ResourcePipeline) GetType() reflect.Type {
	return pipelineType
}

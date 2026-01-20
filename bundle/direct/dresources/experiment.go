package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type ResourceExperiment struct {
	client *databricks.WorkspaceClient
}

func (*ResourceExperiment) New(client *databricks.WorkspaceClient) *ResourceExperiment {
	return &ResourceExperiment{
		client: client,
	}
}

func (*ResourceExperiment) PrepareState(input *resources.MlflowExperiment) *ml.CreateExperiment {
	return &ml.CreateExperiment{
		Name:             input.Name,
		ArtifactLocation: input.ArtifactLocation,
		Tags:             input.Tags,
		ForceSendFields:  utils.FilterFields[ml.CreateExperiment](input.ForceSendFields),
	}
}

func (*ResourceExperiment) RemapState(experiment *ml.Experiment) *ml.CreateExperiment {
	return &ml.CreateExperiment{
		Name:             experiment.Name,
		ArtifactLocation: experiment.ArtifactLocation,
		Tags:             experiment.Tags,
		ForceSendFields:  utils.FilterFields[ml.CreateExperiment](experiment.ForceSendFields),
	}
}

func (r *ResourceExperiment) DoRead(ctx context.Context, id string) (*ml.Experiment, error) {
	result, err := r.client.Experiments.GetExperiment(ctx, ml.GetExperimentRequest{
		ExperimentId: id,
	})
	if err != nil {
		return nil, err
	}
	return result.Experiment, nil
}

func (r *ResourceExperiment) DoCreate(ctx context.Context, config *ml.CreateExperiment) (string, *ml.Experiment, error) {
	result, err := r.client.Experiments.CreateExperiment(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	return result.ExperimentId, nil, nil
}

func (r *ResourceExperiment) DoUpdate(ctx context.Context, id string, config *ml.CreateExperiment, _ Changes) (*ml.Experiment, error) {
	updateReq := ml.UpdateExperiment{
		ExperimentId:    id,
		NewName:         config.Name,
		ForceSendFields: utils.FilterFields[ml.UpdateExperiment](config.ForceSendFields),
	}

	return nil, r.client.Experiments.UpdateExperiment(ctx, updateReq)
}

func (r *ResourceExperiment) DoDelete(ctx context.Context, id string) error {
	return r.client.Experiments.DeleteExperiment(ctx, ml.DeleteExperiment{
		ExperimentId: id,
	})
}

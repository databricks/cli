package dresources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
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
		ForceSendFields:  filterFields[ml.CreateExperiment](input.ForceSendFields),
	}
}

func (*ResourceExperiment) RemapState(experiment *ml.Experiment) *ml.CreateExperiment {
	return &ml.CreateExperiment{
		Name:             experiment.Name,
		ArtifactLocation: experiment.ArtifactLocation,
		Tags:             experiment.Tags,
		ForceSendFields:  filterFields[ml.CreateExperiment](experiment.ForceSendFields),
	}
}

func (r *ResourceExperiment) DoRefresh(ctx context.Context, id string) (*ml.Experiment, error) {
	result, err := r.client.Experiments.GetExperiment(ctx, ml.GetExperimentRequest{
		ExperimentId: id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment %s: %w", id, err)
	}
	return result.Experiment, nil
}

func (r *ResourceExperiment) DoCreate(ctx context.Context, config *ml.CreateExperiment) (string, error) {
	result, err := r.client.Experiments.CreateExperiment(ctx, *config)
	if err != nil {
		return "", fmt.Errorf("failed to create experiment: %w", err)
	}
	return result.ExperimentId, nil
}

func (r *ResourceExperiment) DoUpdate(ctx context.Context, id string, config *ml.CreateExperiment) error {
	updateReq := ml.UpdateExperiment{
		ExperimentId:    id,
		NewName:         config.Name,
		ForceSendFields: filterFields[ml.UpdateExperiment](config.ForceSendFields),
	}

	err := r.client.Experiments.UpdateExperiment(ctx, updateReq)
	if err != nil {
		return fmt.Errorf("failed to update experiment %s: %w", id, err)
	}
	return nil
}

func (r *ResourceExperiment) DoDelete(ctx context.Context, id string) error {
	err := r.client.Experiments.DeleteExperiment(ctx, ml.DeleteExperiment{
		ExperimentId: id,
	})
	if err != nil {
		return fmt.Errorf("failed to delete experiment %s: %w", id, err)
	}
	return nil
}

func (*ResourceExperiment) FieldTriggers() map[string]deployplan.ActionType {
	return map[string]deployplan.ActionType{
		"name":              deployplan.ActionTypeUpdate,
		"artifact_location": deployplan.ActionTypeRecreate,

		// Tags updates are not supported by TF. This mirrors that behaviour.
		"tags": deployplan.ActionTypeSkip,
	}
}

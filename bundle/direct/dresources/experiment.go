package dresources

import (
	"context"

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
		return nil, err
	}
	return result.Experiment, nil
}

func (r *ResourceExperiment) DoCreate(ctx context.Context, config *ml.CreateExperiment) (string, error) {
	result, err := r.client.Experiments.CreateExperiment(ctx, *config)
	if err != nil {
		return "", err
	}
	return result.ExperimentId, nil
}

func (r *ResourceExperiment) DoUpdate(ctx context.Context, id string, config *ml.CreateExperiment) error {
	updateReq := ml.UpdateExperiment{
		ExperimentId:    id,
		NewName:         config.Name,
		ForceSendFields: filterFields[ml.UpdateExperiment](config.ForceSendFields),
	}

	return r.client.Experiments.UpdateExperiment(ctx, updateReq)
}

func (r *ResourceExperiment) DoDelete(ctx context.Context, id string) error {
	return r.client.Experiments.DeleteExperiment(ctx, ml.DeleteExperiment{
		ExperimentId: id,
	})
}

func (*ResourceExperiment) FieldTriggersLocal() map[string]deployplan.ActionType {
	// TF implementation: https://github.com/databricks/terraform-provider-databricks/blob/6c106e8e7052bb2726148d66309fd460ed444236/mlflow/resource_mlflow_experiment.go#L22
	return map[string]deployplan.ActionType{
		"name":              deployplan.ActionTypeUpdate,
		"artifact_location": deployplan.ActionTypeRecreate,

		// Tags updates are not supported by TF. This mirrors that behaviour.
		"tags": deployplan.ActionTypeSkip,
	}
}

func (*ResourceExperiment) FieldTriggersRemote() map[string]deployplan.ActionType {
	// TF implementation: https://github.com/databricks/terraform-provider-databricks/blob/6c106e8e7052bb2726148d66309fd460ed444236/mlflow/resource_mlflow_experiment.go#L22
	return map[string]deployplan.ActionType{
		"name":              deployplan.ActionTypeUpdate,
		"artifact_location": deployplan.ActionTypeRecreate,

		// Tags updates are not supported by TF. This mirrors that behaviour.
		"tags": deployplan.ActionTypeSkip,
	}
}

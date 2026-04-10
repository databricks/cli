package dresources

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
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

func (r *ResourceExperiment) DoUpdate(ctx context.Context, id string, config *ml.CreateExperiment, _ *PlanEntry) (*ml.Experiment, error) {
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

// OverrideChangeDesc suppresses drift for the experiment name field when the
// only difference is the /Workspace prefix. Stripping the prefix is necessary
// to avoid a persistent diff because the backend strips the /Workspace prefix,
// so remote returns "/Users/..." while the config has "/Workspace/Users/...".
//
// This matches the Terraform provider's experimentNameSuppressDiff behavior.
// https://github.com/databricks/terraform-provider-databricks/blob/8945a7b2328659b1fc976d04e32457305860131f/mlflow/resource_mlflow_experiment.go#L13
func (*ResourceExperiment) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, _ *ml.Experiment) error {
	if change.Action == deployplan.Skip {
		return nil
	}

	if path.String() != "name" {
		return nil
	}

	newStr, newOk := change.New.(string)
	remoteStr, remoteOk := change.Remote.(string)
	if !newOk || !remoteOk {
		return nil
	}

	// Normalize by stripping the /Workspace/ prefix (keeping the trailing slash
	// to avoid false matches like "/WorkspaceExtra/...").
	normalizedNew := stripWorkspacePrefix(newStr)
	normalizedRemote := stripWorkspacePrefix(remoteStr)
	if normalizedNew == normalizedRemote {
		change.Action = deployplan.Skip
		change.Reason = deployplan.ReasonAlias
	}

	return nil
}

// stripWorkspacePrefix removes the "/Workspace" portion from paths like
// "/Workspace/Users/..." while preserving the leading slash. Uses "/Workspace/"
// with trailing slash to avoid false matches on paths like "/WorkspaceExtra/...".
func stripWorkspacePrefix(s string) string {
	if strings.HasPrefix(s, "/Workspace/") {
		return s[len("/Workspace"):]
	}
	return s
}

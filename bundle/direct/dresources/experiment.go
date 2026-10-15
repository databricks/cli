package dresources

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/sdkclients"
	experiments "github.com/databricks/cli/libs/sdktypes/experiments/v1"
	"github.com/databricks/cli/libs/sdktypes/sdkconv"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	sdkexperiments "github.com/databricks/sdk-go/experiments/v1"
)

type ResourceExperiment struct {
	client *sdkclients.Clients
}

func (*ResourceExperiment) New(client *sdkclients.Clients) *ResourceExperiment {
	return &ResourceExperiment{
		client: client,
	}
}

func (*ResourceExperiment) PrepareState(input *resources.MlflowExperiment) *experiments.CreateExperimentRequest {
	return &input.CreateExperimentRequest
}

func (*ResourceExperiment) RemapState(experiment *experiments.Experiment) *experiments.CreateExperimentRequest {
	return &experiments.CreateExperimentRequest{
		Name:             experiment.Name,
		ArtifactLocation: experiment.ArtifactLocation,
		Tags:             experiment.Tags,
		TraceLocation:    experiment.TraceLocation,
	}
}

func (r *ResourceExperiment) DoRead(ctx context.Context, id string) (*experiments.Experiment, error) {
	result, err := r.client.Experiments.GetExperiment(ctx, sdkexperiments.GetExperimentRequest{
		ExperimentId: new(id),
	})
	if err != nil {
		return nil, err
	}
	return sdkconv.FromSDK[experiments.Experiment](result.Experiment)
}

func (r *ResourceExperiment) DoCreate(ctx context.Context, config *experiments.CreateExperimentRequest) (string, *experiments.Experiment, error) {
	request, err := toSDK[sdkexperiments.CreateExperimentRequest](config)
	if err != nil {
		return "", nil, err
	}
	result, err := r.client.Experiments.CreateExperiment(ctx, request)
	if err != nil {
		return "", nil, err
	}
	return utils.Deref(result.ExperimentId), nil, nil
}

func (r *ResourceExperiment) DoUpdate(ctx context.Context, id string, config *experiments.CreateExperimentRequest, _ *PlanEntry) (*experiments.Experiment, error) {
	_, err := r.client.Experiments.UpdateExperiment(ctx, sdkexperiments.UpdateExperimentRequest{
		ExperimentId: new(id),
		NewName:      config.Name,
	})
	return nil, err
}

func (r *ResourceExperiment) DoDelete(ctx context.Context, id string, _ *experiments.CreateExperimentRequest) error {
	_, err := r.client.Experiments.DeleteExperiment(ctx, sdkexperiments.DeleteExperimentRequest{
		ExperimentId: new(id),
	})
	return err
}

// OverrideChangeDesc suppresses drift for the experiment name field when the
// only difference is the /Workspace prefix. Stripping the prefix is necessary
// to avoid a persistent diff because the backend strips the /Workspace prefix,
// so remote returns "/Users/..." while the config has "/Workspace/Users/...".
//
// This matches the Terraform provider's experimentNameSuppressDiff behavior.
// https://github.com/databricks/terraform-provider-databricks/blob/8945a7b2328659b1fc976d04e32457305860131f/mlflow/resource_mlflow_experiment.go#L13
func (*ResourceExperiment) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, _ *experiments.Experiment) error {
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

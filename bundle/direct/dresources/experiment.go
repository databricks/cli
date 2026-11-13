package dresources

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
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
		TraceLocation:    input.TraceLocation,
		ForceSendFields:  utils.FilterFields[ml.CreateExperiment](input.ForceSendFields),
	}
}

func (*ResourceExperiment) RemapState(experiment *ml.Experiment) *ml.CreateExperiment {
	return &ml.CreateExperiment{
		Name:             experiment.Name,
		ArtifactLocation: experiment.ArtifactLocation,
		Tags:             experiment.Tags,
		TraceLocation:    experiment.TraceLocation,
		ForceSendFields:  utils.FilterFields[ml.CreateExperiment](experiment.ForceSendFields),
	}
}

func (*ResourceExperiment) KeyedSlices() map[string]any {
	return map[string]any{
		"tags": func(tag ml.ExperimentTag) (string, string) { return "key", tag.Key },
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

func (r *ResourceExperiment) DoUpdate(ctx context.Context, id string, config *ml.CreateExperiment, entry *PlanEntry) (*ml.Experiment, error) {
	tagChanges := entry.Changes.HasChange(structpath.MustParsePath("tags"))
	var remoteTags map[string]string
	if tagChanges {
		remoteTags = make(map[string]string)
		for _, tag := range entry.RemoteState.(*ml.Experiment).Tags {
			remoteTags[tag.Key] = tag.Value
		}
		removedKey, err := removedExperimentTag(entry.Changes, config.Tags, remoteTags)
		if err != nil {
			return nil, err
		}
		if removedKey != "" {
			return nil, fmt.Errorf("cannot remove experiment tag %q: the MLflow API does not support deleting experiment tags", removedKey)
		}
	}

	if entry.Changes.HasChange(structpath.MustParsePath("name")) {
		updateReq := ml.UpdateExperiment{
			ExperimentId:    id,
			NewName:         config.Name,
			ForceSendFields: utils.FilterFields[ml.UpdateExperiment](config.ForceSendFields),
		}
		if err := r.client.Experiments.UpdateExperiment(ctx, updateReq); err != nil {
			return nil, err
		}
	}

	if tagChanges {
		for _, tag := range config.Tags {
			if remoteValue, exists := remoteTags[tag.Key]; exists && remoteValue == tag.Value {
				continue
			}
			if err := r.client.Experiments.SetExperimentTag(ctx, ml.SetExperimentTag{
				ExperimentId: id,
				Key:          tag.Key,
				Value:        tag.Value,
			}); err != nil {
				return nil, fmt.Errorf("failed to set experiment %s tag %q: %w", id, tag.Key, err)
			}
		}
	}

	return nil, nil
}

func removedExperimentTag(changes Changes, desired []ml.ExperimentTag, remote map[string]string) (string, error) {
	desiredKeys := make(map[string]bool, len(desired))
	for _, tag := range desired {
		desiredKeys[tag.Key] = true
	}

	var removedKeys []string
	for field, change := range changes {
		if change.Action == deployplan.Skip || change.Old == nil {
			continue
		}
		path, err := structpath.ParsePath(field)
		if err != nil || !path.HasPrefix(structpath.MustParsePath("tags")) {
			continue
		}
		if field == "tags" {
			// A loaded plan stores change values as untyped JSON.
			data, err := json.Marshal(change.Old)
			if err != nil {
				return "", fmt.Errorf("encoding previously configured experiment tags: %w", err)
			}
			var oldTags []ml.ExperimentTag
			if err := json.Unmarshal(data, &oldTags); err != nil {
				return "", fmt.Errorf("decoding previously configured experiment tags: %w", err)
			}
			for _, tag := range oldTags {
				removedKeys = append(removedKeys, tag.Key)
			}
		} else if change.New == nil {
			keyField, key, ok := path.KeyValue()
			if ok && keyField == "key" && path.Parent().String() == "tags" {
				removedKeys = append(removedKeys, key)
			}
		}
	}

	slices.Sort(removedKeys)
	for _, key := range removedKeys {
		if !desiredKeys[key] {
			if _, exists := remote[key]; exists {
				return key, nil
			}
		}
	}
	return "", nil
}

func (r *ResourceExperiment) DoDelete(ctx context.Context, id string, _ *ml.CreateExperiment) error {
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

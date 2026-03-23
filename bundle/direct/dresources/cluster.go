package dresources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

type ResourceCluster struct {
	client *databricks.WorkspaceClient
}

func (r *ResourceCluster) New(client *databricks.WorkspaceClient) any {
	return &ResourceCluster{
		client: client,
	}
}

func (r *ResourceCluster) PrepareState(input *resources.Cluster) *compute.ClusterSpec {
	return &input.ClusterSpec
}

// clusterRemapCopy maps ClusterDetails (remote GET response) to ClusterSpec (local state).
var clusterRemapCopy = newCopy[compute.ClusterDetails, compute.ClusterSpec]()

func (r *ResourceCluster) RemapState(input *compute.ClusterDetails) *compute.ClusterSpec {
	spec := clusterRemapCopy.Do(input)
	if input.Spec != nil {
		spec.ApplyPolicyDefaultValues = input.Spec.ApplyPolicyDefaultValues
	}
	return spec
}

func (r *ResourceCluster) DoRead(ctx context.Context, id string) (*compute.ClusterDetails, error) {
	return r.client.Clusters.GetByClusterId(ctx, id)
}

// clusterCreateCopy maps ClusterSpec (local state) to CreateCluster (API request).
var clusterCreateCopy = newCopy[compute.ClusterSpec, compute.CreateCluster]()

func (r *ResourceCluster) DoCreate(ctx context.Context, config *compute.ClusterSpec) (string, *compute.ClusterDetails, error) {
	create := clusterCreateCopy.Do(config)
	forceNumWorkers(config, &create.ForceSendFields)
	wait, err := r.client.Clusters.Create(ctx, *create)
	if err != nil {
		return "", nil, err
	}
	return wait.ClusterId, nil, nil
}

// clusterEditCopy maps ClusterSpec (local state) to EditCluster (API request).
var clusterEditCopy = newCopy[compute.ClusterSpec, compute.EditCluster]()

func (r *ResourceCluster) DoUpdate(ctx context.Context, id string, config *compute.ClusterSpec, _ Changes) (*compute.ClusterDetails, error) {
	edit := clusterEditCopy.Do(config)
	edit.ClusterId = id
	forceNumWorkers(config, &edit.ForceSendFields)

	// Same retry as in TF provider logic
	// https://github.com/databricks/terraform-provider-databricks/blob/3eecd0f90cf99d7777e79a3d03c41f9b2aafb004/clusters/resource_cluster.go#L624
	timeout := 15 * time.Minute
	_, err := retries.Poll(ctx, timeout, func() (*compute.WaitGetClusterRunning[struct{}], *retries.Err) {
		wait, err := r.client.Clusters.Edit(ctx, *edit)
		if err == nil {
			return wait, nil
		}

		var apiErr *apierr.APIError
		// Only Running and Terminated clusters can be modified. In particular, autoscaling clusters cannot be modified
		// while the resizing is ongoing. We retry in this case. Scaling can take several minutes.
		if errors.As(err, &apiErr) && apiErr.ErrorCode == "INVALID_STATE" {
			return nil, retries.Continues(fmt.Sprintf("cluster %s cannot be modified in its current state: %s", id, apiErr.Message))
		}
		return nil, retries.Halt(err)
	})
	return nil, err
}

func (r *ResourceCluster) DoResize(ctx context.Context, id string, config *compute.ClusterSpec) error {
	_, err := r.client.Clusters.Resize(ctx, compute.ResizeCluster{
		ClusterId:       id,
		NumWorkers:      config.NumWorkers,
		Autoscale:       config.Autoscale,
		ForceSendFields: utils.FilterFields[compute.ResizeCluster](config.ForceSendFields),
	})
	return err
}

func (r *ResourceCluster) DoDelete(ctx context.Context, id string) error {
	return r.client.Clusters.PermanentDeleteByClusterId(ctx, id)
}

func (r *ResourceCluster) OverrideChangeDesc(ctx context.Context, p *structpath.PathNode, change *ChangeDesc, remoteState *compute.ClusterDetails) error {
	// We're only interested in downgrading some updates to skips. Changes that already skipped or cause recreation should remain unchanged.
	if change.Action != deployplan.Update {
		return nil
	}

	path := p.Prefix(1).String()
	switch path {
	case "data_security_mode":
		// We do change skip here in the same way TF provider does suppress diff if the alias is used.
		// https://github.com/databricks/terraform-provider-databricks/blob/main/clusters/resource_cluster.go#L109-L117
		if change.New == compute.DataSecurityModeDataSecurityModeStandard && change.Remote == compute.DataSecurityModeUserIsolation && change.New == change.Old {
			change.Action = deployplan.Skip
			change.Reason = deployplan.ReasonAlias
		} else if change.New == compute.DataSecurityModeDataSecurityModeDedicated && change.Remote == compute.DataSecurityModeSingleUser && change.New == change.Old {
			change.Action = deployplan.Skip
			change.Reason = deployplan.ReasonAlias
		} else if change.New == compute.DataSecurityModeDataSecurityModeAuto && (change.Remote == compute.DataSecurityModeSingleUser || change.Remote == compute.DataSecurityModeUserIsolation) && change.New == change.Old {
			change.Action = deployplan.Skip
			change.Reason = deployplan.ReasonAlias
		}

	case "num_workers", "autoscale":
		if remoteState.State == compute.StateRunning {
			change.Action = deployplan.Resize
		}
	}
	return nil
}

// forceNumWorkers ensures NumWorkers is sent when Autoscale is not set,
// because the API requires one of them.
func forceNumWorkers(config *compute.ClusterSpec, fsf *[]string) {
	if config.Autoscale == nil && config.NumWorkers == 0 {
		*fsf = append(*fsf, "NumWorkers")
	}
}


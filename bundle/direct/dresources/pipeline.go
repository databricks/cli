package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/sdkclients"
	pipelines "github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	"github.com/databricks/cli/libs/sdktypes/sdkconv"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	sdkpipelines "github.com/databricks/sdk-go/pipelines/v2"
)

// PipelineState is the state type for Pipeline resources. It extends CreatePipeline with
// the CascadeOnDestroy field, a delete-time setting that is not part of the pipeline spec
type PipelineState struct {
	pipelines.CreatePipelineRequest

	// CascadeOnDestroy controls whether deleting the pipeline also deletes its datasets (MVs,
	// STs, Views). Nil means the server default (cascade) applies. Read from persisted state at
	// delete time; never sent on create/update.
	CascadeOnDestroy *bool `json:"cascade_on_destroy,omitempty"`
}

// PipelineRemote is the return type for DoRead. It embeds CreatePipeline so that all
// paths in StateType are valid paths in RemoteType.
// Note that cascade_on_destroy is intentionally absent: it is a delete-time-only setting
// that the GET response never returns, so the engine suppresses its drift automatically
type PipelineRemote struct {
	pipelines.CreatePipelineRequest

	// Remote-specific fields from pipelines.GetPipelineResponse
	Cause                        *string                               `json:"cause,omitempty"`
	ClusterId                    *string                               `json:"cluster_id,omitempty"`
	CreatorUserName              *string                               `json:"creator_user_name,omitempty"`
	EffectiveBudgetPolicyId      *string                               `json:"effective_budget_policy_id,omitempty"`
	EffectivePublishingMode      pipelines.PublishingMode              `json:"effective_publishing_mode,omitempty"`
	EffectiveServerlessComputeId *string                               `json:"effective_serverless_compute_id,omitempty"`
	Health                       pipelines.PipelineHealthStatus        `json:"health,omitempty"`
	LastModified                 *int64                                `json:"last_modified,omitempty"`
	LatestUpdates                []pipelines.UpdateStateInfo           `json:"latest_updates,omitempty"`
	PipelineId                   *string                               `json:"pipeline_id,omitempty"`
	RunAsUserName                *string                               `json:"run_as_user_name,omitempty"`
	State                        pipelines.PipelineState_PipelineState `json:"state,omitempty"`
}

type ResourcePipeline struct {
	client *sdkclients.Clients
}

func (*ResourcePipeline) New(client *sdkclients.Clients) *ResourcePipeline {
	return &ResourcePipeline{
		client: client,
	}
}

func (*ResourcePipeline) PrepareState(input *resources.Pipeline) *PipelineState {
	return &PipelineState{
		CreatePipelineRequest: input.CreatePipelineRequest,
		CascadeOnDestroy:      input.CascadeOnDestroy,
	}
}

func (*ResourcePipeline) RemapState(remote *PipelineRemote) *PipelineState {
	return &PipelineState{
		CreatePipelineRequest: remote.CreatePipelineRequest,
		// cascade_on_destroy is input-only and absent from PipelineRemote, so it stays nil here.
		CascadeOnDestroy: nil,
	}
}

func (r *ResourcePipeline) DoRead(ctx context.Context, id string) (*PipelineRemote, error) {
	resp, err := r.client.Pipelines.Get(ctx, sdkpipelines.GetPipelineRequest{PipelineId: new(id)})
	if err != nil {
		return nil, err
	}
	return makePipelineRemote(resp)
}

func makePipelineRemote(resp *sdkpipelines.GetPipelineResponse) (*PipelineRemote, error) {
	p, err := sdkconv.FromSDK[pipelines.GetPipelineResponse](resp)
	if err != nil {
		return nil, err
	}
	var createPipeline pipelines.CreatePipelineRequest
	if p.Spec != nil {
		// PipelineSpec is CreatePipelineRequest without the request-only fields
		// (allow_duplicate_names, dry_run) and the ones returned at the top level.
		if err := sdkconv.Convert(&createPipeline, p.Spec); err != nil {
			return nil, err
		}
		createPipeline.Parameters = p.Parameters
		createPipeline.RunAs = p.RunAs
	}
	return &PipelineRemote{
		CreatePipelineRequest:        createPipeline,
		Cause:                        p.Cause,
		ClusterId:                    p.ClusterId,
		CreatorUserName:              p.CreatorUserName,
		EffectiveBudgetPolicyId:      p.EffectiveBudgetPolicyId,
		EffectivePublishingMode:      p.EffectivePublishingMode,
		EffectiveServerlessComputeId: p.EffectiveServerlessComputeId,
		Health:                       p.Health,
		LastModified:                 p.LastModified,
		LatestUpdates:                p.LatestUpdates,
		PipelineId:                   p.PipelineId,
		RunAsUserName:                p.RunAsUserName,
		State:                        p.State,
	}, nil
}

func (r *ResourcePipeline) DoCreate(ctx context.Context, config *PipelineState) (string, *PipelineRemote, error) {
	request, err := sdkconv.ToSDK[sdkpipelines.CreatePipelineRequest](&config.CreatePipelineRequest)
	if err != nil {
		return "", nil, err
	}
	response, err := r.client.Pipelines.Create(ctx, *request)
	if err != nil {
		return "", nil, err
	}
	return utils.Deref(response.PipelineId), nil, nil
}

func (r *ResourcePipeline) DoUpdate(ctx context.Context, id string, config *PipelineState, entry *PlanEntry) (*PipelineRemote, error) {
	// cascade_on_destroy is a delete-time-only setting with no update API, so a change to it
	// alone must persist to state without a pipeline Update call.
	if !entry.Changes.HasChangeExcept("cascade_on_destroy") {
		return nil, nil
	}

	// EditPipelineRequest is CreatePipelineRequest without dry_run plus the pipeline id.
	create := config.CreatePipelineRequest
	create.DryRun = nil
	request, err := sdkconv.ToSDK[sdkpipelines.EditPipelineRequest](&create)
	if err != nil {
		return nil, err
	}
	request.PipelineId = new(id)
	_, err = r.client.Pipelines.Edit(ctx, *request)
	return nil, err
}

func (r *ResourcePipeline) DoDelete(ctx context.Context, id string, state *PipelineState) error {
	// A nil cascade_on_destroy preserves the backend default (cascade); an explicit
	// false is sent as cascade=false.
	_, err := r.client.Pipelines.Delete(ctx, sdkpipelines.DeletePipelineRequest{
		PipelineId: new(id),
		Cascade:    state.CascadeOnDestroy,
		Force:      nil, // Not set by DABs
	})
	return err
}

// OverrideChangeDesc forces a state-only Update when cascade_on_destroy changes locally.
// The field is absent from PipelineRemote, so unsetting puts us in the scenario where
// old=false, new=nil, remote=nil.
// The problem is that classifier skips the change if remote and new are "empty". There is a bug
// where "false" value for a boolean field is treated as empty. Hence, we set this override.
func (*ResourcePipeline) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, _ *PipelineRemote) error {
	if path.String() == "cascade_on_destroy" && !structdiff.IsEqual(change.Old, change.New) {
		change.Action = deployplan.Update
	}
	return nil
}

// Note, terraform provider either
// a) reads back state at least once and fails create if state is "failed"
// b) repeatededly reads state until state is "running" (if spec.Contionous is set).
// TODO: investigate if we need to mimic this behaviour or can rely on Create status code.

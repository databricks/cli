package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

// PipelineState is the state type for Pipeline resources. It extends CreatePipeline with
// the CascadeOnDestroy field, a delete-time setting that is not part of the pipeline spec
type PipelineState struct {
	pipelines.CreatePipeline

	// CascadeOnDestroy controls whether deleting the pipeline also deletes its datasets (MVs,
	// STs, Views). Nil means the server default (cascade) applies. Read from persisted state at
	// delete time; never sent on create/update.
	CascadeOnDestroy *bool `json:"cascade_on_destroy,omitempty"`
}

func (s *PipelineState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s PipelineState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// PipelineRemote is the return type for DoRead. It embeds CreatePipeline so that all
// paths in StateType are valid paths in RemoteType.
// Note that cascade_on_destroy is intentionally absent: it is a delete-time-only setting
// that the GET response never returns, so the engine suppresses its drift automatically
type PipelineRemote struct {
	pipelines.CreatePipeline

	// Remote-specific fields from pipelines.GetPipelineResponse
	Cause                        string                              `json:"cause,omitempty"`
	ClusterId                    string                              `json:"cluster_id,omitempty"`
	CreatorUserName              string                              `json:"creator_user_name,omitempty"`
	EffectiveBudgetPolicyId      string                              `json:"effective_budget_policy_id,omitempty"`
	EffectivePublishingMode      pipelines.PublishingMode            `json:"effective_publishing_mode,omitempty"`
	EffectiveServerlessComputeId string                              `json:"effective_serverless_compute_id,omitempty"`
	Health                       pipelines.GetPipelineResponseHealth `json:"health,omitempty"`
	LastModified                 int64                               `json:"last_modified,omitempty"`
	LatestUpdates                []pipelines.UpdateStateInfo         `json:"latest_updates,omitempty"`
	PipelineId                   string                              `json:"pipeline_id,omitempty"`
	RunAsUserName                string                              `json:"run_as_user_name,omitempty"`
	State                        pipelines.PipelineState             `json:"state,omitempty"`
}

// Custom marshaler needed because embedded CreatePipeline has its own MarshalJSON
// which would otherwise take over and ignore the additional fields.
func (s *PipelineRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s PipelineRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourcePipeline struct {
	client *databricks.WorkspaceClient
}

func (*ResourcePipeline) New(client *databricks.WorkspaceClient) *ResourcePipeline {
	return &ResourcePipeline{
		client: client,
	}
}

func (*ResourcePipeline) PrepareState(input *resources.Pipeline) *PipelineState {
	return &PipelineState{
		CreatePipeline:   input.CreatePipeline,
		CascadeOnDestroy: input.CascadeOnDestroy,
	}
}

func (*ResourcePipeline) RemapState(remote *PipelineRemote) *PipelineState {
	return &PipelineState{
		CreatePipeline: remote.CreatePipeline,
		// cascade_on_destroy is input-only and absent from PipelineRemote, so it stays nil here.
		CascadeOnDestroy: nil,
	}
}

func (r *ResourcePipeline) DoRead(ctx context.Context, id string) (*PipelineRemote, error) {
	resp, err := r.client.Pipelines.GetByPipelineId(ctx, id)
	if err != nil {
		return nil, err
	}
	return makePipelineRemote(resp), nil
}

// AllowDuplicateNames and DryRun are request-only, so PipelineSpec has no value for them.
// Continuous and Development are deprecated in the SDK but remain supported bundle config
// fields, so they are copied here and into the edit request like any other field.
var toCreatePipeline = newCopier[pipelines.PipelineSpec, pipelines.CreatePipeline]()

func makePipelineRemote(p *pipelines.GetPipelineResponse) *PipelineRemote {
	var createPipeline pipelines.CreatePipeline
	if p.Spec != nil {
		createPipeline = *toCreatePipeline.Copy(p.Spec)
		// The response reports these next to the spec, not in it.
		createPipeline.Parameters = p.Parameters
		createPipeline.RunAs = p.RunAs
	}
	return &PipelineRemote{
		CreatePipeline:               createPipeline,
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
	}
}

func (r *ResourcePipeline) DoCreate(ctx context.Context, config *PipelineState) (string, *PipelineRemote, error) {
	response, err := r.client.Pipelines.Create(ctx, config.CreatePipeline)
	if err != nil {
		return "", nil, err
	}
	return response.PipelineId, nil, nil
}

var toEditPipeline = newCopier[PipelineState, pipelines.EditPipeline]()

func (r *ResourcePipeline) DoUpdate(ctx context.Context, id string, config *PipelineState, entry *PlanEntry) (*PipelineRemote, error) {
	// cascade_on_destroy is a delete-time-only setting with no update API, so a change to it
	// alone must persist to state without a pipeline Update call.
	if !entry.Changes.HasChangeExcept("cascade_on_destroy") {
		return nil, nil
	}

	request := toEditPipeline.Copy(config)
	request.PipelineId = id

	return nil, r.client.Pipelines.Update(ctx, *request)
}

func (r *ResourcePipeline) DoDelete(ctx context.Context, id string, state *PipelineState) error {
	if state.CascadeOnDestroy == nil {
		// No explicit cascade_on_destroy in config: preserve the backend default (cascade).
		return r.client.Pipelines.DeleteByPipelineId(ctx, id)
	}
	return r.client.Pipelines.Delete(ctx, pipelines.DeletePipelineRequest{
		PipelineId: id,
		Cascade:    *state.CascadeOnDestroy,
		Force:      false,
		// Cascade is marshaled as `url:"cascade,omitempty"`, so a false value would be dropped from
		// the query string. We specify `cascade` in ForceSendFields so the SDK send cascade=false explicitly
		ForceSendFields: []string{"Cascade"},
	})
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

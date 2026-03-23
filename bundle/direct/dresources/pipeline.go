package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/fieldcopy"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

// PipelineRemote is the return type for DoRead. It embeds CreatePipeline so that all
// paths in StateType are valid paths in RemoteType.
type PipelineRemote struct {
	pipelines.CreatePipeline

	// Remote-specific fields from pipelines.GetPipelineResponse
	Cause                   string                              `json:"cause,omitempty"`
	ClusterId               string                              `json:"cluster_id,omitempty"`
	CreatorUserName         string                              `json:"creator_user_name,omitempty"`
	EffectiveBudgetPolicyId string                              `json:"effective_budget_policy_id,omitempty"`
	EffectivePublishingMode pipelines.PublishingMode            `json:"effective_publishing_mode,omitempty"`
	Health                  pipelines.GetPipelineResponseHealth `json:"health,omitempty"`
	LastModified            int64                               `json:"last_modified,omitempty"`
	LatestUpdates           []pipelines.UpdateStateInfo         `json:"latest_updates,omitempty"`
	PipelineId              string                              `json:"pipeline_id,omitempty"`
	RunAsUserName           string                              `json:"run_as_user_name,omitempty"`
	State                   pipelines.PipelineState             `json:"state,omitempty"`
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

func (*ResourcePipeline) PrepareState(input *resources.Pipeline) *pipelines.CreatePipeline {
	return &input.CreatePipeline
}

func (*ResourcePipeline) RemapState(remote *PipelineRemote) *pipelines.CreatePipeline {
	return &remote.CreatePipeline
}

func (r *ResourcePipeline) DoRead(ctx context.Context, id string) (*PipelineRemote, error) {
	resp, err := r.client.Pipelines.GetByPipelineId(ctx, id)
	if err != nil {
		return nil, err
	}
	return makePipelineRemote(resp), nil
}

// pipelineSpecCopy maps PipelineSpec (from GET response) to CreatePipeline (local state).
var pipelineSpecCopy = fieldcopy.Copy[pipelines.PipelineSpec, pipelines.CreatePipeline]{
	SkipDst: []string{
		"AllowDuplicateNames", // Request-only field, not in PipelineSpec.
		"DryRun",              // Request-only field, not in PipelineSpec.
		"RunAs",               // Pulled from GetPipelineResponse in post-processing.
	},
}

// pipelineRemoteCopy maps GetPipelineResponse to PipelineRemote extra fields.
var pipelineRemoteCopy = fieldcopy.Copy[pipelines.GetPipelineResponse, PipelineRemote]{
	SkipSrc: []string{
		"Name",
		"RunAs",
		"Spec",
	},
	SkipDst: []string{
		"CreatePipeline", // Populated separately from Spec.
	},
}

func makePipelineRemote(p *pipelines.GetPipelineResponse) *PipelineRemote {
	remote := pipelineRemoteCopy.Do(p)
	if p.Spec != nil {
		remote.CreatePipeline = pipelineSpecCopy.Do(p.Spec)
		remote.CreatePipeline.RunAs = p.RunAs
	}
	return &remote
}

func (r *ResourcePipeline) DoCreate(ctx context.Context, config *pipelines.CreatePipeline) (string, *PipelineRemote, error) {
	response, err := r.client.Pipelines.Create(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	return response.PipelineId, nil, nil
}

// pipelineEditCopy maps CreatePipeline (local state) to EditPipeline (API request).
var pipelineEditCopy = fieldcopy.Copy[pipelines.CreatePipeline, pipelines.EditPipeline]{
	SkipSrc: []string{
		"DryRun", // Request-only field, not in EditPipeline.
	},
	SkipDst: []string{
		"ExpectedLastModified", // Left at zero.
		"PipelineId",           // Set from function parameter.
	},
}

func (r *ResourcePipeline) DoUpdate(ctx context.Context, id string, config *pipelines.CreatePipeline, _ Changes) (*PipelineRemote, error) {
	request := pipelineEditCopy.Do(config)
	request.PipelineId = id
	return nil, r.client.Pipelines.Update(ctx, request)
}

func (r *ResourcePipeline) DoDelete(ctx context.Context, id string) error {
	return r.client.Pipelines.DeleteByPipelineId(ctx, id)
}

// Note, terraform provider either
// a) reads back state at least once and fails create if state is "failed"
// b) repeatededly reads state until state is "running" (if spec.Contionous is set).
// TODO: investigate if we need to mimic this behaviour or can rely on Create status code.

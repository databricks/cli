package dresources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
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

	// Testing is PipelineSpec.testing, which the Go SDK does not have yet.
	Testing *resources.PipelineTesting `json:"testing,omitempty"`
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

	// Testing is spec.testing of the Get response, read by DoRead because the Go SDK does not have it yet.
	Testing *resources.PipelineTesting `json:"testing,omitempty"`
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
		Testing:          input.Testing,
	}
}

func (*ResourcePipeline) RemapState(remote *PipelineRemote) *PipelineState {
	return &PipelineState{
		CreatePipeline: remote.CreatePipeline,
		// cascade_on_destroy is input-only and absent from PipelineRemote, so it stays nil here.
		CascadeOnDestroy: nil,
		Testing:          remote.Testing,
	}
}

// pipelineTestingResponse holds spec.testing of a Get Pipeline response.
type pipelineTestingResponse struct {
	Spec struct {
		Testing *resources.PipelineTesting `json:"testing"`
	} `json:"spec"`
}

// DoRead sends the same request as GetByPipelineId, but raw, so that it can also read spec.testing,
// which GetPipelineResponse does not have yet. Switch back to GetByPipelineId once it does.
func (r *ResourcePipeline) DoRead(ctx context.Context, id string) (*PipelineRemote, error) {
	var raw json.RawMessage
	if err := r.doRaw(ctx, http.MethodGet, "/api/2.0/pipelines/"+id, nil, &raw); err != nil {
		return nil, err
	}
	var resp pipelines.GetPipelineResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	var testing pipelineTestingResponse
	if err := json.Unmarshal(raw, &testing); err != nil {
		return nil, err
	}
	remote := makePipelineRemote(&resp)
	remote.Testing = testing.Spec.Testing
	return remote, nil
}

func makePipelineRemote(p *pipelines.GetPipelineResponse) *PipelineRemote {
	var createPipeline pipelines.CreatePipeline
	if p.Spec != nil {
		spec := p.Spec
		createPipeline = pipelines.CreatePipeline{
			// Note: AllowDuplicateNames and DryRun are not in PipelineSpec,
			// they are request-only fields, so they stay at their zero values.
			AllowDuplicateNames: false,
			BudgetPolicyId:      spec.BudgetPolicyId,
			Catalog:             spec.Catalog,
			Channel:             spec.Channel,
			Clusters:            spec.Clusters,
			Configuration:       spec.Configuration,
			Continuous:          spec.Continuous, //nolint:staticcheck // SA1019: pipeline continuous is deprecated in the SDK but remains a supported bundle config field
			Deployment:          spec.Deployment,
			Development:         spec.Development, //nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
			DryRun:              false,
			Edition:             spec.Edition,
			Environment:         spec.Environment,
			EventLog:            spec.EventLog,
			Filters:             spec.Filters,
			GatewayDefinition:   spec.GatewayDefinition,
			Id:                  spec.Id,
			IngestionDefinition: spec.IngestionDefinition,
			Libraries:           spec.Libraries,
			Name:                spec.Name,
			Notifications:       spec.Notifications,
			Parameters:          p.Parameters,
			ParentPath:          spec.ParentPath,
			Photon:              spec.Photon,
			RestartWindow:       spec.RestartWindow,
			RootPath:            spec.RootPath,
			RunAs:               p.RunAs,
			Schema:              spec.Schema,
			Serverless:          spec.Serverless,
			ServerlessComputeId: spec.ServerlessComputeId,
			Storage:             spec.Storage,
			Tags:                spec.Tags,
			Target:              spec.Target,
			Trigger:             spec.Trigger,
			UsagePolicyId:       spec.UsagePolicyId,
			ForceSendFields:     utils.FilterFields[pipelines.CreatePipeline](spec.ForceSendFields, "AllowDuplicateNames", "DryRun", "Parameters", "RunAs"),
		}
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
		// GetPipelineResponse has no spec.testing; DoRead sets it from the raw response.
		Testing: nil,
	}
}

func (r *ResourcePipeline) DoCreate(ctx context.Context, config *PipelineState) (string, *PipelineRemote, error) {
	if config.Testing != nil {
		var response pipelines.CreatePipelineResponse
		err := r.doWithTesting(ctx, http.MethodPost, "/api/2.0/pipelines", config.CreatePipeline, config.Testing, &response)
		if err != nil {
			return "", nil, err
		}
		return response.PipelineId, nil, nil
	}
	response, err := r.client.Pipelines.Create(ctx, config.CreatePipeline)
	if err != nil {
		return "", nil, err
	}
	return response.PipelineId, nil, nil
}

func (r *ResourcePipeline) DoUpdate(ctx context.Context, id string, config *PipelineState, entry *PlanEntry) (*PipelineRemote, error) {
	// cascade_on_destroy is a delete-time-only setting with no update API, so a change to it
	// alone must persist to state without a pipeline Update call.
	if !entry.Changes.HasChangeExcept("cascade_on_destroy") {
		return nil, nil
	}

	request := pipelines.EditPipeline{
		AllowDuplicateNames:  config.AllowDuplicateNames,
		BudgetPolicyId:       config.BudgetPolicyId,
		Catalog:              config.Catalog,
		Channel:              config.Channel,
		Clusters:             config.Clusters,
		Configuration:        config.Configuration,
		Continuous:           config.Continuous, //nolint:staticcheck // SA1019: pipeline continuous is deprecated in the SDK but remains a supported bundle config field
		Deployment:           config.Deployment,
		Development:          config.Development, //nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
		Edition:              config.Edition,
		Environment:          config.Environment,
		EventLog:             config.EventLog,
		ExpectedLastModified: 0,
		Filters:              config.Filters,
		GatewayDefinition:    config.GatewayDefinition,
		Id:                   config.Id,
		IngestionDefinition:  config.IngestionDefinition,
		Libraries:            config.Libraries,
		Name:                 config.Name,
		Notifications:        config.Notifications,
		Parameters:           config.Parameters,
		ParentPath:           config.ParentPath,
		Photon:               config.Photon,
		RestartWindow:        config.RestartWindow,
		RootPath:             config.RootPath,
		RunAs:                config.RunAs,
		Schema:               config.Schema,
		Serverless:           config.Serverless,
		ServerlessComputeId:  config.ServerlessComputeId,
		Storage:              config.Storage,
		Tags:                 config.Tags,
		Target:               config.Target,
		Trigger:              config.Trigger,
		UsagePolicyId:        config.UsagePolicyId,
		PipelineId:           id,
		ForceSendFields:      utils.FilterFields[pipelines.EditPipeline](config.ForceSendFields),
	}

	if config.Testing != nil {
		return nil, r.doWithTesting(ctx, http.MethodPut, "/api/2.0/pipelines/"+id, request, config.Testing, nil)
	}
	return nil, r.client.Pipelines.Update(ctx, request)
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

// doWithTesting sends a Create or Edit request with the testing field, which the Go SDK does not
// have yet. The body is the SDK's JSON for the request plus testing. Remove this once
// CreatePipeline and EditPipeline have the field; pipelines without testing already use the SDK.
func (r *ResourcePipeline) doWithTesting(ctx context.Context, method, path string, request any, testing *resources.PipelineTesting, response any) error {
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	// RawMessage values keep the SDK's encoding, including large integers, unchanged.
	var body map[string]json.RawMessage
	if err := json.Unmarshal(data, &body); err != nil {
		return err
	}
	body["testing"], err = json.Marshal(testing)
	if err != nil {
		return err
	}
	return r.doRaw(ctx, method, path, body, response)
}

// doRaw sends a request that the Go SDK cannot express yet because it lacks the testing field.
func (r *ResourcePipeline) doRaw(ctx context.Context, method, path string, request, response any) error {
	apiClient, err := client.New(r.client.Config)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}
	return apiClient.Do(ctx, method, path, auth.WorkspaceIDHeaders(r.client.Config), nil, request, response)
}

// Note, terraform provider either
// a) reads back state at least once and fails create if state is "failed"
// b) repeatededly reads state until state is "running" (if spec.Contionous is set).
// TODO: investigate if we need to mimic this behaviour or can rely on Create status code.

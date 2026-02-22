package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/utils"
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
			Continuous:          spec.Continuous,
			Deployment:          spec.Deployment,
			Development:         spec.Development,
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
			Photon:              spec.Photon,
			RestartWindow:       spec.RestartWindow,
			RootPath:            spec.RootPath,
			RunAs:               p.RunAs,
			Schema:              spec.Schema,
			Serverless:          spec.Serverless,
			Storage:             spec.Storage,
			Tags:                spec.Tags,
			Target:              spec.Target,
			Trigger:             spec.Trigger,
			UsagePolicyId:       spec.UsagePolicyId,
			ForceSendFields:     utils.FilterFields[pipelines.CreatePipeline](spec.ForceSendFields, "AllowDuplicateNames", "DryRun", "RunAs"),
		}
	}
	return &PipelineRemote{
		CreatePipeline:          createPipeline,
		Cause:                   p.Cause,
		ClusterId:               p.ClusterId,
		CreatorUserName:         p.CreatorUserName,
		EffectiveBudgetPolicyId: p.EffectiveBudgetPolicyId,
		Health:                  p.Health,
		LastModified:            p.LastModified,
		LatestUpdates:           p.LatestUpdates,
		PipelineId:              p.PipelineId,
		RunAsUserName:           p.RunAsUserName,
		State:                   p.State,
	}
}

func (r *ResourcePipeline) DoCreate(ctx context.Context, config *pipelines.CreatePipeline) (string, *PipelineRemote, error) {
	response, err := r.client.Pipelines.Create(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	return response.PipelineId, nil, nil
}

func (r *ResourcePipeline) DoUpdate(ctx context.Context, id string, config *pipelines.CreatePipeline, _ Changes) (*PipelineRemote, error) {
	request := pipelines.EditPipeline{
		AllowDuplicateNames:  config.AllowDuplicateNames,
		BudgetPolicyId:       config.BudgetPolicyId,
		Catalog:              config.Catalog,
		Channel:              config.Channel,
		Clusters:             config.Clusters,
		Configuration:        config.Configuration,
		Continuous:           config.Continuous,
		Deployment:           config.Deployment,
		Development:          config.Development,
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
		Photon:               config.Photon,
		RestartWindow:        config.RestartWindow,
		RootPath:             config.RootPath,
		RunAs:                config.RunAs,
		Schema:               config.Schema,
		Serverless:           config.Serverless,
		Storage:              config.Storage,
		Tags:                 config.Tags,
		Target:               config.Target,
		Trigger:              config.Trigger,
		UsagePolicyId:        config.UsagePolicyId,
		PipelineId:           id,
		ForceSendFields:      utils.FilterFields[pipelines.EditPipeline](config.ForceSendFields),
	}

	return nil, r.client.Pipelines.Update(ctx, request)
}

func (r *ResourcePipeline) DoDelete(ctx context.Context, id string) error {
	return r.client.Pipelines.DeleteByPipelineId(ctx, id)
}

// Note, terraform provider either
// a) reads back state at least once and fails create if state is "failed"
// b) repeatededly reads state until state is "running" (if spec.Contionous is set).
// TODO: investigate if we need to mimic this behaviour or can rely on Create status code.

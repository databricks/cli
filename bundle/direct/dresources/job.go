package dresources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// JobRemote is the return type for DoRead. It embeds JobSettings so that all
// paths in StateType are valid paths in RemoteType.
type JobRemote struct {
	jobs.JobSettings

	// Remote-specific fields from jobs.Job
	CreatedTime             int64                   `json:"created_time,omitempty"`
	CreatorUserName         string                  `json:"creator_user_name,omitempty"`
	EffectiveBudgetPolicyId string                  `json:"effective_budget_policy_id,omitempty"`
	EffectiveUsagePolicyId  string                  `json:"effective_usage_policy_id,omitempty"`
	JobId                   int64                   `json:"job_id,omitempty"`
	RunAsUserName           string                  `json:"run_as_user_name,omitempty"`
	TriggerState            *jobs.TriggerStateProto `json:"trigger_state,omitempty"`
}

// Custom marshaler needed because embedded JobSettings has its own MarshalJSON
// which would otherwise take over and ignore the additional fields.
func (s *JobRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s JobRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourceJob struct {
	client *databricks.WorkspaceClient
}

func (*ResourceJob) New(client *databricks.WorkspaceClient) *ResourceJob {
	return &ResourceJob{
		client: client,
	}
}

func (*ResourceJob) PrepareState(input *resources.Job) *jobs.JobSettings {
	return &input.JobSettings
}

func (*ResourceJob) RemapState(remote *JobRemote) *jobs.JobSettings {
	return &remote.JobSettings
}

func getTaskKey(x jobs.Task) (string, string) {
	return "task_key", x.TaskKey
}

func (*ResourceJob) KeyedSlices() map[string]any {
	return map[string]any{
		"tasks": getTaskKey,
	}
}

func (r *ResourceJob) DoRead(ctx context.Context, id string) (*JobRemote, error) {
	idInt, err := parseJobID(id)
	if err != nil {
		return nil, err
	}
	// GetByJobId only fetches the first page (100 tasks). Jobs.Get handles
	// pagination and returns the complete job with all tasks merged.
	job, err := r.client.Jobs.Get(ctx, jobs.GetJobRequest{
		JobId:           idInt,
		PageToken:       "",
		ForceSendFields: nil,
	})
	if err != nil {
		return nil, err
	}
	return makeJobRemote(job), nil
}

func makeJobRemote(job *jobs.Job) *JobRemote {
	var settings jobs.JobSettings
	if job.Settings != nil {
		settings = *job.Settings
	}
	return &JobRemote{
		JobSettings:             settings,
		CreatedTime:             job.CreatedTime,
		CreatorUserName:         job.CreatorUserName,
		EffectiveBudgetPolicyId: job.EffectiveBudgetPolicyId,
		EffectiveUsagePolicyId:  job.EffectiveUsagePolicyId,
		JobId:                   job.JobId,
		RunAsUserName:           job.RunAsUserName,
		TriggerState:            job.TriggerState,
	}
}

// jobCreateCopy maps JobSettings (local state) to CreateJob (API request).
var jobCreateCopy = newCopy[jobs.JobSettings, jobs.CreateJob]()

func (r *ResourceJob) DoCreate(ctx context.Context, config *jobs.JobSettings) (string, *JobRemote, error) {
	response, err := r.client.Jobs.Create(ctx, *jobCreateCopy.Do(config))
	if err != nil {
		return "", nil, err
	}
	return strconv.FormatInt(response.JobId, 10), nil, nil
}

func (r *ResourceJob) DoUpdate(ctx context.Context, id string, config *jobs.JobSettings, _ Changes) (*JobRemote, error) {
	idInt, err := parseJobID(id)
	if err != nil {
		return nil, err
	}
	return nil, r.client.Jobs.Reset(ctx, jobs.ResetJob{
		JobId:       idInt,
		NewSettings: *config,
	})
}

func (r *ResourceJob) DoDelete(ctx context.Context, id string) error {
	idInt, err := parseJobID(id)
	if err != nil {
		return err
	}
	return r.client.Jobs.DeleteByJobId(ctx, idInt)
}

func parseJobID(id string) (int64, error) {
	result, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("internal error: job id is not integer: %q: %w", id, err)
	}
	return result, nil
}


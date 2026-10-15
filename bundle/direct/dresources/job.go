package dresources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	jobs "github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/sdktypes/sdkconv"
	"github.com/databricks/cli/libs/utils"
	sdkjobs "github.com/databricks/sdk-go/jobs/v2"
)

// JobRemote is the return type for DoRead. It embeds JobSettings so that all
// paths in StateType are valid paths in RemoteType.
type JobRemote struct {
	jobs.JobSettings

	// Remote-specific fields from jobs.GetJobResponse
	CreatedTime             *int64                `json:"created_time,omitempty"`
	CreatorUserName         *string               `json:"creator_user_name,omitempty"`
	EffectiveBudgetPolicyId *string               `json:"effective_budget_policy_id,omitempty"`
	EffectiveUsagePolicyId  *string               `json:"effective_usage_policy_id,omitempty"`
	JobId                   *int64                `json:"job_id,omitempty"`
	RunAsUserName           *string               `json:"run_as_user_name,omitempty"`
	TriggerDetails          []jobs.TriggerDetails `json:"trigger_details,omitempty"`
	TriggerState            *jobs.TriggerState    `json:"trigger_state,omitempty"`
}

type ResourceJob struct {
	client *sdkclients.Clients
}

func (*ResourceJob) New(client *sdkclients.Clients) *ResourceJob {
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

func getTaskKey(x jobs.TaskSettings) (string, string) {
	return "task_key", utils.Deref(x.TaskKey)
}

func getParameterName(x jobs.JobLevelParameter) (string, string) {
	return "name", utils.Deref(x.Name)
}

func getJobClusterKey(x jobs.JobCluster) (string, string) {
	return "job_cluster_key", utils.Deref(x.JobClusterKey)
}

func getEnvironmentKey(x jobs.JobEnvironment) (string, string) {
	return "environment_key", utils.Deref(x.EnvironmentKey)
}

func getDependsOnTaskKey(x jobs.TaskDependency) (string, string) {
	return "task_key", utils.Deref(x.TaskKey)
}

func getWebhookKey(x jobs.Webhook) (string, string) {
	return "id", utils.Deref(x.Id)
}

// The Jobs API returns webhook_notifications.on_* in an arbitrary order, so
// diff them by id to avoid a phantom diff that never converges.
var webhookNotificationEvents = []string{
	"on_start",
	"on_success",
	"on_failure",
	"on_duration_warning_threshold_exceeded",
	"on_streaming_backlog_exceeded",
}

// webhook_notifications appears at the job, task, and for_each task levels.
var webhookNotificationParents = []string{
	"webhook_notifications",
	"tasks[*].webhook_notifications",
	"tasks[*].for_each_task.task.webhook_notifications",
}

func (*ResourceJob) KeyedSlices() map[string]any {
	result := map[string]any{
		"tasks":                                  getTaskKey,
		"parameters":                             getParameterName,
		"job_clusters":                           getJobClusterKey,
		"environments":                           getEnvironmentKey,
		"tasks[*].depends_on":                    getDependsOnTaskKey,
		"tasks[*].for_each_task.task.depends_on": getDependsOnTaskKey,
	}
	for _, parent := range webhookNotificationParents {
		for _, event := range webhookNotificationEvents {
			result[parent+"."+event] = getWebhookKey
		}
	}
	return result
}

func (r *ResourceJob) DoRead(ctx context.Context, id string) (*JobRemote, error) {
	idInt, err := parseJobID(id)
	if err != nil {
		return nil, err
	}
	job, err := r.getJob(ctx, idInt)
	if err != nil {
		return nil, err
	}
	return makeJobRemote(job)
}

// getJob fetches the job with all its tasks. jobs/get returns at most 100
// tasks (and job clusters, parameters, environments) per page; later pages
// are fetched with next_page_token and merged.
func (r *ResourceJob) getJob(ctx context.Context, id int64) (*sdkjobs.GetJobResponse, error) {
	req := sdkjobs.GetJobRequest{
		JobId:               new(id),
		IncludeTriggerState: nil, // Not set by DABs
		PageToken:           nil, // Not set by DABs
	}
	job, err := r.client.Jobs.GetJob(ctx, req)
	if err != nil {
		return nil, err
	}
	for job.NextPageToken != nil && *job.NextPageToken != "" {
		req.PageToken = job.NextPageToken
		next, err := r.client.Jobs.GetJob(ctx, req)
		if err != nil {
			return nil, err
		}
		if job.Settings != nil && next.Settings != nil {
			job.Settings.Tasks = append(job.Settings.Tasks, next.Settings.Tasks...)
			job.Settings.JobClusters = append(job.Settings.JobClusters, next.Settings.JobClusters...)
			job.Settings.Parameters = append(job.Settings.Parameters, next.Settings.Parameters...)
			job.Settings.Environments = append(job.Settings.Environments, next.Settings.Environments...)
		}
		job.NextPageToken = next.NextPageToken
	}
	return job, nil
}

func makeJobRemote(resp *sdkjobs.GetJobResponse) (*JobRemote, error) {
	job, err := sdkconv.FromSDK[jobs.GetJobResponse](resp)
	if err != nil {
		return nil, err
	}
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
		TriggerDetails:          job.TriggerDetails,
		TriggerState:            job.TriggerState,
	}, nil
}

func (r *ResourceJob) DoCreate(ctx context.Context, config *jobs.JobSettings) (string, *JobRemote, error) {
	// access_control_list is not supported by DABs; permissions are a separate resource.
	request, err := sdkconv.ToSDK[sdkjobs.CreateJobRequest](config)
	if err != nil {
		return "", nil, err
	}
	response, err := r.client.Jobs.CreateJob(ctx, *request)
	if err != nil {
		return "", nil, err
	}
	return strconv.FormatInt(utils.Deref(response.JobId), 10), nil, nil
}

func (r *ResourceJob) DoUpdate(ctx context.Context, id string, config *jobs.JobSettings, _ *PlanEntry) (*JobRemote, error) {
	idInt, err := parseJobID(id)
	if err != nil {
		return nil, err
	}
	settings, err := jobs.JobSettingsToSDK(config)
	if err != nil {
		return nil, err
	}
	_, err = r.client.Jobs.ResetJob(ctx, sdkjobs.ResetJobRequest{
		JobId:       new(idInt),
		NewSettings: settings,
	})
	return nil, err
}

func (r *ResourceJob) DoDelete(ctx context.Context, id string, _ *jobs.JobSettings) error {
	idInt, err := parseJobID(id)
	if err != nil {
		return err
	}
	_, err = r.client.Jobs.DeleteJob(ctx, sdkjobs.DeleteJobRequest{JobId: new(idInt)})
	return err
}

func parseJobID(id string) (int64, error) {
	result, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("internal error: job id is not integer: %q: %w", id, err)
	}
	return result, nil
}

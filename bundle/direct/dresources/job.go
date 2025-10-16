package dresources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

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

func (*ResourceJob) RemapState(jobs *jobs.Job) (*jobs.JobSettings, error) {
	return jobs.Settings, nil
}

func (r *ResourceJob) DoRefresh(ctx context.Context, id string) (*jobs.Job, error) {
	idInt, err := parseJobID(id)
	if err != nil {
		return nil, err
	}
	return r.client.Jobs.GetByJobId(ctx, idInt)
}

func (r *ResourceJob) DoCreate(ctx context.Context, config *jobs.JobSettings) (string, error) {
	request, err := makeCreateJob(*config)
	if err != nil {
		return "", err
	}
	response, err := r.client.Jobs.Create(ctx, request)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(response.JobId, 10), nil
}

func (r *ResourceJob) DoUpdate(ctx context.Context, id string, config *jobs.JobSettings) error {
	request, err := makeResetJob(*config, id)
	if err != nil {
		return err
	}
	return r.client.Jobs.Reset(ctx, request)
}

func (r *ResourceJob) DoDelete(ctx context.Context, id string) error {
	idInt, err := parseJobID(id)
	if err != nil {
		return err
	}
	return r.client.Jobs.DeleteByJobId(ctx, idInt)
}

func makeCreateJob(config jobs.JobSettings) (jobs.CreateJob, error) {
	// Note, exhaustruct linter validates that all off CreateJob fields are initialized.
	// We don't have linter that validates that all of config fields are used.
	result := jobs.CreateJob{
		AccessControlList:    nil, // Not supported by DABs
		BudgetPolicyId:       config.BudgetPolicyId,
		Continuous:           config.Continuous,
		Deployment:           config.Deployment,
		Description:          config.Description,
		EditMode:             config.EditMode,
		EmailNotifications:   config.EmailNotifications,
		Environments:         config.Environments,
		Format:               config.Format,
		GitSource:            config.GitSource,
		Health:               config.Health,
		JobClusters:          config.JobClusters,
		MaxConcurrentRuns:    config.MaxConcurrentRuns,
		Name:                 config.Name,
		NotificationSettings: config.NotificationSettings,
		Parameters:           config.Parameters,
		PerformanceTarget:    config.PerformanceTarget,
		Queue:                config.Queue,
		RunAs:                config.RunAs,
		Schedule:             config.Schedule,
		Tags:                 config.Tags,
		Tasks:                config.Tasks,
		TimeoutSeconds:       config.TimeoutSeconds,
		Trigger:              config.Trigger,
		UsagePolicyId:        config.UsagePolicyId,
		WebhookNotifications: config.WebhookNotifications,
		ForceSendFields:      filterFields[jobs.CreateJob](config.ForceSendFields, "AccessControlList"),
	}

	return result, nil
}

func makeResetJob(config jobs.JobSettings, id string) (jobs.ResetJob, error) {
	idInt, err := parseJobID(id)
	if err != nil {
		return jobs.ResetJob{}, err
	}
	result := jobs.ResetJob{
		JobId:       idInt,
		NewSettings: config,
	}
	return result, err
}

func parseJobID(id string) (int64, error) {
	result, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("internal error: job id is not integer: %q: %w", id, err)
	}
	return result, nil
}

package tnresources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type ResourceJob struct {
	client      *databricks.WorkspaceClient
	config      jobs.JobSettings
	remoteState *jobs.Job
}

func NewResourceJob(client *databricks.WorkspaceClient, config *resources.Job) (*ResourceJob, error) {
	return &ResourceJob{
		client:      client,
		config:      config.JobSettings,
		remoteState: nil,
	}, nil
}

func (r *ResourceJob) Config() any {
	return r.config
}

func (r *ResourceJob) RemoteState() any {
	return r.remoteState
}

func (r *ResourceJob) DoRefresh(ctx context.Context, id string) error {
	idInt, err := parseJobID(id)
	if err != nil {
		return err
	}
	resp, err := r.client.Jobs.GetByJobId(ctx, idInt)
	if err != nil {
		return err
	}
	r.remoteState = resp
	return nil
}

func (r *ResourceJob) DoCreate(ctx context.Context) (string, error) {
	request, err := makeCreateJob(r.config)
	if err != nil {
		return "", err
	}
	response, err := r.client.Jobs.Create(ctx, request)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(response.JobId, 10), nil
}

func (r *ResourceJob) DoUpdate(ctx context.Context, id string) error {
	request, err := makeResetJob(r.config, id)
	if err != nil {
		return err
	}
	return r.client.Jobs.Reset(ctx, request)
}

func DeleteJob(ctx context.Context, client *databricks.WorkspaceClient, id string) error {
	idInt, err := parseJobID(id)
	if err != nil {
		return err
	}
	return client.Jobs.DeleteByJobId(ctx, idInt)
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
		ForceSendFields:      filterFields[jobs.CreateJob](config.ForceSendFields),
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

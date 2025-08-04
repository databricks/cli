package tnresources

import (
	"context"
	"strconv"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type ResourceJob struct {
	client *databricks.WorkspaceClient
	config jobs.JobSettings
}

func NewResourceJob(client *databricks.WorkspaceClient, job *resources.Job) (*ResourceJob, error) {
	return &ResourceJob{
		client: client,
		// TODO Use Processor with explicit field mapping
		config: job.JobSettings,
	}, nil
}

func (r *ResourceJob) Config() any {
	return r.config
}

func (r *ResourceJob) DoCreate(ctx context.Context) (string, error) {
	request, err := makeCreateJob(r.config)
	if err != nil {
		return "", err
	}
	response, err := r.client.Jobs.Create(ctx, request)
	if err != nil {
		return "", SDKError{Method: "Jobs.Create", Err: err}
	}
	return strconv.FormatInt(response.JobId, 10), nil
}

func (r *ResourceJob) DoUpdate(ctx context.Context, id string) error {
	request, err := makeResetJob(r.config, id)
	if err != nil {
		return err
	}
	err = r.client.Jobs.Reset(ctx, request)
	if err != nil {
		return SDKError{Method: "Jobs.Reset", Err: err}
	}
	return nil
}

func DeleteJob(ctx context.Context, client *databricks.WorkspaceClient, id string) error {
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}
	err = client.Jobs.DeleteByJobId(ctx, idInt)
	if err != nil {
		return SDKError{Method: "Jobs.DeleteByJobId", Err: err}
	}
	return nil
}

func (r *ResourceJob) WaitAfterCreate(ctx context.Context) error {
	// Intentional no-op
	return nil
}

func (r *ResourceJob) WaitAfterUpdate(ctx context.Context) error {
	// Intentional no-op
	return nil
}

func makeCreateJob(config jobs.JobSettings) (jobs.CreateJob, error) {
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
		WebhookNotifications: config.WebhookNotifications,

		ForceSendFields: filterForceSendFields[jobs.CreateJob](config.ForceSendFields),
	}

	// TODO: Validate copy - all fields must be initialized or explicitly allowed to be empty
	// Unset AccessControlList
	return result, nil
}

func makeResetJob(config jobs.JobSettings, id string) (jobs.ResetJob, error) {
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return jobs.ResetJob{}, err
	}
	result := jobs.ResetJob{JobId: idInt, NewSettings: config}
	// TODO: Validate copy - all fields must be initialized or explicitly allowed to be empty
	return result, err
}

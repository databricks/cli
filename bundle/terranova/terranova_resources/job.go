package terranova_resources

import (
	"context"
	"reflect"
	"strconv"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type ResourceJob struct {
	client *databricks.WorkspaceClient
	config jobs.JobSettings
}

func NewResourceJob(client *databricks.WorkspaceClient, job resources.Job) (ResourceJob, error) {
	return ResourceJob{
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

func (r *ResourceJob) DoUpdate(ctx context.Context, id string) (string, error) {
	request, err := makeResetJob(r.config, id)
	if err != nil {
		return "", err
	}
	err = r.client.Jobs.Reset(ctx, request)
	if err != nil {
		return "", SDKError{Method: "Jobs.Reset", Err: err}
	}
	return id, nil
}

func (r *ResourceJob) DoDelete(ctx context.Context, id string) error {
	return nil
}

func (r *ResourceJob) WaitAfterCreate(ctx context.Context) error {
	return nil
}

func (r *ResourceJob) WaitAfterUpdate(ctx context.Context) error {
	return nil
}

func (r *ResourceJob) ClassifyChanges(changes []structdiff.Change) ChangeType {
	return ChangeTypeUpdate
}

func makeCreateJob(config jobs.JobSettings) (jobs.CreateJob, error) {
	result := jobs.CreateJob{}
	// TODO: Validate copy - all fields must be initialized or explicitly allowed to be empty
	// Unset AccessControlList
	err := copyViaJSON(&result, config)
	return result, err
}

var jobSettingsType = reflect.TypeOf(jobs.JobSettings{})

func (r *ResourceJob) GetType() reflect.Type {
	return jobSettingsType
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

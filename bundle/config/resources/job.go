package resources

import (
	"context"
	"net/url"
	"reflect"
	"strconv"

	"github.com/databricks/cli/libs/jsonschema"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type Job struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	jobs.JobSettings

	Permissions []JobPermission `json:"permissions,omitempty"`
}

func (j *Job) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, j)
}

func (j Job) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(j)
}

func (j *Job) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	jobId, err := strconv.Atoi(id)
	if err != nil {
		return false, err
	}
	_, err = w.Jobs.Get(ctx, jobs.GetJobRequest{
		JobId: int64(jobId),
	})
	if err != nil {
		log.Debugf(ctx, "job %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (*Job) SchemaTransforms() map[reflect.Type]func(jsonschema.Schema) jsonschema.Schema {
	return map[reflect.Type]func(jsonschema.Schema) jsonschema.Schema{
		reflect.TypeFor[Job](): func(s jsonschema.Schema) jsonschema.Schema {
			// This field has been deprecated in jobs API v2.1 and is always set to
			// "MULTI_TASK" in the backend. We should not expose it to the user.
			delete(s.Properties, "format")

			// These fields are only meant to be set by the DABs client (ie the CLI)
			// and thus should not be exposed to the user. These are used to annotate
			// jobs that were created by DABs.
			delete(s.Properties, "deployment")
			delete(s.Properties, "edit_mode")
			return s
		},
		reflect.TypeFor[jobs.GitSource](): func(s jsonschema.Schema) jsonschema.Schema {
			// These fields are readonly and are not meant to be set by the user.
			delete(s.Properties, "job_source")
			delete(s.Properties, "git_snapshot")
			return s
		},
		// The CLI sets these to track the bundle deployment in the Deployment
		// Metadata Service; they are not user-configurable, so they must not
		// appear in the JSON schema or the generated annotation files. The
		// parent "deployment" block is already removed from the Job schema
		// above; this removes the fields from the type definition itself.
		reflect.TypeFor[jobs.JobDeployment](): func(s jsonschema.Schema) jsonschema.Schema {
			delete(s.Properties, "deployment_id")
			delete(s.Properties, "version_id")
			return s
		},
	}
}

func (j *Job) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "job",
		PluralName:    "jobs",
		SingularTitle: "Job",
		PluralTitle:   "Jobs",
	}
}

func (j *Job) InitializeURL(baseURL url.URL) {
	if j.ID == "" {
		return
	}
	j.URL = workspaceurls.ResourceURL(baseURL, "jobs", j.ID)
}

func (j *Job) GetName() string {
	return j.Name
}

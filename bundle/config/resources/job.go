package resources

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type Job struct {
	ID             string         `json:"id,omitempty" bundle:"readonly"`
	Permissions    []Permission   `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	URL            string         `json:"url,omitempty" bundle:"internal"`

	*jobs.JobSettings
}

func (s *Job) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s Job) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
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

func (j *Job) TerraformResourceName() string {
	return "databricks_job"
}

func (j *Job) InitializeURL(baseURL url.URL) {
	if j.ID == "" {
		return
	}
	baseURL.Path = fmt.Sprintf("jobs/%s", j.ID)
	j.URL = baseURL.String()
}

func (j *Job) GetName() string {
	return j.Name
}

func (j *Job) GetURL() string {
	return j.URL
}

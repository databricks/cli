package resources

import (
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	sdkjobs "github.com/databricks/sdk-go/jobs/v2"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"context"
	"net/url"
	"strconv"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type Job struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	jobs.JobSettings

	Permissions []JobPermission `json:"permissions,omitempty"`
}



func (j *Job) Exists(ctx context.Context, w *sdkclients.Clients, id string) (bool, error) {
	jobId, err := strconv.Atoi(id)
	if err != nil {
		return false, err
	}
	_, err = w.Jobs.GetJob(ctx, sdkjobs.GetJobRequest{
		JobId: new(int64(jobId)),
	})
	if err != nil {
		log.Debugf(ctx, "job %s does not exist", id)
		return false, err
	}
	return true, nil
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
	return utils.Deref(j.Name)
}

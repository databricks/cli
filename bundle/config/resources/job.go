package resources

import (
	"context"
	"strconv"

	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type Job struct {
	ID             string         `json:"id,omitempty" bundle:"readonly"`
	Permissions    []Permission   `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`

	paths.Paths

	*jobs.JobSettings
}

func (s *Job) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s Job) MarshalJSON() ([]byte, error) {
	// If local_ssd_count is set, we need to force send it
	p := dyn.NewPattern(
		dyn.Key("job_clusters"),
		dyn.AnyIndex(),
		dyn.Key("new_cluster"),
		dyn.Key("gcp_attributes"),
		dyn.Key("local_ssd_count"),
	)

	dyn.MapByPattern(s.DynamicValue, p, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		s.JobSettings.JobClusters[p[1].Index()].NewCluster.GcpAttributes.ForceSendFields = []string{"LocalSsdCount"}
		return v, nil
	})

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

package resources

import "github.com/databricks/databricks-sdk-go/service/jobs"

type Job struct {
	ID string `json:"id,omitempty"`

	*jobs.JobSettings
}

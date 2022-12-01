package resources

import "github.com/databricks/databricks-sdk-go/service/jobs"

type Workflow struct {
	ID string `json:"id,omitempty"`

	*jobs.JobSettings
}

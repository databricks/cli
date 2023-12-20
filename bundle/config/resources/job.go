package resources

import (
	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type Job struct {
	ID          string       `json:"id,omitempty" bundle:"readonly"`
	Permissions []Permission `json:"permissions,omitempty"`

	paths.Paths

	*jobs.JobSettings
}

func (s *Job) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s Job) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

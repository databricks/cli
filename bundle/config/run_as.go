package config

import "github.com/databricks/databricks-sdk-go/service/jobs"

type RunAs struct {
	jobs.JobRunAs

	// TODO: write comment explaint the purpose of this field
	// TODO: Add info about using this flag in a bundle in the diagnostic message.
	UseLegacy bool `json:"use_legacy,omitempty"`
}

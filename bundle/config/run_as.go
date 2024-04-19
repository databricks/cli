package config

import "github.com/databricks/databricks-sdk-go/service/jobs"

type RunAs struct {
	jobs.JobRunAs

	// Enable legacy run_as behavior. That is:
	// - Set the run_as identity as the owner of any pipelines in the bundle.
	// - Do not error in the presence of resources that do not support run_as.
	//   As of April 2024 this includes pipelines and model serving endpoints.
	//
	// This mode of run_as requires the deploying user to be a workspace and metastore
	// admin for working properly. Use of this flag is not recommend for new bundles,
	// and it is only provided for backward compatibility.
	UseLegacy bool `json:"use_legacy,omitempty"`
}

package config

type Deployment struct {
	// FailOnActiveRuns specifies whether to fail the deployment if there are
	// running jobs or pipelines in the workspace. Defaults to false.
	FailOnActiveRuns bool `json:"fail_on_active_runs,omitempty"`

	// Lock configures locking behavior on deployment.
	Lock Lock `json:"lock,omitempty"`

	// History reports what the deployment metadata service has recorded for this
	// bundle. Output only: it is read from the service for 'bundle summary' and is
	// unset when the bundle does not record deployment history.
	History *DeploymentHistory `json:"history,omitempty" bundle:"readonly"`
}

// DeploymentHistory identifies the bundle's deployment in the deployment
// metadata service.
type DeploymentHistory struct {
	// DeploymentID is the ID the service assigned to this bundle's deployment.
	DeploymentID string `json:"deployment_id,omitempty"`

	// LatestVersionID is the most recent version recorded for the deployment. It is
	// unset when the deployment exists but has no version yet.
	LatestVersionID string `json:"latest_version_id,omitempty"`
}

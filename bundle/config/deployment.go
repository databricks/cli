package config

type Deployment struct {
	// FailOnActiveRuns specifies whether to fail the deployment if there are
	// running jobs or pipelines in the workspace. Defaults to false.
	FailOnActiveRuns bool `json:"fail_on_active_runs,omitempty"`

	// Lock configures locking behavior on deployment.
	Lock Lock `json:"lock,omitempty"`

	// DeploymentID is the ID the service assigned to this bundle's deployment.
	DeploymentID string `json:"deployment_id,omitempty"`

	// LatestVersionID is the most recent version recorded for the deployment. It is
	// zero when the deployment exists but has no version yet.
	LatestVersionID int `json:"latest_version_id,omitempty"`
}

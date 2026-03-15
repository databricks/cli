package config

type Deployment struct {
	// FailOnActiveRuns specifies whether to fail the deployment if there are
	// running jobs or pipelines in the workspace. Defaults to false.
	FailOnActiveRuns bool `json:"fail_on_active_runs,omitempty"`

	// Lock configures locking behavior on deployment.
	Lock Lock `json:"lock,omitempty"`
}

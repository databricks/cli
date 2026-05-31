package config

type Deployment struct {
	// FailOnActiveRuns specifies whether to fail the deployment if there are
	// running jobs or pipelines in the workspace. Defaults to false.
	FailOnActiveRuns bool `json:"fail_on_active_runs,omitempty"`

	// Lock configures locking behavior on deployment.
	Lock Lock `json:"lock,omitempty"`

	// ManagedState opts the bundle into the deployment metadata service (DMS)
	// for locking and resource-state management. Can also be set with the
	// DATABRICKS_BUNDLE_MANAGED_STATE environment variable; configuration takes
	// priority over the environment variable.
	ManagedState *bool `json:"managed_state,omitempty"`
}

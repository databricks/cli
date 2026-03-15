package config

import "github.com/databricks/cli/bundle/config/engine"

type Deployment struct {
	// FailOnActiveRuns specifies whether to fail the deployment if there are
	// running jobs or pipelines in the workspace. Defaults to false.
	FailOnActiveRuns bool `json:"fail_on_active_runs,omitempty"`

	// Lock configures locking behavior on deployment.
	Lock Lock `json:"lock,omitempty"`

	// Engine specifies the deployment engine to use ("terraform" or "direct").
	// Can be overridden with the DATABRICKS_BUNDLE_ENGINE environment variable.
	Engine engine.EngineType `json:"engine,omitempty"`
}

package config

type Deployment struct {
	// FailOnActiveRuns specifies whether to fail the deployment if there are
	// running jobs or pipelines in the workspace. Defaults to false.
	FailOnActiveRuns bool `json:"fail_on_active_runs,omitempty"`

	// ImmutableFolder specifies that bundle files and artifacts are uploaded as a
	// single immutable snapshot rather than being synced individually. When true,
	// the deployment calls /api/2.0/repos/snapshots with a zip containing all files
	// and sets workspace.file_path and workspace.artifact_path to the returned
	// content-addressed path. validate and plan make no mutative API calls.
	ImmutableFolder bool `json:"immutable_folder,omitempty"`

	// Lock configures locking behavior on deployment.
	Lock Lock `json:"lock,omitempty"`
}

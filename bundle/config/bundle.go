package config

import "github.com/databricks/cli/bundle/config/engine"

type Terraform struct {
	ExecPath string `json:"exec_path"`
}

type Bundle struct {
	Name string `json:"name"`

	// TODO
	// Default cluster to run commands on (Python, Scala).
	// DefaultCluster string `json:"default_cluster,omitempty"`

	// TODO
	// Default warehouse to run SQL on.
	// DefaultWarehouse string `json:"default_warehouse,omitempty"`

	// Target is set by the mutator that selects the target.
	Target string `json:"target,omitempty" bundle:"readonly"`

	// DEPRECATED. Left for backward compatibility with Target
	Environment string `json:"environment,omitempty" bundle:"readonly"`

	// Terraform holds configuration related to Terraform.
	// For example, where to find the binary, which version to use, etc.
	Terraform *Terraform `json:"terraform,omitempty" bundle:"readonly"`

	// Force-override Git branch validation.
	Force bool `json:"force,omitempty" bundle:"readonly"`

	// Contains Git information like current commit, current branch and
	// origin url. Automatically loaded by reading .git directory if not specified
	Git Git `json:"git,omitempty"`

	// Determines the mode of the target.
	// For example, 'mode: development' can be used for deployments for
	// development purposes.
	// Annotated readonly as this should be set at the target level.
	Mode Mode `json:"mode,omitempty" bundle:"readonly"`

	// DEPRECATED: Overrides the compute used for jobs and other supported assets.
	ComputeId string `json:"compute_id,omitempty"`

	// Overrides the cluster used for jobs and other supported assets.
	ClusterId string `json:"cluster_id,omitempty"`

	// Engine specifies the deployment engine to use ("terraform" or "direct").
	// Can be overridden with the DATABRICKS_BUNDLE_ENGINE environment variable.
	Engine engine.EngineType `json:"engine,omitempty"`

	// Deployment section specifies deployment related configuration for bundle
	Deployment Deployment `json:"deployment,omitempty"`

	// Databricks CLI version constraints required to run the bundle.
	DatabricksCliVersion string `json:"databricks_cli_version,omitempty"`

	// A stable generated UUID for the bundle. This is normally serialized by
	// Databricks first party template when a user runs bundle init.
	Uuid string `json:"uuid,omitempty"`

	// Immutable specifies that bundle files and artifacts are uploaded as a single
	// immutable snapshot rather than being synced individually. When true, the
	// deployment calls /api/2.0/repos/snapshots with a zip containing all files
	// and sets workspace.file_path and workspace.artifact_path to the returned
	// content-addressed path. validate and plan make no mutative API calls.
	Immutable bool `json:"immutable,omitempty"`
}

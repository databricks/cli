package config

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

	// Transformers apply some transformation throughout the bundle, e.g.
	// adding a name prefix to deployed resources.
	Transformers Transformers `json:"mutators,omitempty"`

	// Overrides the compute used for jobs and other supported assets.
	ComputeID string `json:"compute_id,omitempty"`

	// Deployment section specifies deployment related configuration for bundle
	Deployment Deployment `json:"deployment,omitempty"`

	// Databricks CLI version constraints required to run the bundle.
	DatabricksCliVersion string `json:"databricks_cli_version,omitempty"`
}

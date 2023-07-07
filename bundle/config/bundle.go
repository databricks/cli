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

	// Environment is set by the mutator that selects the environment.
	Environment string `json:"environment,omitempty" bundle:"readonly"`

	// Terraform holds configuration related to Terraform.
	// For example, where to find the binary, which version to use, etc.
	Terraform *Terraform `json:"terraform,omitempty" bundle:"readonly"`

	// Lock configures locking behavior on deployment.
	Lock Lock `json:"lock" bundle:"readonly"`

	// Contains Git information like current commit, current branch and
	// origin url. Automatically loaded by reading .git directory if not specified
	Git Git `json:"git,omitempty"`

	// Determines the mode of the environment.
	// For example, 'mode: development' can be used for deployments for
	// development purposes.
	Mode Mode `json:"mode,omitempty" bundle:"readonly"`

	// Overrides the compute used for jobs and other supported assets.
	ComputeID string `json:"compute_id,omitempty"`
}

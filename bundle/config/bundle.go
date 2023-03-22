package config

type Terraform struct {
	ExecPath string `json:"exec_path"`
}

type Lock struct {
	// Force acquisition of deployment lock even if it is currently held.
	// This may be necessary if a prior deployment failed to release the lock.
	Force bool `json:"force"`
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
	Environment string `json:"environment,omitempty"`

	// Terraform holds configuration related to Terraform.
	// For example, where to find the binary, which version to use, etc.
	Terraform *Terraform `json:"terraform,omitempty"`

	// Lock configures the bundle's locking behavior on deployment.
	Lock Lock `json:"lock"`
}

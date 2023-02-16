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
	Environment string `json:"environment,omitempty"`

	// Terraform holds configuration related to Terraform.
	// For example, where to find the binary, which version to use, etc.
	Terraform *Terraform `json:"terraform,omitempty"`
}

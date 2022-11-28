package config

type Bundle struct {
	Name string `json:"name,omitempty"`

	// TODO
	// Default cluster to run commands on (Python, Scala).
	// DefaultCluster string `json:"default_cluster,omitempty"`

	// TODO
	// Default warehouse to run SQL on.
	// DefaultWarehouse string `json:"default_warehouse,omitempty"`

	// Environment is set by the mutator that selects the environment.
	Environment string `json:"environment,omitempty"`
}

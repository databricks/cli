package config

import "github.com/databricks/cli/ucm/config/engine"

// Ucm holds top-level metadata about a ucm deployment (parallel to
// bundle.Bundle in DAB).
type Ucm struct {
	// Name uniquely identifies this ucm deployment.
	Name string `json:"name"`

	// Engine selects the deployment engine ("terraform" or "direct").
	// Can be overridden with the DATABRICKS_UCM_ENGINE environment variable.
	Engine engine.EngineType `json:"engine,omitempty"`

	// Target records which target is currently selected. It is populated
	// by the SelectTarget mutator; users do not set it in ucm.yml.
	Target string `json:"target,omitempty" bundle:"readonly"`
}

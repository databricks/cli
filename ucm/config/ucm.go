package config

import "github.com/databricks/cli/ucm/config/engine"

// Ucm is the top-level ucm: block of ucm.yml. Mirrors bundle.Bundle for DABs,
// but scoped to Unity Catalog declarative management.
type Ucm struct {
	// Name identifies the ucm project. Required.
	Name string `json:"name"`

	// Engine selects the deployment engine ("terraform" or "direct").
	// Can be overridden with the DATABRICKS_UCM_ENGINE environment variable.
	Engine engine.EngineType `json:"engine,omitempty"`

	// Target records which target is currently selected. It is populated
	// by the SelectTarget mutator; users do not set it in ucm.yml.
	Target string `json:"target,omitempty" bundle:"readonly"`
}

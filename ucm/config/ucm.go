package config

// Ucm holds top-level metadata about a ucm deployment (parallel to
// bundle.Bundle in DAB).
type Ucm struct {
	// Name uniquely identifies this ucm deployment.
	Name string `json:"name"`

	// Target records which target is currently selected. It is populated
	// by the SelectTarget mutator; users do not set it in ucm.yml.
	Target string `json:"target,omitempty" bundle:"readonly"`
}

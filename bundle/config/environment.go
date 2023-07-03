package config

type Mode string

// Environment defines overrides for a single environment.
// This structure is recursively merged into the root configuration.
type Environment struct {
	// Default marks that this environment must be used if one isn't specified
	// by the user (through environment variable or command line argument).
	Default bool `json:"default,omitempty"`

	Mode Mode `json:"mode,omitempty"`

	Bundle *Bundle `json:"bundle,omitempty"`

	Workspace *Workspace `json:"workspace,omitempty"`

	Artifacts map[string]*Artifact `json:"artifacts,omitempty"`

	Resources *Resources `json:"resources,omitempty"`

	// Override default values for defined variables
	// Does not permit defining new variables or redefining existing ones
	// in the scope of an environment
	Variables map[string]string `json:"variables,omitempty"`
}

const (
	// Right now, we just have a default / "" mode and a "development" mode.
	// Additional modes are expected to come for pull-requests and production.
	Development Mode = "development"
)

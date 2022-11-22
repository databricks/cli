package config

// Workspace defines configurables at the workspace level.
type Workspace struct {
	// TODO: Add all unified authentication configurables.
	Host    string `json:"host,omitempty"`
	Profile string `json:"profile,omitempty"`
}

package config

// Workspace defines configurables at the workspace level.
type Workspace struct {
	Host    string `json:"host,omitempty"`
	Profile string `json:"profile,omitempty"`
}

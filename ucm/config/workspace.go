package config

// Workspace describes a Databricks workspace that ucm targets.
//
// M0 holds only the host URL. Profile was added to support the
// databrickscfg-profile resolution flow mirroring DAB (see
// ucm/workspace_client.go). Fuller auth wiring (OAuth M2M client id/secret,
// account id) lands in M1 along with the real deploy path.
type Workspace struct {
	Host    string `json:"host,omitempty"`
	Profile string `json:"profile,omitempty"`
}

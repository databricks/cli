package config

// Workspace describes a Databricks workspace that ucm targets.
//
// M0 holds only the host URL. Auth wiring (OAuth M2M client id/secret,
// account id, profile resolution) lands in M1 along with the real deploy
// path.
type Workspace struct {
	Host string `json:"host,omitempty"`
}

package config

import (
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
)

// Workspace describes a Databricks workspace that ucm targets.
//
// M0 holds only the host URL. Profile was added to support the
// databrickscfg-profile resolution flow mirroring DAB (see
// ucm/workspace_client.go). Fuller auth wiring (OAuth M2M client id/secret,
// account id) lands in M1 along with the real deploy path.
type Workspace struct {
	Host    string `json:"host,omitempty"`
	Profile string `json:"profile,omitempty"`

	// RootPath is the workspace filesystem root for this deployment. Defaults
	// to "~/databricks/ucm/<name>/<target>" via DefineDefaultWorkspaceRoot and
	// is expanded to "/Workspace/Users/<user>/..." by ExpandWorkspaceRoot.
	RootPath string `json:"root_path,omitempty"`

	// StatePath is the workspace sub-path that holds the remote state artifacts
	// (terraform.tfstate, ucm-state.json). Defaults to "<RootPath>/state" via
	// DefineDefaultWorkspacePaths.
	StatePath string `json:"state_path,omitempty"`
}

// Config returns the SDK config built from this Workspace's auth fields.
// Mirrors bundle/config/workspace.go's Workspace.Config(); ucm's M0 surface
// is only Host+Profile, so the returned config is intentionally minimal.
func (w *Workspace) Config() *sdkconfig.Config {
	return &sdkconfig.Config{
		Host:    w.Host,
		Profile: w.Profile,
	}
}

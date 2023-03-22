package config

import (
	"os"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/scim"
)

type PathLike struct {
	// Workspace contains a WSFS path.
	Workspace string `json:"workspace,omitempty"`

	// DBFS contains a DBFS path.
	DBFS string `json:"dbfs,omitempty"`
}

// IsSet returns whether either path is non-nil.
func (p PathLike) IsSet() bool {
	return p.Workspace != "" || p.DBFS != ""
}

type Lock struct {
	// Enabled toggles deployment lock. True by default.
	Enabled *bool `json:"enabled"`

	// Force acquisition of deployment lock even if it is currently held.
	// This may be necessary if a prior deployment failed to release the lock.
	Force bool `json:"force"`
}

func (lock Lock) IsEnabled() bool {
	if lock.Enabled != nil {
		return *lock.Enabled
	}
	return true
}

// Workspace defines configurables at the workspace level.
type Workspace struct {
	// Unified authentication attributes.
	//
	// We omit sensitive attributes as they should never be hardcoded.
	// They must be specified through environment variables instead.
	//
	// For example: token, password, Google credentials, Azure client secret, etc.
	//

	// Generic attributes.
	Host    string `json:"host,omitempty"`
	Profile string `json:"profile,omitempty"`

	// Google specific attributes.
	GoogleServiceAccount string `json:"google_service_account,omitempty"`

	// Azure specific attributes.
	AzureResourceID  string `json:"azure_workspace_resource_id,omitempty"`
	AzureUseMSI      bool   `json:"azure_use_msi,omitempty"`
	AzureClientID    string `json:"azure_client_id,omitempty"`
	AzureTenantID    string `json:"azure_tenant_id,omitempty"`
	AzureEnvironment string `json:"azure_environment,omitempty"`
	AzureLoginAppID  string `json:"azure_login_app_id,omitempty"`

	// CurrentUser holds the current user.
	// This is set after configuration initialization.
	CurrentUser *scim.User `json:"current_user,omitempty"`

	// Remote base path for deployment state, for artifacts, as synchronization target.
	// This defaults to "~/.bundle/${bundle.name}/${bundle.environment}" where "~" expands to
	// the current user's home directory in the workspace (e.g. `/Users/jane@doe.com`).
	Root string `json:"root,omitempty"`

	// Remote path to synchronize local files to.
	// This defaults to "${workspace.root}/files".
	FilePath PathLike `json:"file_path,omitempty"`

	// Remote path for build artifacts.
	// This defaults to "${workspace.root}/artifacts".
	ArtifactPath PathLike `json:"artifact_path,omitempty"`

	// Remote path for deployment state.
	// This defaults to "${workspace.root}/state".
	StatePath PathLike `json:"state_path,omitempty"`

	// Lock configures locking behavior on deployment.
	Lock Lock `json:"lock"`
}

func (w *Workspace) Client() (*databricks.WorkspaceClient, error) {
	config := databricks.Config{
		// Generic
		Host:    w.Host,
		Profile: w.Profile,

		// Google
		GoogleServiceAccount: w.GoogleServiceAccount,

		// Azure
		AzureResourceID:  w.AzureResourceID,
		AzureUseMSI:      w.AzureUseMSI,
		AzureClientID:    w.AzureClientID,
		AzureTenantID:    w.AzureTenantID,
		AzureEnvironment: w.AzureEnvironment,
		AzureLoginAppID:  w.AzureLoginAppID,
	}

	return databricks.NewWorkspaceClient(&config)
}

func init() {
	os.Setenv("BRICKS_CLI_PATH", os.Args[0])
}

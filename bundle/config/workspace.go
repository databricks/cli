package config

import (
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

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
	CurrentUser *iam.User `json:"current_user,omitempty" bundle:"readonly"`

	// Remote workspace base path for deployment state, for artifacts, as synchronization target.
	// This defaults to "~/.bundle/${bundle.name}/${bundle.environment}" where "~" expands to
	// the current user's home directory in the workspace (e.g. `/Users/jane@doe.com`).
	RootPath string `json:"root_path,omitempty"`

	// Remote workspace path to synchronize local files to.
	// This defaults to "${workspace.root}/files".
	FilesPath string `json:"file_path,omitempty"`

	// Remote workspace path for build artifacts.
	// This defaults to "${workspace.root}/artifacts".
	ArtifactsPath string `json:"artifact_path,omitempty"`

	// Remote workspace path for deployment state.
	// This defaults to "${workspace.root}/state".
	StatePath string `json:"state_path,omitempty"`
}

func (w *Workspace) Client() (*databricks.WorkspaceClient, error) {
	cfg := databricks.Config{
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

	// HACKY fix to not used host based auth when the profile is already set
	profile := os.Getenv("DATABRICKS_CONFIG_PROFILE")

	// If only the host is configured, we try and unambiguously match it to
	// a profile in the user's databrickscfg file. Override the default loaders.
	if w.Host != "" && w.Profile == "" && profile == "" {
		cfg.Loaders = []config.Loader{
			// Load auth creds from env vars
			config.ConfigAttributes,

			// Our loader that resolves a profile from the host alone.
			// This only kicks in if the above loaders don't configure auth.
			databrickscfg.ResolveProfileFromHost,
		}
	}

	return databricks.NewWorkspaceClient(&cfg)
}

func init() {
	arg0 := os.Args[0]

	// Configure DATABRICKS_CLI_PATH only if our caller intends to use this specific version of this binary.
	// Otherwise, if it is equal to its basename, processes can find it in $PATH.
	if arg0 != filepath.Base(arg0) {
		os.Setenv("DATABRICKS_CLI_PATH", arg0)
	}
}

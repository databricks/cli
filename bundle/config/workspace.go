package config

import (
	"github.com/databricks/databricks-sdk-go"
)

type PathLike struct {
	// Workspace contains a WSFS path.
	Workspace *string `json:"workspace,omitempty"`

	// DBFS contains a DBFS path.
	DBFS *string `json:"dbfs,omitempty"`
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

	// Remote path for artifacts.
	// This can specify a workspace path, a DBFS path, or both.
	// Some artifacts must be stored in the workspace (e.g. notebooks).
	// Some artifacts must be stored on DBFS (e.g. wheels, JARs).
	ArtifactPath PathLike `json:"artifact_path"`
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

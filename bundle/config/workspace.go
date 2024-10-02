package config

import (
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/marshal"
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
	Host               string `json:"host,omitempty"`
	Profile            string `json:"profile,omitempty"`
	AuthType           string `json:"auth_type,omitempty"`
	MetadataServiceURL string `json:"metadata_service_url,omitempty" bundle:"internal"`

	// OAuth specific attributes.
	ClientID string `json:"client_id,omitempty"`

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
	CurrentUser *User `json:"current_user,omitempty" bundle:"readonly"`

	// Remote workspace base path for deployment state, for artifacts, as synchronization target.
	// This defaults to "~/.bundle/${bundle.name}/${bundle.target}" where "~" expands to
	// the current user's home directory in the workspace (e.g. `/Workspace/Users/jane@doe.com`).
	RootPath string `json:"root_path,omitempty"`

	// Remote workspace path to synchronize local files to.
	// This defaults to "${workspace.root}/files".
	FilePath string `json:"file_path,omitempty"`

	// Remote workspace path for resources with a presence in the workspace.
	// These are kept outside [FilePath] to avoid potential naming collisions.
	// This defaults to "${workspace.root}/resources".
	ResourcePath string `json:"resource_path,omitempty"`

	// Remote workspace path for build artifacts.
	// This defaults to "${workspace.root}/artifacts".
	ArtifactPath string `json:"artifact_path,omitempty"`

	// Remote workspace path for deployment state.
	// This defaults to "${workspace.root}/state".
	StatePath string `json:"state_path,omitempty"`
}

type User struct {
	// A short name for the user, based on the user's UserName.
	ShortName string `json:"short_name,omitempty" bundle:"readonly"`

	*iam.User
}

func (s *User) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s User) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (w *Workspace) Config() *config.Config {
	cfg := &config.Config{
		// Generic
		Host:               w.Host,
		Profile:            w.Profile,
		AuthType:           w.AuthType,
		MetadataServiceURL: w.MetadataServiceURL,

		// OAuth
		ClientID: w.ClientID,

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

	for k := range config.ConfigAttributes {
		attr := &config.ConfigAttributes[k]
		if !attr.IsZero(cfg) {
			cfg.SetAttrSource(attr, config.Source{Type: config.SourceType("bundle")})
		}
	}

	return cfg
}

func (w *Workspace) Client() (*databricks.WorkspaceClient, error) {
	cfg := w.Config()

	// If only the host is configured, we try and unambiguously match it to
	// a profile in the user's databrickscfg file. Override the default loaders.
	if w.Host != "" && w.Profile == "" {
		cfg.Loaders = []config.Loader{
			// Load auth creds from env vars
			config.ConfigAttributes,

			// Our loader that resolves a profile from the host alone.
			// This only kicks in if the above loaders don't configure auth.
			databrickscfg.ResolveProfileFromHost,
		}
	}

	// Resolve the configuration. This is done by [databricks.NewWorkspaceClient] as well, but here
	// we need to verify that a profile, if loaded, matches the host configured in the bundle.
	err := cfg.EnsureResolved()
	if err != nil {
		return nil, err
	}

	// Now that the configuration is resolved, we can verify that the host in the bundle configuration
	// is identical to the host associated with the selected profile.
	if w.Host != "" && w.Profile != "" {
		err := databrickscfg.ValidateConfigAndProfileHost(cfg, w.Profile)
		if err != nil {
			return nil, err
		}
	}

	return databricks.NewWorkspaceClient((*databricks.Config)(cfg))
}

func init() {
	arg0 := os.Args[0]

	// Configure DATABRICKS_CLI_PATH only if our caller intends to use this specific version of this binary.
	// Otherwise, if it is equal to its basename, processes can find it in $PATH.
	if arg0 != filepath.Base(arg0) {
		os.Setenv("DATABRICKS_CLI_PATH", arg0)
	}
}

package config

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/liteswap"
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

	// Unified host specific attributes.
	//
	// ExperimentalIsUnifiedHost is a deprecated no-op. Unified hosts are now
	// detected automatically from /.well-known/databricks-config. The field is
	// retained so existing databricks.yml files using it still validate against
	// the bundle schema.
	ExperimentalIsUnifiedHost bool   `json:"experimental_is_unified_host,omitempty"`
	AccountID                 string `json:"account_id,omitempty"`
	WorkspaceID               string `json:"workspace_id,omitempty"`

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

	// A short name for the user that is stripped off of non-alphanumeric character
	// Can be used as a prefix for resources that use their name as part of their URL (e.g. Apps, Database Instances)
	DomainFriendlyName string `json:"domain_friendly_name,omitempty" bundle:"readonly"`

	*iam.User
}

func (s *User) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s User) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (w *Workspace) Config(ctx context.Context) *config.Config {
	// Once bundle deploy started, old deployment is partially destroyed, so we should do utmost to complete it.
	// Having client-side timeouts that kill the deployment seems counter-productive. We should just keep on
	// trying and the user should be the one interrupting it if they decide so.
	// Default is 30s
	httpTimeout := 90
	if v, ok := env.HTTPTimeoutSeconds(ctx); ok {
		if n, err := strconv.Atoi(v); err == nil {
			httpTimeout = n
		}
	}

	cfg := &config.Config{
		HTTPTimeoutSeconds: httpTimeout,

		// Default is 5min
		RetryTimeoutSeconds: 15 * 60,

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
		AzureLoginAppID:  w.AzureLoginAppID, //nolint:staticcheck // SA1019: still accepted in bundle config; forwarded until the SDK drops the field

		// Unified host
		AccountID:   w.AccountID,
		WorkspaceID: w.WorkspaceID,
	}
	liteswap.Apply(ctx, cfg)

	for k := range config.ConfigAttributes {
		attr := &config.ConfigAttributes[k]
		if !attr.IsZero(cfg) {
			cfg.SetAttrSource(attr, config.Source{Type: config.SourceType("bundle")})
		}
	}

	return cfg
}

// NormalizeHostURL extracts query parameters from the host URL and populates
// the corresponding fields if not already set. This allows users to paste SPOG
// URLs (e.g. https://host.databricks.com/?o=12345) directly into their bundle
// config. Must be called before Config() so the extracted fields are included
// in the SDK config used for profile resolution and authentication.
func (w *Workspace) NormalizeHostURL() {
	params := auth.ExtractHostQueryParams(w.Host)
	w.Host = params.Host
	if w.WorkspaceID == "" {
		w.WorkspaceID = params.WorkspaceID
	}
	if w.AccountID == "" {
		w.AccountID = params.AccountID
	}
}

func (w *Workspace) Client(ctx context.Context) (*databricks.WorkspaceClient, error) {
	// Extract query parameters (?o=, ?a=) from the host URL before building
	// the SDK config. This ensures workspace_id and account_id are available
	// for profile resolution during EnsureResolved().
	w.NormalizeHostURL()

	cfg := w.Config(ctx)

	switch {
	case w.Profile != "":
		// An explicit profile wins over auth env vars (#5096).
		// ValidateConfigAndProfileHost below still checks host agreement.
		cfg.Loaders = databrickscfg.ProfileAuthLoaders
	case w.Host != "":
		// If only the host is configured, we try and unambiguously match it to
		// a profile in the user's databrickscfg file. Override the default loaders.
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

// StrictProfile returns the profile that the target must authenticate with when
// strict profile matching is enabled. Each workspace profile allows one pair of
// host and workspace ID, and a profile without a workspace_id allows targets
// without one. Account profiles are skipped. The target's pair must match exactly
// one profile. Anything else is an error, so credentials from the environment
// only reach a host and workspace that a profile allows.
func (w *Workspace) StrictProfile(ctx context.Context) (string, error) {
	// Match the host and workspace ID that Client resolves, without rewriting
	// the target's configuration.
	target := *w
	target.NormalizeHostURL()
	if target.Host == "" {
		return "", errors.New("strict profile matching: workspace.host must be set")
	}

	withHost := profile.WithHost(target.Host)
	profiles, err := profile.GetProfiler(ctx).LoadProfiles(ctx, func(p profile.Profile) bool {
		return withHost(p) && profile.MatchWorkspaceProfiles(p)
	})
	if err != nil {
		return "", err
	}
	if len(profiles) == 0 {
		return "", fmt.Errorf("strict profile matching: no workspace profile matches host %s", target.Host)
	}

	var matches profile.Profiles
	for _, p := range profiles {
		if p.WorkspaceID == target.WorkspaceID {
			matches = append(matches, p)
		}
	}
	switch {
	case len(matches) == 1:
		return matches[0].Name, nil
	case len(matches) > 1:
		return "", fmt.Errorf("strict profile matching: multiple profiles match host %s: %s", target.Host, strings.Join(matches.Names(), ", "))
	case target.WorkspaceID == "":
		return "", fmt.Errorf("strict profile matching: host %s requires a workspace ID; set workspace.workspace_id or add ?o=<workspace-id> to workspace.host", target.Host)
	default:
		return "", fmt.Errorf("strict profile matching: no profile for host %s has workspace_id %s", target.Host, target.WorkspaceID)
	}
}

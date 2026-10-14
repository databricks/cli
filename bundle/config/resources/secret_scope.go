package resources

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/secrets/v1"
	"github.com/databricks/cli/libs/utils"
	sdksecrets "github.com/databricks/sdk-go/secrets/v1"
	"net/url"
)

type SecretScopePermissionLevel string

const (
	SecretScopePermissionLevelRead   SecretScopePermissionLevel = "READ"
	SecretScopePermissionLevelWrite  SecretScopePermissionLevel = "WRITE"
	SecretScopePermissionLevelManage SecretScopePermissionLevel = "MANAGE"
)

// Values enables generated enum validation. Order matches privilege rank (READ < WRITE < MANAGE).
func (SecretScopePermissionLevel) Values() []SecretScopePermissionLevel {
	return []SecretScopePermissionLevel{
		SecretScopePermissionLevelRead,
		SecretScopePermissionLevelWrite,
		SecretScopePermissionLevelManage,
	}
}

// SecretScopePermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any secret scope.
// Secret scopes permissions are mapped to Secret ACLs
type SecretScopePermission struct {
	Level SecretScopePermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type SecretScope struct { //nolint:recvcheck // pointer receiver needed for UnmarshalJSON, value for other methods
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`

	// A unique name to identify the secret scope.
	Name string `json:"name"`

	Permissions []SecretScopePermission `json:"permissions,omitempty"`

	// Secret scope configuration is explicitly defined here with individual fields
	// to maintain API stability and prevent unintended configuration changes.
	// This approach decouples our configuration from potential upstream model/SDK changes
	// to `workspace.SecretScope`. While the upstream type serves as a response payload
	// for workspace.ListScopesResponse, we adopt its field naming conventions
	// for better developer experience compared to `workspace.CreateScope`.

	// The type of secret scope backend.
	BackendType secrets.ScopeBackendType `json:"backend_type,omitempty"`
	// The metadata for the secret scope if the type is `AZURE_KEYVAULT`
	KeyvaultMetadata *secrets.AzureKeyVaultSecretScopeMetadata `json:"keyvault_metadata,omitempty"`
}

func (s SecretScope) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	// NOTE: Scope lookup by name is not directly supported by the Secret scopes API
	// As of May 2025 there is no direct API method to retrieve a scope using its name as an identifier.
	// While scope names serve as unique identifiers, the API only provides:
	// - List operations that returns a list of scopes
	// - Other operational methods (e.g., reading a secret from a scope and parsing error messages)
	//
	// The indirect methods are not semantically ideal for simple existence checks, so we use the list API here
	resp, err := w.Secrets.ListScopes(ctx, sdksecrets.ListScopesRequest{})
	if err != nil {
		return false, nil //nolint:nilerr // treat API errors as "scope not found"
	}

	for _, scope := range resp.Scopes {
		if utils.Deref(scope.Name) == name {
			return true, nil
		}
	}

	return false, nil
}

func (s SecretScope) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "secret_scope",
		PluralName:    "secret_scopes",
		SingularTitle: "Secret Scope",
		PluralTitle:   "Secret Scopes",
	}
}

func (s SecretScope) GetName() string {
	if s.ID != "" {
		return s.ID
	}
	return s.Name
}

func (s SecretScope) GetURL() (string, bool) {
	// Secret scopes do not have a URL
	return "", false
}

func (s SecretScope) InitializeURL(_ url.URL) {
	// Secret scopes do not have a URL
}

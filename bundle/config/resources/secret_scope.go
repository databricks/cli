package resources

import (
	"context"
	"net/url"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type SecretScopePermissionLevel string

// SecretScopePermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any secret scope.
// Secret scopes permissions are mapped to Secret ACLs
type SecretScopePermission struct {
	Level SecretScopePermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type SecretScope struct {
	// ID is Name that is stored in resources state, usually the same as Name unless re-deployment is pending.
	ID string `json:"id,omitempty" bundle:"readonly"`

	// A unique name to identify the secret scope.
	Name string `json:"name"`

	Permissions    []SecretScopePermission `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus          `json:"modified_status,omitempty" bundle:"internal"`

	// Secret scope configuration is explicitly defined here with individual fields
	// to maintain API stability and prevent unintended configuration changes.
	// This approach decouples our configuration from potential upstream model/SDK changes
	// to `workspace.SecretScope`. While the upstream type serves as a response payload
	// for workspace.ListScopesResponse, we adopt its field naming conventions
	// for better developer experience compared to `workspace.CreateScope`.

	// The type of secret scope backend.
	BackendType workspace.ScopeBackendType `json:"backend_type,omitempty"`
	// The metadata for the secret scope if the type is `AZURE_KEYVAULT`
	KeyvaultMetadata *workspace.AzureKeyVaultSecretScopeMetadata `json:"keyvault_metadata,omitempty"`
}

func (s *SecretScope) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s SecretScope) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s SecretScope) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	// NOTE: Scope lookup by name is not directly supported by the Secret scopes API
	// As of May 2025 there is no direct API method to retrieve a scope using its name as an identifier.
	// While scope names serve as unique identifiers, the API only provides:
	// - List operations that returns a list of scopes
	// - Other operational methods (e.g., reading a secret from a scope and parsing error messages)
	//
	// The indirect methods are not semantically ideal for simple existence checks, so we use the list API here
	scopes, err := w.Secrets.ListScopesAll(ctx)
	if err != nil {
		return false, nil
	}

	for _, scope := range scopes {
		if scope.Name == name {
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

func (s SecretScope) GetURL() string {
	// Secret scopes do not have a URL
	return ""
}

func (s SecretScope) InitializeURL(_ url.URL) {
	// Secret scopes do not have a URL
}

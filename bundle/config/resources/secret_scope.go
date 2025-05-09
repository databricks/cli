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
	Name string `json:"name"`

	Permissions    []SecretScopePermission `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus          `json:"modified_status,omitempty" bundle:"internal"`

	*workspace.SecretScope
}

func (s *SecretScope) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s SecretScope) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s SecretScope) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
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
		SingularName:          "secret_scope",
		PluralName:            "secret_scopes",
		SingularTitle:         "Secret Scope",
		PluralTitle:           "Secret Scopes",
		TerraformResourceName: "databricks_secret_scope",
	}
}

func (s SecretScope) TerraformResourceName() string {
	return "databricks_secret_scope"
}

func (s SecretScope) GetName() string {
	return s.Name
}

func (s SecretScope) GetURL() string {
	// Secret scopes do not have a URL
	return ""
}

func (s SecretScope) InitializeURL(_ url.URL) {
	// Secret scopes do not have a URL
}

func (s SecretScope) IsNil() bool {
	return s.SecretScope == nil
}

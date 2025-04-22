package resources

import (
	"context"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"net/url"
)

type SecretScope struct {
	Name                   string `json:"name"`
	InitialManagePrincipal string `json:"initial_manage_principal"`

	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	URL            string         `json:"url,omitempty" bundle:"internal"`

	*workspace.SecretScope
}

func (s *SecretScope) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s SecretScope) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s SecretScope) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (s SecretScope) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:          "secret_scope",
		PluralName:            "secret_scopes",
		SingularTitle:         "Secret Scope",
		PluralTitle:           "Secret Scope",
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
	return s.URL
}

func (s SecretScope) InitializeURL(baseURL url.URL) {
	//TODO implement me
	panic("implement me")
}

func (s SecretScope) IsNil() bool {
	return s.SecretScope == nil
}

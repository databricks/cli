package resources

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type Schema struct {
	// List of grants to apply on this schema.
	Grants []Grant `json:"grants,omitempty"`

	// Full name of the schema (catalog_name.schema_name). This value is read from
	// the terraform state after deployment succeeds.
	ID string `json:"id,omitempty" bundle:"readonly"`

	*catalog.CreateSchema

	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	URL            string         `json:"url,omitempty" bundle:"internal"`
}

func (s *Schema) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	return false, errors.New("schema.Exists() is not supported")
}

func (s *Schema) TerraformResourceName() string {
	return "databricks_schema"
}

func (s *Schema) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	baseURL.Path = "explore/data/" + strings.ReplaceAll(s.ID, ".", "/")
	s.URL = baseURL.String()
}

func (s *Schema) GetURL() string {
	return s.URL
}

func (s *Schema) GetName() string {
	return s.Name
}

func (s *Schema) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s Schema) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *Schema) IsNil() bool {
	return s.CreateSchema == nil
}

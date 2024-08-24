package resources

import (
	"context"
	"fmt"
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
	return false, fmt.Errorf("schema.Exists() is not supported")
}

func (s *Schema) TerraformResourceName() string {
	return "databricks_schema"
}

func (s *Schema) InitializeURL(urlPrefix string, urlSuffix string) {
	if s.ID == "" {
		return
	}
	s.URL = urlPrefix + "explore/data/" + strings.ReplaceAll(s.ID, ".", "/") + urlSuffix
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

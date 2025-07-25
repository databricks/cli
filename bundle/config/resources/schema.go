package resources

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/databricks-sdk-go/apierr"

	"github.com/databricks/cli/libs/log"

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

	catalog.CreateSchema

	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	URL            string         `json:"url,omitempty" bundle:"internal"`
}

func (s *Schema) Exists(ctx context.Context, w *databricks.WorkspaceClient, fullName string) (bool, error) {
	log.Tracef(ctx, "Checking if schema with fullName=%s exists", fullName)

	_, err := w.Schemas.GetByFullName(ctx, fullName)
	if err != nil {
		log.Debugf(ctx, "schema with full name %s does not exist: %v", fullName, err)

		if apierr.IsMissing(err) {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (*Schema) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "schema",
		PluralName:    "schemas",
		SingularTitle: "Schema",
		PluralTitle:   "Schemas",
	}
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

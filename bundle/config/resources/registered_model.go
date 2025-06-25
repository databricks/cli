package resources

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type RegisteredModel struct {
	// This is a resource agnostic implementation of grants.
	// Implementation could be different based on the resource type.
	Grants []Grant `json:"grants,omitempty"`

	// This represents the id which is the full name of the model
	// (catalog_name.schema_name.model_name) that can be used
	// as a reference in other resources. This value is returned by terraform.
	ID string `json:"id,omitempty" bundle:"readonly"`

	// This represents the input args for terraform, and will get converted
	// to a HCL representation for CRUD
	catalog.CreateRegisteredModelRequest

	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	URL            string         `json:"url,omitempty" bundle:"internal"`
}

func (s *RegisteredModel) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s RegisteredModel) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *RegisteredModel) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.RegisteredModels.Get(ctx, catalog.GetRegisteredModelRequest{
		FullName: id,
	})
	if err != nil {
		log.Debugf(ctx, "registered model %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (*RegisteredModel) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "registered_model",
		PluralName:    "registered_models",
		SingularTitle: "Registered Model",
		PluralTitle:   "Registered Models",
	}
}

func (s *RegisteredModel) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	baseURL.Path = "explore/data/models/" + strings.ReplaceAll(s.ID, ".", "/")
	s.URL = baseURL.String()
}

func (s *RegisteredModel) GetName() string {
	return s.Name
}

func (s *RegisteredModel) GetURL() string {
	return s.URL
}

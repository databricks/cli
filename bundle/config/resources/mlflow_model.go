package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type MlflowModelPermissionLevel string

// MlflowModelPermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any model.
type MlflowModelPermission struct {
	Level MlflowModelPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type MlflowModel struct {
	ID             string                  `json:"id,omitempty" bundle:"readonly"`
	Permissions    []MlflowModelPermission `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus          `json:"modified_status,omitempty" bundle:"internal"`
	URL            string                  `json:"url,omitempty" bundle:"internal"`

	ml.CreateModelRequest
}

func (s *MlflowModel) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s MlflowModel) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *MlflowModel) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.ModelRegistry.GetModel(ctx, ml.GetModelRequest{
		Name: id,
	})
	if err != nil {
		log.Debugf(ctx, "model %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (j *MlflowModel) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "model",
		PluralName:    "models",
		SingularTitle: "Model",
		PluralTitle:   "Models",
	}
}

func (s *MlflowModel) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	baseURL.Path = "ml/models/" + s.ID
	s.URL = baseURL.String()
}

func (s *MlflowModel) GetName() string {
	return s.Name
}

func (s *MlflowModel) GetURL() string {
	return s.URL
}

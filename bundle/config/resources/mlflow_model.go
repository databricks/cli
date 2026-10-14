package resources

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/modelregistry/v1"
	"github.com/databricks/cli/libs/utils"
	sdkmodelregistry "github.com/databricks/sdk-go/modelregistry/v1"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type MlflowModel struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	modelregistry.CreateRegisteredModelRequest

	Permissions []MlflowModelPermission `json:"permissions,omitempty"`
}

func (s *MlflowModel) Exists(ctx context.Context, w *sdkclients.Clients, id string) (bool, error) {
	_, err := w.ModelRegistry.GetRegisteredModelDatabricks(ctx, sdkmodelregistry.GetRegisteredModelDatabricksRequest{
		Name: new(id),
	})
	if err != nil {
		log.Debugf(ctx, "model %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (s *MlflowModel) ResourceDescription() ResourceDescription {
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
	s.URL = workspaceurls.ResourceURL(baseURL, "models", s.ID)
}

func (s *MlflowModel) GetName() string {
	return utils.Deref(s.Name)
}

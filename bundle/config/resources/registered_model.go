package resources

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/uc/grants/v1"
	"github.com/databricks/cli/libs/sdktypes/uc/registeredmodels/v1"
	"github.com/databricks/cli/libs/utils"
	sdkregisteredmodels "github.com/databricks/sdk-go/uc/registeredmodels/v1"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type RegisteredModel struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`

	registeredmodels.CreateRegisteredModelRequest

	// List of grants to apply on this registered model.
	Grants []grants.PrivilegeAssignment `json:"grants,omitempty"`
}

func (s *RegisteredModel) Exists(ctx context.Context, w *sdkclients.Clients, id string) (bool, error) {
	_, err := w.RegisteredModels.GetRegisteredModel(ctx, sdkregisteredmodels.GetRegisteredModelRequest{
		FullNameArg: new(id),
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
	s.URL = workspaceurls.ResourceURL(baseURL, "registered_models", s.ID)
}

func (s *RegisteredModel) GetName() string {
	return utils.Deref(s.Name)
}

package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/aigateway/v1"
	"github.com/databricks/cli/libs/sdktypes/uc/grants/v1"
	sdkaigateway "github.com/databricks/sdk-go/aigateway/v1"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

// ModelServiceConfig is the bundle-authored state for an AI Gateway model
// service.
//
// The SDK models the create inputs `parent` and `model_service_id` as URL
// parameters (`json:"-"`) that sit outside the ModelService body, and derives
// the resource `name` (`model-services/{catalog}.{schema}.{model_service}`)
// server-side. We therefore cannot embed catalog.CreateModelServiceRequest the
// way volume embeds catalog.CreateVolumeRequestContent: its identity fields
// would be invisible to the bundle schema. Instead we expose a flat struct with
// the immutable identity (parent + model_service_id) plus the mutable body.
//
// Owner is intentionally not exposed yet (mirrors volume, which does not manage
// owner): the API returns effective_owner rather than owner on read, so round
// tripping it needs extra care. See the direct engine resource for the CRUD.
type ModelServiceConfig struct {
	// Parent schema, format `schemas/{catalog}.{schema}`. Immutable: the server
	// derives `name` from parent + model_service_id, so changing it recreates
	// the resource.
	Parent string `json:"parent"`
	// Leaf id of the model service, e.g. "my_model_service". Immutable.
	ModelServiceId string `json:"model_service_id"`
	// User-provided description.
	Comment string `json:"comment,omitempty"`
	// Operational configuration: destinations, routing, rate limits, inference
	// table.
	Config *aigateway.ModelServiceConfig `json:"config,omitempty"`
}

type ModelService struct {
	BaseResource
	ModelServiceConfig
	ID string `json:"id,omitempty" bundle:"readonly"`

	// List of grants to apply on this model service. Grants are applied via the
	// permissions API (the `.grants` sub-resource), not the model-service body, so
	// this field lives on the wrapper and stays out of the diffed state that
	// PrepareState returns (the embedded ModelServiceConfig).
	Grants []grants.PrivilegeAssignment `json:"grants,omitempty"`
}

// UnmarshalJSON / MarshalJSON are defined on the wrapper so it does not inherit
// ModelServiceConfig's promoted marshaler, which would silently drop the
// BaseResource fields (id, url, lifecycle, modified_status).

func (m *ModelService) Exists(ctx context.Context, w *sdkclients.Clients, id string) (bool, error) {
	// The engine tracks the id as the bare {catalog}.{schema}.{model_service};
	// the API addresses the resource by its full name.
	_, err := w.AiGateway.GetModelService(ctx, sdkaigateway.GetModelServiceRequest{Name: new("model-services/" + id)})
	if err != nil {
		log.Debugf(ctx, "model service %s does not exist", id)
		if apierrs.IsMissing(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (*ModelService) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "model_service",
		PluralName:    "model_services",
		SingularTitle: "Model service",
		PluralTitle:   "Model services",
	}
}

func (m *ModelService) InitializeURL(baseURL url.URL) {
	if m.ID == "" {
		return
	}
	// The id is the bare {catalog}.{schema}.{model_service}; ResourceURL splits
	// it into the Catalog Explorer path explore/data/model-services/...
	m.URL = workspaceurls.ResourceURL(baseURL, "model_services", m.ID)
}

func (m *ModelService) GetName() string {
	return m.ModelServiceId
}

package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// ModelProviderServiceConfig is the bundle-authored state for an AI Gateway
// model provider service.
//
// It mirrors ModelServiceConfig: the SDK models the create inputs `parent` and
// `model_provider_service_id` as URL parameters (`json:"-"`) outside the
// ModelProviderService body and derives the resource `name`
// (`model-provider-services/{catalog}.{schema}.{model_provider_service}`)
// server-side, so we expose a flat struct with the immutable identity plus the
// mutable body. Owner is not exposed yet (the API returns effective_owner on
// read, not owner).
//
// Note: config.provider_type is immutable after create (the API rejects
// changing it via Update); it is classified recreate_on_changes in
// resources.yml.
type ModelProviderServiceConfig struct {
	// Parent schema, format `schemas/{catalog}.{schema}`. Immutable: the server
	// derives `name` from parent + model_provider_service_id, so changing it
	// recreates the resource.
	Parent string `json:"parent"`
	// Leaf id of the model provider service, e.g. "openai_prod". Immutable.
	ModelProviderServiceId string `json:"model_provider_service_id"`
	// User-provided description.
	Comment string `json:"comment,omitempty"`
	// Behavioral configuration: provider connection, model catalog, passthrough
	// policy. config.provider_type is required at create and immutable after.
	Config *catalog.ModelProviderServiceConfig `json:"config,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`
}

func (c *ModelProviderServiceConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c ModelProviderServiceConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

type ModelProviderService struct {
	BaseResource
	ModelProviderServiceConfig
}

func (m *ModelProviderService) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	// The engine tracks the id as the bare
	// {catalog}.{schema}.{model_provider_service}; the API addresses the
	// resource by its full name.
	_, err := w.AiGateway.GetModelProviderService(ctx, catalog.GetModelProviderServiceRequest{Name: "model-provider-services/" + id})
	if err != nil {
		log.Debugf(ctx, "model provider service %s does not exist", id)
		if apierr.IsMissing(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (*ModelProviderService) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "model_provider_service",
		PluralName:    "model_provider_services",
		SingularTitle: "Model provider service",
		PluralTitle:   "Model provider services",
	}
}

func (m *ModelProviderService) InitializeURL(_ url.URL) {
	// AI Gateway model provider services do not have a Catalog Explorer URL
	// wired up here yet; leave URL unset until the UI route is confirmed.
}

func (m *ModelProviderService) GetName() string {
	return m.ModelProviderServiceId
}

func (m *ModelProviderService) GetURL() string {
	return m.URL
}

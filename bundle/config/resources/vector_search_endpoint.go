package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/vectorsearch/v1"
	"github.com/databricks/cli/libs/utils"
	sdkvectorsearch "github.com/databricks/sdk-go/vectorsearch/v1"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type VectorSearchEndpoint struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	vectorsearch.CreateEndpointRequest

	Permissions []VectorSearchEndpointPermission `json:"permissions,omitempty"`
}

func (e *VectorSearchEndpoint) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.VectorSearch.GetEndpoint(ctx, sdkvectorsearch.GetEndpointRequest{Name: new(name)})
	if err != nil {
		log.Debugf(ctx, "vector search endpoint %s does not exist: %v", name, err)
		if apierrs.IsMissing(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (e *VectorSearchEndpoint) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "vector_search_endpoint",
		PluralName:    "vector_search_endpoints",
		SingularTitle: "Vector Search Endpoint",
		PluralTitle:   "Vector Search Endpoints",
	}
}

func (e *VectorSearchEndpoint) InitializeURL(baseURL url.URL) {
	if e.Name == nil {
		return
	}
	e.URL = workspaceurls.ResourceURL(baseURL, "vector_search_endpoints", *e.Name)
}

func (e *VectorSearchEndpoint) GetName() string {
	return utils.Deref(e.Name)
}

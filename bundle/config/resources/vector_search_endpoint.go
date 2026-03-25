package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

type VectorSearchEndpoint struct {
	BaseResource
	vectorsearch.CreateEndpoint
}

func (e *VectorSearchEndpoint) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, e)
}

func (e VectorSearchEndpoint) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(e)
}

func (e *VectorSearchEndpoint) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.VectorSearchEndpoints.GetEndpoint(ctx, vectorsearch.GetEndpointRequest{EndpointName: name})
	if err != nil {
		log.Debugf(ctx, "vector search endpoint %s does not exist", name)
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
	if e.ID == "" {
		return
	}
	baseURL.Path = "compute/vector-search/" + e.Name
	e.URL = baseURL.String()
}

func (e *VectorSearchEndpoint) GetName() string {
	return e.Name
}

func (e *VectorSearchEndpoint) GetURL() string {
	return e.URL
}

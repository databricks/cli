package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

type VectorSearchEndpoint struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	vectorsearch.CreateEndpoint

	Permissions []VectorSearchEndpointPermission `json:"permissions,omitempty"`
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
		log.Debugf(ctx, "vector search endpoint %s does not exist: %v", name, err)
		if apierr.IsMissing(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (*VectorSearchEndpoint) RunAsSupport() RunAsSupport {
	return RunAsNotApplicable
}

func (*VectorSearchEndpoint) Approval() Approval {
	return Approval{}
}

func (e *VectorSearchEndpoint) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "vector_search_endpoint",
		PluralName:    "vector_search_endpoints",
		SingularTitle: "Vector Search Endpoint",
		PluralTitle:   "Vector Search Endpoints",
	}
}

func (*VectorSearchEndpoint) PermissionsSupport() PermissionsSupport {
	return PermissionsSupport{
		// https://docs.databricks.com/aws/en/security/auth/access-control/#vector-search-endpoint-acls
		BundleLevels: BundlePermissionLevels{
			CanManage: "CAN_MANAGE",
			CanView:   "CAN_USE",
		},
		RequestObjectType:  "vector-search-endpoints",
		ResponseObjectType: "vector-search-endpoints",
		ObjectIDField:      "endpoint_uuid", // endpoint UUID, not the endpoint name used as deployment ID
	}
}

func (*VectorSearchEndpoint) WorkspaceURL() (workspaceurls.Pattern, bool) {
	return workspaceurls.Pattern{Path: "compute/vector-search/%s"}, true
}

func (e *VectorSearchEndpoint) InitializeURL(baseURL url.URL) {
	if e.Name == "" {
		return
	}
	e.URL = workspaceURL(e, baseURL, e.Name)
}

func (*VectorSearchEndpoint) NamePrefixExemptReason() string {
	// GET/UPDATE/DELETE address the endpoint by name.
	return "the endpoint name is the primary key, so prefixing it would change the resource's identity rather than just its display name"
}

func (e *VectorSearchEndpoint) GetName() string {
	return e.Name
}

package resources

import (
	"context"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/uc/grants/v1"
	"github.com/databricks/cli/libs/sdktypes/vectorsearch/v1"
	"github.com/databricks/cli/libs/utils"
	sdkvectorsearch "github.com/databricks/sdk-go/vectorsearch/v1"
	"net/url"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type VectorSearchIndex struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	vectorsearch.CreateVectorIndexRequest

	// List of grants to apply on this vector search index.
	Grants []grants.PrivilegeAssignment `json:"grants,omitempty"`
}

func (e *VectorSearchIndex) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.VectorSearch.GetVectorIndex(ctx, sdkvectorsearch.GetVectorIndexRequest{Name: new(name)})
	if err != nil {
		log.Debugf(ctx, "vector search index %s does not exist: %v", name, err)
		if apierrs.IsMissing(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (e *VectorSearchIndex) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "vector_search_index",
		PluralName:    "vector_search_indexes",
		SingularTitle: "Vector Search Index",
		PluralTitle:   "Vector Search Indexes",
	}
}

func (e *VectorSearchIndex) InitializeURL(baseURL url.URL) {
	// UC explore expects /{catalog}/{schema}/{name}, so bail if the name isn't
	// a fully resolved three-part identifier; an unresolved ${...} reference
	// would otherwise produce a misleading URL.
	if strings.Count(utils.Deref(e.Name), ".") != 2 {
		return
	}
	e.URL = workspaceurls.ResourceURL(baseURL, "vector_search_indexes", *e.Name)
}

func (e *VectorSearchIndex) GetName() string {
	return utils.Deref(e.Name)
}

package resources

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

type VectorSearchIndex struct {
	BaseResource
	vectorsearch.CreateVectorIndexRequest

	// List of grants to apply on this vector search index.
	Grants []catalog.PrivilegeAssignment `json:"grants,omitempty"`
}

func (e *VectorSearchIndex) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, e)
}

func (e VectorSearchIndex) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(e)
}

func (e *VectorSearchIndex) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.VectorSearchIndexes.GetIndexByIndexName(ctx, name)
	if err != nil {
		log.Debugf(ctx, "vector search index %s does not exist: %v", name, err)
		if apierr.IsMissing(err) {
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
	if strings.Count(e.Name, ".") != 2 {
		return
	}
	e.URL = workspaceurls.ResourceURL(baseURL, "vector_search_indexes", e.Name)
}

func (e *VectorSearchIndex) GetName() string {
	return e.Name
}

func (e *VectorSearchIndex) GetURL() string {
	return e.URL
}

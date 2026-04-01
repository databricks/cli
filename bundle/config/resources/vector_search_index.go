package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

type VectorSearchIndex struct {
	BaseResource
	vectorsearch.CreateVectorIndexRequest
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
	if e.ID == "" {
		return
	}
	baseURL.Path = "compute/vector-search/indexes/" + e.Name
	e.URL = baseURL.String()
}

func (e *VectorSearchIndex) GetName() string {
	return e.Name
}

func (e *VectorSearchIndex) GetURL() string {
	return e.URL
}

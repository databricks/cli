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

func (i *VectorSearchIndex) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, i)
}

func (i VectorSearchIndex) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(i)
}

func (i *VectorSearchIndex) Exists(ctx context.Context, w *databricks.WorkspaceClient, indexName string) (bool, error) {
	_, err := w.VectorSearchIndexes.GetIndex(ctx, vectorsearch.GetIndexRequest{
		IndexName: indexName,
	})
	if err != nil {
		log.Debugf(ctx, "vector search index %s does not exist: %v", indexName, err)
		if apierr.IsMissing(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (*VectorSearchIndex) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "vector_search_index",
		PluralName:    "vector_search_indexes",
		SingularTitle: "Vector Search Index",
		PluralTitle:   "Vector Search Indexes",
	}
}

func (i *VectorSearchIndex) InitializeURL(baseURL url.URL) {
	if i.ID == "" {
		return
	}
	baseURL.Path = "explore/vector-search/" + i.EndpointName + "/" + i.ID
	i.URL = baseURL.String()
}

func (i *VectorSearchIndex) GetURL() string {
	return i.URL
}

func (i *VectorSearchIndex) GetName() string {
	return i.Name
}

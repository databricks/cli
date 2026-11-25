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
	ID string `json:"id,omitempty" bundle:"readonly"`
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

func (*VectorSearchIndex) RunAsSupport() RunAsSupport {
	return RunAsNotApplicable
}

func (*VectorSearchIndex) Approval() Approval {
	return Approval{
		Order: 100,
		Deploy: `
This action will result in the deletion or recreation of the following Vector Search indexes.
Recreating a Delta Sync index re-runs the full embedding pipeline; recreating a Direct Access
index drops all upserted vectors. Both can be expensive to rebuild:`,
		Destroy: `This action will result in the deletion of the following Vector Search indexes.
For Delta Sync indexes, the source Delta Table is preserved but the embedding pipeline is removed.
For Direct Access indexes, all upserted vectors are permanently lost:`,
	}
}

func (e *VectorSearchIndex) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "vector_search_index",
		PluralName:    "vector_search_indexes",
		SingularTitle: "Vector Search Index",
		PluralTitle:   "Vector Search Indexes",
	}
}

func (*VectorSearchIndex) GrantsSupport() GrantsSupport {
	return GrantsSupport{SecurableType: "table"}
}

func (*VectorSearchIndex) WorkspaceURL() (workspaceurls.Pattern, bool) {
	return workspaceurls.Pattern{Path: "explore/data/%s", DotSeparatedID: true}, true
}

func (e *VectorSearchIndex) InitializeURL(baseURL url.URL) {
	// UC explore expects /{catalog}/{schema}/{name}, so bail if the name isn't
	// a fully resolved three-part identifier; an unresolved ${...} reference
	// would otherwise produce a misleading URL.
	if strings.Count(e.Name, ".") != 2 {
		return
	}
	e.URL = workspaceURL(e, baseURL, e.Name)
}

func (*VectorSearchIndex) NamePrefixExemptReason() string {
	// CreateIndex addresses the index by name and DoCreate returns it as the deployment id.
	return "the 3-part UC name (catalog.schema.index) is the API primary key, so prefixing it would change the resource's identity rather than just its display name"
}

func (e *VectorSearchIndex) GetName() string {
	return e.Name
}

func (e *VectorSearchIndex) UCNames() []UCName {
	return []UCName{ucThreePartName(&e.Name)}
}

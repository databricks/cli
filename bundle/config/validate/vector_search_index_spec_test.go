package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVectorSearchIndexSpec(t *testing.T) {
	cases := []struct {
		name      string
		index     resources.VectorSearchIndex
		wantError string
	}{
		{
			name: "delta sync with delta spec is valid",
			index: resources.VectorSearchIndex{
				CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{
					Name:               "main.default.idx",
					IndexType:          vectorsearch.VectorIndexTypeDeltaSync,
					DeltaSyncIndexSpec: &vectorsearch.DeltaSyncVectorIndexSpecRequest{},
				},
			},
		},
		{
			name: "direct access with direct spec is valid",
			index: resources.VectorSearchIndex{
				CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{
					Name:                  "main.default.idx",
					IndexType:             vectorsearch.VectorIndexTypeDirectAccess,
					DirectAccessIndexSpec: &vectorsearch.DirectAccessVectorIndexSpec{},
				},
			},
		},
		{
			name: "delta sync without delta spec is rejected",
			index: resources.VectorSearchIndex{
				CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{
					Name:      "main.default.idx",
					IndexType: vectorsearch.VectorIndexTypeDeltaSync,
				},
			},
			wantError: `vector_search_indexes: missing delta_sync_index_spec for index_type "DELTA_SYNC"`,
		},
		{
			name: "delta sync with direct spec is rejected",
			index: resources.VectorSearchIndex{
				CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{
					Name:                  "main.default.idx",
					IndexType:             vectorsearch.VectorIndexTypeDeltaSync,
					DeltaSyncIndexSpec:    &vectorsearch.DeltaSyncVectorIndexSpecRequest{},
					DirectAccessIndexSpec: &vectorsearch.DirectAccessVectorIndexSpec{},
				},
			},
			wantError: `vector_search_indexes: direct_access_index_spec is not allowed when index_type is "DELTA_SYNC"`,
		},
		{
			name: "direct access without direct spec is rejected",
			index: resources.VectorSearchIndex{
				CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{
					Name:      "main.default.idx",
					IndexType: vectorsearch.VectorIndexTypeDirectAccess,
				},
			},
			wantError: `vector_search_indexes: missing direct_access_index_spec for index_type "DIRECT_ACCESS"`,
		},
		{
			name: "direct access with delta spec is rejected",
			index: resources.VectorSearchIndex{
				CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{
					Name:                  "main.default.idx",
					IndexType:             vectorsearch.VectorIndexTypeDirectAccess,
					DeltaSyncIndexSpec:    &vectorsearch.DeltaSyncVectorIndexSpecRequest{},
					DirectAccessIndexSpec: &vectorsearch.DirectAccessVectorIndexSpec{},
				},
			},
			wantError: `vector_search_indexes: delta_sync_index_spec is not allowed when index_type is "DIRECT_ACCESS"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						VectorSearchIndexes: map[string]*resources.VectorSearchIndex{
							"idx": &tc.index,
						},
					},
				},
			}
			diags := VectorSearchIndexSpec().Apply(t.Context(), b)
			if tc.wantError == "" {
				require.Empty(t, diags, "expected no diagnostics, got %v", diags)
				return
			}
			require.Len(t, diags, 1)
			assert.Equal(t, tc.wantError, diags[0].Summary)
		})
	}
}

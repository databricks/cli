package dresources

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	registerTestConfig("vector_search_indexes", &resources.VectorSearchIndex{
		Name:         "main.default.my_index",
		EndpointName: "my-index-endpoint",
		PrimaryKey:   "id",
		IndexType:    vectorsearch.VectorIndexTypeDeltaSync,
		DeltaSyncIndexSpec: &vectorsearch.DeltaSyncVectorIndexSpecRequest{
			SourceTable:  "main.default.source_table",
			PipelineType: vectorsearch.PipelineTypeTriggered,
		},
		Grants: []catalog.PrivilegeAssignment{{
			Principal:  "user@example.com",
			Privileges: []catalog.Privilege{catalog.PrivilegeSelect},
		}},
	})

	registerTestDeps("vector_search_indexes", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		_, err := client.VectorSearchEndpoints.CreateEndpoint(ctx, vectorsearch.CreateEndpoint{
			Name:         "my-index-endpoint",
			EndpointType: vectorsearch.EndpointTypeStandard,
		})
		if err != nil {
			return nil, err
		}

		return testConfig["vector_search_indexes"], nil
	})

	registerTestDeps("vector_search_indexes.grants", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		return &GrantsState{
			SecurableType: "table",
			FullName:      "main.default.my_index",
			EmbeddedSlice: []catalog.PrivilegeAssignment{{
				Privileges: []catalog.Privilege{catalog.PrivilegeSelect},
				Principal:  "user@example.com",
			}},
		}, nil
	})

	registerCreateRecoveryOptOut("vector_search_indexes", "the read response has no create time")
}

// TestVectorSearchIndexAllSDKFieldsAreClassified guards against a future SDK
// bump silently adding a field that the planner classifies as Update. The
// resource has no update API and intentionally omits DoUpdate, so any
// unclassified field would surface as a deploy-time framework error
// ("resource does not support update action but plan produced update"). This
// test catches the gap at unit-test time instead.
func TestVectorSearchIndexAllSDKFieldsAreClassified(t *testing.T) {
	config := GetResourceConfig("vector_search_indexes")
	require.NotNil(t, config)

	classified := map[string]bool{}
	for _, field := range config.RecreateOnChanges {
		classified[field.Field.String()] = true
	}
	// provided_id_fields also recreate on local changes, so they are classified.
	for _, field := range config.ProvidedIDFields {
		classified[field.Field.String()] = true
	}
	for _, field := range config.IgnoreRemoteChanges {
		classified[field.Field.String()] = true
	}

	sdkType := reflect.TypeFor[vectorsearch.CreateVectorIndexRequest]()
	for field := range sdkType.Fields() {
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		jsonTag = strings.TrimSuffix(jsonTag, ",omitempty")
		assert.Truef(t, classified[jsonTag],
			"field %q is not declared in vector_search_indexes.yml; "+
				"vector_search_indexes has no update API, so every SDK field must be in "+
				"recreate_on_changes, provided_id_fields or ignore_remote_changes",
			jsonTag,
		)
	}
}

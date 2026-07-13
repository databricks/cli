package dresources

import (
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			"field %q is not declared in resources.yml under vector_search_indexes; "+
				"vector_search_indexes has no update API, so every SDK field must be in "+
				"recreate_on_changes, provided_id_fields or ignore_remote_changes",
			jsonTag,
		)
	}
}

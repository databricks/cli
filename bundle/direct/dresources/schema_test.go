package dresources

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceSchema_DoUpdate_WithUnsupportedForceSendFields(t *testing.T) {
	_, client := setupTestServerClient(t)

	adapter := (*ResourceSchema)(nil).New(client)
	ctx := t.Context()

	config := &SchemaState{
		CatalogName:     "main",
		Name:            "test_schema",
		Comment:         "original comment",
		Properties:      map[string]string{"key": "value"},
		StorageRoot:     "",
		ForceSendFields: nil,
		SchemaId:        "",
	}

	id, _, err := adapter.DoCreate(ctx, config)
	require.NoError(t, err)

	config.Comment = "updated comment"
	config.Properties = map[string]string{"key": "updated_value"}
	config.ForceSendFields = []string{
		"Comment",
		"Properties",
		"EnablePredictiveOptimization", // Unsupported - should be filtered out
		"NewName",                      // Unsupported - should be filtered out
		"Owner",                        // Unsupported - should be filtered out
	}

	_, err = adapter.DoUpdate(ctx, id, config, &PlanEntry{})
	require.NoError(t, err)

	result, err := adapter.DoRead(ctx, id)
	require.NoError(t, err)

	result.CreatedAt = 0
	result.UpdatedAt = 0
	result.SchemaId = ""

	resultJSON, err := json.Marshal(result)
	require.NoError(t, err)
	// browse_only is force-sent from create onwards, matching UC, which returns it on
	// every read including after an update.
	expected := `{
		"browse_only": false,
		"catalog_name": "main",
		"catalog_type": "MANAGED_CATALOG",
		"created_at": 0,
		"created_by": "tester@databricks.com",
		"comment": "updated comment",
		"custom_max_retention_hours": 0,
		"effective_predictive_optimization_flag": {
			"inherited_from_name": "deco-uc-prod-isolated-aws-us-east-1",
			"inherited_from_type": "METASTORE",
			"value": "ENABLE"
		},
		"enable_predictive_optimization": "INHERIT",
		"properties": {"key": "updated_value"},
		"full_name": "main.test_schema",
		"metastore_id": "120efa64-9b68-46ba-be38-f319458430d2",
		"name": "test_schema",
		"owner": "tester@databricks.com",
		"updated_at": 0,
		"updated_by": "tester@databricks.com",
		"schema_id": ""
	}`
	assert.JSONEq(t, expected, string(resultJSON))
}

// schemaCreateScript plays the given responses, in order, to the schema create and get requests.
// A response is a catalog.SchemaInfo or a testserver.Response for an error.
type schemaCreateScript struct {
	creates []any
	gets    []any

	createCount int
	getCount    int
}

func (s *schemaCreateScript) createSchema(t *testing.T) (*catalog.SchemaInfo, string, error) {
	server := testserver.New(t)
	server.Handle("POST", "/api/2.1/unity-catalog/schemas", func(req testserver.Request) any {
		s.createCount++
		require.LessOrEqual(t, s.createCount, len(s.creates), "unexpected create request")
		return s.creates[s.createCount-1]
	})
	server.Handle("GET", "/api/2.1/unity-catalog/schemas/{full_name}", func(req testserver.Request) any {
		s.getCount++
		require.LessOrEqual(t, s.getCount, len(s.gets), "unexpected get request")
		return s.gets[s.getCount-1]
	})
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)
	me, err := iamutil.GetCurrentUser(t.Context(), client)
	require.NoError(t, err)

	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	_, remote, err := (*ResourceSchema)(nil).New(client).DoCreate(ctx, &SchemaState{
		CatalogName: "main", Name: "s",
	})
	return remote, me.UserName, err
}

func testSchema(createdBy string) catalog.SchemaInfo {
	return catalog.SchemaInfo{CatalogName: "main", Name: "s", FullName: "main.s", CreatedBy: createdBy, CreatedAt: time.Now().UnixMilli()}
}

func TestSchemaCreateRecoveryRejectsSchemaOfAnotherUser(t *testing.T) {
	s := &schemaCreateScript{
		creates: []any{featureTimeout},
		gets:    []any{testSchema("someone-else@example.com")},
	}
	_, _, err := s.createSchema(t)
	require.EqualError(t, err, "schema main.s exists but was created by another user")
	requireStatus(t, err, 504)
	assert.Equal(t, 1, s.createCount)
}

func TestSchemaCreateRecoveryLooksAgainAfterSchemaAlreadyExists(t *testing.T) {
	notFound := featureError(404, "SCHEMA_DOES_NOT_EXIST", "Schema 'main.s' does not exist.")
	alreadyExists := featureError(400, "SCHEMA_ALREADY_EXISTS", "Schema 's' already exists")
	// The first request lands after the first lookup: the repeat is rejected because the schema
	// exists, and the next lookup finds it.
	s := &schemaCreateScript{
		creates: []any{featureTimeout, alreadyExists},
		gets:    []any{notFound, testSchema(testserver.TestUser.UserName)},
	}
	remote, user, err := s.createSchema(t)
	require.NoError(t, err)
	assert.Equal(t, user, remote.CreatedBy)
	assert.Equal(t, 2, s.createCount)
	assert.Equal(t, 2, s.getCount)
}

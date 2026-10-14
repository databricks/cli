package prompt_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/apps/manifest"
	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/httpfixtures"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testBranch          = "projects/p1/branches/main"
	endpointsResource   = "/api/2.0/postgres/" + testBranch + "/endpoints"
	databasesResource   = "/api/2.0/postgres/" + testBranch + "/databases"
	readWriteEndpoint   = `{"endpoints":[{"name":"projects/p1/branches/main/endpoints/ep1","status":{"endpoint_type":"ENDPOINT_TYPE_READ_WRITE","hosts":{"host":"my-host.example.com"}}}]}`
	unavailableResponse = `{"error_code":"TEMPORARILY_UNAVAILABLE","message":"API unavailable"}`
)

func newPostgresResource() manifest.Resource {
	return manifest.Resource{
		ResourceKey: "postgres",
		Fields: map[string]manifest.ResourceField{
			"branch":       {Description: "branch path"},
			"database":     {Description: "database name"},
			"host":         {Resolve: "postgres:host"},
			"databaseName": {Resolve: "postgres:databaseName"},
			"endpointPath": {Resolve: "postgres:endpointPath"},
		},
	}
}

// postgresContext serves the fixtures and returns a context with a workspace
// client pointing at them. Unexpected requests fail the test.
func postgresContext(t *testing.T, fixtures httpfixtures.Fixtures) context.Context {
	cfg, _ := fixtures.Config(t)
	w, err := sdkclients.New(t.Context(), cfg)
	require.NoError(t, err)
	return cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), w)
}

func TestResolvePostgresValuesHappyPath(t *testing.T) {
	ctx := postgresContext(t, httpfixtures.Fixtures{
		{Method: "GET", Resource: endpointsResource, Response: readWriteEndpoint},
		{Method: "GET", Resource: databasesResource, Response: `{"databases":[{"name":"projects/p1/branches/main/databases/mydb","status":{"postgres_database":"my_pg_db"}}]}`},
	})

	result, err := prompt.ResolvePostgresValues(ctx, newPostgresResource(), testBranch, testBranch+"/databases/mydb", "")
	require.NoError(t, err)

	assert.Equal(t, map[string]string{
		"postgres.host":         "my-host.example.com",
		"postgres.databaseName": "my_pg_db",
		"postgres.endpointPath": "projects/p1/branches/main/endpoints/ep1",
	}, result)
}

func TestResolvePostgresValuesNoReadWriteEndpoint(t *testing.T) {
	ctx := postgresContext(t, httpfixtures.Fixtures{
		{Method: "GET", Resource: endpointsResource, Response: `{"endpoints":[{"name":"projects/p1/branches/main/endpoints/ep1","status":{"endpoint_type":"ENDPOINT_TYPE_READ_ONLY"}}]}`},
		{Method: "GET", Resource: databasesResource, Response: `{"databases":[{"name":"projects/p1/branches/main/databases/mydb","status":{"postgres_database":"my_pg_db"}}]}`},
	})

	result, err := prompt.ResolvePostgresValues(ctx, newPostgresResource(), testBranch, testBranch+"/databases/mydb", "")
	require.NoError(t, err)

	// host and endpointPath should be empty since no ReadWrite endpoint found.
	assert.Equal(t, map[string]string{
		"postgres.host":         "",
		"postgres.databaseName": "my_pg_db",
		"postgres.endpointPath": "",
	}, result)
}

func TestResolvePostgresValuesDatabaseNotFound(t *testing.T) {
	ctx := postgresContext(t, httpfixtures.Fixtures{
		{Method: "GET", Resource: endpointsResource, Response: readWriteEndpoint},
		{Method: "GET", Resource: databasesResource, Response: `{"databases":[{"name":"projects/p1/branches/main/databases/other","status":{"postgres_database":"other_db"}}]}`},
	})

	result, err := prompt.ResolvePostgresValues(ctx, newPostgresResource(), testBranch, testBranch+"/databases/nonexistent", "")
	require.NoError(t, err)

	// databaseName should be empty since no match.
	assert.Equal(t, map[string]string{
		"postgres.host":         "my-host.example.com",
		"postgres.databaseName": "",
		"postgres.endpointPath": "projects/p1/branches/main/endpoints/ep1",
	}, result)
}

func TestResolvePostgresValuesSkipsDatabaseListWhenNameProvided(t *testing.T) {
	// No databases fixture: listing databases would fail the test.
	ctx := postgresContext(t, httpfixtures.Fixtures{
		{Method: "GET", Resource: endpointsResource, Response: readWriteEndpoint},
	})

	result, err := prompt.ResolvePostgresValues(ctx, newPostgresResource(), testBranch, testBranch+"/databases/mydb", "my_pg_db")
	require.NoError(t, err)

	assert.Equal(t, map[string]string{
		"postgres.host":         "my-host.example.com",
		"postgres.databaseName": "my_pg_db",
		"postgres.endpointPath": "projects/p1/branches/main/endpoints/ep1",
	}, result)
}

func TestResolvePostgresValuesEndpointAPIError(t *testing.T) {
	ctx := postgresContext(t, httpfixtures.Fixtures{
		{Method: "GET", Resource: endpointsResource, Status: http.StatusBadRequest, Response: unavailableResponse},
	})

	_, err := prompt.ResolvePostgresValues(ctx, newPostgresResource(), testBranch, "some-db", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolving endpoint details")
}

func TestResolvePostgresValuesDatabaseAPIError(t *testing.T) {
	ctx := postgresContext(t, httpfixtures.Fixtures{
		{Method: "GET", Resource: endpointsResource, Response: readWriteEndpoint},
		{Method: "GET", Resource: databasesResource, Status: http.StatusBadRequest, Response: unavailableResponse},
	})

	_, err := prompt.ResolvePostgresValues(ctx, newPostgresResource(), testBranch, "some-db", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolving database name")
}

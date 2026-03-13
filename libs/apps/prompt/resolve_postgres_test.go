package prompt

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/libs/apps/manifest"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func TestResolvePostgresValuesHappyPath(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)

	branchName := "projects/p1/branches/main"
	dbName := "projects/p1/branches/main/databases/mydb"

	// Mock ListEndpoints
	endpoints := listing.SliceIterator[postgres.Endpoint]{
		{
			Name: "projects/p1/branches/main/endpoints/ep1",
			Status: &postgres.EndpointStatus{
				EndpointType: postgres.EndpointTypeEndpointTypeReadWrite,
				Hosts:        &postgres.EndpointHosts{Host: "my-host.example.com"},
			},
		},
	}
	m.GetMockPostgresAPI().EXPECT().
		ListEndpoints(mock.Anything, postgres.ListEndpointsRequest{Parent: branchName}).
		Return(&endpoints).Once()

	// Mock ListDatabases
	databases := listing.SliceIterator[postgres.Database]{
		{
			Name:   dbName,
			Status: &postgres.DatabaseDatabaseStatus{PostgresDatabase: "my_pg_db"},
		},
	}
	m.GetMockPostgresAPI().EXPECT().
		ListDatabases(mock.Anything, postgres.ListDatabasesRequest{Parent: branchName}).
		Return(&databases).Once()

	r := newPostgresResource()
	result, err := ResolvePostgresValues(ctx, r, branchName, dbName, "")
	require.NoError(t, err)

	assert.Equal(t, map[string]string{
		"postgres.host":         "my-host.example.com",
		"postgres.databaseName": "my_pg_db",
		"postgres.endpointPath": "projects/p1/branches/main/endpoints/ep1",
	}, result)
}

func TestResolvePostgresValuesNoReadWriteEndpoint(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)

	branchName := "projects/p1/branches/main"
	dbName := "projects/p1/branches/main/databases/mydb"

	// Return only a read-only endpoint.
	endpoints := listing.SliceIterator[postgres.Endpoint]{
		{
			Name: "projects/p1/branches/main/endpoints/ep1",
			Status: &postgres.EndpointStatus{
				EndpointType: postgres.EndpointTypeEndpointTypeReadOnly,
			},
		},
	}
	m.GetMockPostgresAPI().EXPECT().
		ListEndpoints(mock.Anything, postgres.ListEndpointsRequest{Parent: branchName}).
		Return(&endpoints).Once()

	databases := listing.SliceIterator[postgres.Database]{
		{
			Name:   dbName,
			Status: &postgres.DatabaseDatabaseStatus{PostgresDatabase: "my_pg_db"},
		},
	}
	m.GetMockPostgresAPI().EXPECT().
		ListDatabases(mock.Anything, postgres.ListDatabasesRequest{Parent: branchName}).
		Return(&databases).Once()

	r := newPostgresResource()
	result, err := ResolvePostgresValues(ctx, r, branchName, dbName, "")
	require.NoError(t, err)

	// host and endpointPath should be empty since no ReadWrite endpoint found.
	assert.Equal(t, map[string]string{
		"postgres.host":         "",
		"postgres.databaseName": "my_pg_db",
		"postgres.endpointPath": "",
	}, result)
}

func TestResolvePostgresValuesDatabaseNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)

	branchName := "projects/p1/branches/main"
	dbName := "projects/p1/branches/main/databases/nonexistent"

	endpoints := listing.SliceIterator[postgres.Endpoint]{
		{
			Name: "projects/p1/branches/main/endpoints/ep1",
			Status: &postgres.EndpointStatus{
				EndpointType: postgres.EndpointTypeEndpointTypeReadWrite,
				Hosts:        &postgres.EndpointHosts{Host: "my-host.example.com"},
			},
		},
	}
	m.GetMockPostgresAPI().EXPECT().
		ListEndpoints(mock.Anything, postgres.ListEndpointsRequest{Parent: branchName}).
		Return(&endpoints).Once()

	// Return databases that don't match dbName.
	databases := listing.SliceIterator[postgres.Database]{
		{
			Name:   "projects/p1/branches/main/databases/other",
			Status: &postgres.DatabaseDatabaseStatus{PostgresDatabase: "other_db"},
		},
	}
	m.GetMockPostgresAPI().EXPECT().
		ListDatabases(mock.Anything, postgres.ListDatabasesRequest{Parent: branchName}).
		Return(&databases).Once()

	r := newPostgresResource()
	result, err := ResolvePostgresValues(ctx, r, branchName, dbName, "")
	require.NoError(t, err)

	// databaseName should be empty since no match.
	assert.Equal(t, map[string]string{
		"postgres.host":         "my-host.example.com",
		"postgres.databaseName": "",
		"postgres.endpointPath": "projects/p1/branches/main/endpoints/ep1",
	}, result)
}

func TestResolvePostgresValuesSkipsDatabaseListWhenNameProvided(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)

	branchName := "projects/p1/branches/main"
	dbName := "projects/p1/branches/main/databases/mydb"

	endpoints := listing.SliceIterator[postgres.Endpoint]{
		{
			Name: "projects/p1/branches/main/endpoints/ep1",
			Status: &postgres.EndpointStatus{
				EndpointType: postgres.EndpointTypeEndpointTypeReadWrite,
				Hosts:        &postgres.EndpointHosts{Host: "my-host.example.com"},
			},
		},
	}
	m.GetMockPostgresAPI().EXPECT().
		ListEndpoints(mock.Anything, postgres.ListEndpointsRequest{Parent: branchName}).
		Return(&endpoints).Once()

	// ListDatabases should NOT be called since pgDatabaseName is pre-provided.

	r := newPostgresResource()
	result, err := ResolvePostgresValues(ctx, r, branchName, dbName, "my_pg_db")
	require.NoError(t, err)

	assert.Equal(t, map[string]string{
		"postgres.host":         "my-host.example.com",
		"postgres.databaseName": "my_pg_db",
		"postgres.endpointPath": "projects/p1/branches/main/endpoints/ep1",
	}, result)
}

func TestResolvePostgresValuesEndpointAPIError(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)

	branchName := "projects/p1/branches/main"

	// Return an iterator that yields an error.
	m.GetMockPostgresAPI().EXPECT().
		ListEndpoints(mock.Anything, postgres.ListEndpointsRequest{Parent: branchName}).
		RunAndReturn(func(_ context.Context, _ postgres.ListEndpointsRequest) listing.Iterator[postgres.Endpoint] {
			return &errorIterator[postgres.Endpoint]{err: errors.New("API unavailable")}
		}).Once()

	r := newPostgresResource()
	_, err := ResolvePostgresValues(ctx, r, branchName, "some-db", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolving endpoint details")
}

// errorIterator is a test helper that always returns an error.
type errorIterator[T any] struct {
	err error
}

func (e *errorIterator[T]) HasNext(_ context.Context) bool { return true }

func (e *errorIterator[T]) Next(_ context.Context) (T, error) {
	var zero T
	return zero, e.err
}

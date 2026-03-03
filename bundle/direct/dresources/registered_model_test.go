package dresources

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRegisteredModelTest(t *testing.T) *ResourceRegisteredModel {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	return (&ResourceRegisteredModel{}).New(client)
}

func TestSyncAliases_AddsNewAliases(t *testing.T) {
	r := setupRegisteredModelTest(t)
	ctx := context.Background()

	config := &catalog.CreateRegisteredModelRequest{
		Name: "my_model", CatalogName: "main", SchemaName: "default",
		Aliases: []catalog.RegisteredModelAlias{
			{AliasName: "champion", VersionNum: 1},
			{AliasName: "staging", VersionNum: 2},
		},
	}
	id, _, err := r.DoCreate(ctx, config)
	require.NoError(t, err)

	_, err = r.WaitAfterCreate(ctx, config)
	require.NoError(t, err)

	remote, err := r.DoRead(ctx, id)
	require.NoError(t, err)
	assert.Len(t, remote.Aliases, 2)

	m := aliasMap(remote.Aliases)
	assert.Equal(t, 1, m["champion"])
	assert.Equal(t, 2, m["staging"])
}

func TestSyncAliases_UpdatesAndDeletesAliases(t *testing.T) {
	r := setupRegisteredModelTest(t)
	ctx := context.Background()

	id, _, err := r.DoCreate(ctx, &catalog.CreateRegisteredModelRequest{
		Name: "my_model", CatalogName: "main", SchemaName: "default",
		Aliases: []catalog.RegisteredModelAlias{
			{AliasName: "champion", VersionNum: 1},
			{AliasName: "staging", VersionNum: 2},
		},
	})
	require.NoError(t, err)

	// Modify champion version, remove staging, add latest.
	err = r.syncAliases(ctx, id, []catalog.RegisteredModelAlias{
		{AliasName: "champion", VersionNum: 5},
		{AliasName: "latest", VersionNum: 3},
	}, nil)
	require.NoError(t, err)

	remote, err := r.DoRead(ctx, id)
	require.NoError(t, err)
	assert.Len(t, remote.Aliases, 2)

	m := aliasMap(remote.Aliases)
	assert.Equal(t, 5, m["champion"])
	assert.Equal(t, 3, m["latest"])
	_, hasStaging := m["staging"]
	assert.False(t, hasStaging)
}

func TestSyncAliases_RemovesAllAliases(t *testing.T) {
	r := setupRegisteredModelTest(t)
	ctx := context.Background()

	id, _, err := r.DoCreate(ctx, &catalog.CreateRegisteredModelRequest{
		Name: "my_model", CatalogName: "main", SchemaName: "default",
		Aliases: []catalog.RegisteredModelAlias{
			{AliasName: "champion", VersionNum: 1},
		},
	})
	require.NoError(t, err)

	err = r.syncAliases(ctx, id, nil, nil)
	require.NoError(t, err)

	remote, err := r.DoRead(ctx, id)
	require.NoError(t, err)
	assert.Empty(t, remote.Aliases)
}

func aliasMap(aliases []catalog.RegisteredModelAlias) map[string]int {
	m := make(map[string]int, len(aliases))
	for _, a := range aliases {
		m[a.AliasName] = a.VersionNum
	}
	return m
}

package dresources

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
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

func createRegisteredModelWithAliases(t *testing.T, r *ResourceRegisteredModel, aliases []catalog.RegisteredModelAlias) (string, *catalog.CreateRegisteredModelRequest) {
	ctx := t.Context()
	config := &catalog.CreateRegisteredModelRequest{
		Name: "my_model", CatalogName: "main", SchemaName: "default",
		Aliases: aliases,
	}
	id, _, err := r.DoCreate(ctx, config)
	require.NoError(t, err)

	_, err = r.WaitAfterCreate(ctx, id, config)
	require.NoError(t, err)

	return id, config
}

func TestRegisteredModelWaitAfterCreateSyncsAliases(t *testing.T) {
	r := setupRegisteredModelTest(t)

	id, _ := createRegisteredModelWithAliases(t, r, []catalog.RegisteredModelAlias{
		{AliasName: "champion", VersionNum: 1},
		{AliasName: "staging", VersionNum: 2},
	})

	remote, err := r.DoRead(t.Context(), id)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"champion": 1, "staging": 2}, aliasMap(remote.Aliases))
}

func TestRegisteredModelDoUpdateSyncsChangedAliases(t *testing.T) {
	r := setupRegisteredModelTest(t)
	ctx := t.Context()

	id, config := createRegisteredModelWithAliases(t, r, []catalog.RegisteredModelAlias{
		{AliasName: "champion", VersionNum: 1},
		{AliasName: "staging", VersionNum: 2},
	})

	remoteBefore, err := r.DoRead(ctx, id)
	require.NoError(t, err)

	// Modify champion version, remove staging, add latest.
	config.Aliases = []catalog.RegisteredModelAlias{
		{AliasName: "champion", VersionNum: 5},
		{AliasName: "latest", VersionNum: 3},
	}
	_, err = r.DoUpdate(ctx, id, config, &PlanEntry{
		Changes:     Changes{"aliases": &ChangeDesc{Action: deployplan.Update}},
		RemoteState: remoteBefore,
	})
	require.NoError(t, err)

	remote, err := r.DoRead(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"champion": 5, "latest": 3}, aliasMap(remote.Aliases))
}

func TestRegisteredModelDoUpdateRemovesAllAliases(t *testing.T) {
	r := setupRegisteredModelTest(t)
	ctx := t.Context()

	id, config := createRegisteredModelWithAliases(t, r, []catalog.RegisteredModelAlias{
		{AliasName: "champion", VersionNum: 1},
	})

	remoteBefore, err := r.DoRead(ctx, id)
	require.NoError(t, err)

	config.Aliases = nil
	_, err = r.DoUpdate(ctx, id, config, &PlanEntry{
		Changes:     Changes{"aliases": &ChangeDesc{Action: deployplan.Update}},
		RemoteState: remoteBefore,
	})
	require.NoError(t, err)

	remote, err := r.DoRead(ctx, id)
	require.NoError(t, err)
	assert.Empty(t, remote.Aliases)
}

func TestRegisteredModelDoUpdateSkipsAliasSyncWhenUnchanged(t *testing.T) {
	r := setupRegisteredModelTest(t)
	ctx := t.Context()

	id, config := createRegisteredModelWithAliases(t, r, []catalog.RegisteredModelAlias{
		{AliasName: "champion", VersionNum: 1},
	})

	// No "aliases" entry in Changes: DoUpdate must not touch aliases even
	// though the config differs from remote.
	config.Aliases = nil
	config.Comment = "updated"
	_, err := r.DoUpdate(ctx, id, config, &PlanEntry{
		Changes: Changes{"comment": &ChangeDesc{Action: deployplan.Update}},
	})
	require.NoError(t, err)

	remote, err := r.DoRead(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"champion": 1}, aliasMap(remote.Aliases))
}

func aliasMap(aliases []catalog.RegisteredModelAlias) map[string]int {
	m := make(map[string]int, len(aliases))
	for _, a := range aliases {
		m[a.AliasName] = a.VersionNum
	}
	return m
}

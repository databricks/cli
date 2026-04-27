package phases_test

import (
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"testing"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/cli/ucm/phases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindTerraformEngineRunsImportAndPushes(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Resources.Catalogs = map[string]*resources.Catalog{
		"main": {CreateCatalog: catalog.CreateCatalog{Name: "main"}},
	}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Bind(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	}, phases.BindRequest{Kind: phases.ImportCatalog, Name: "main", Key: "main"})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 1, f.tf.ImportCalls)
	assert.Equal(t, "databricks_catalog.main", f.tf.LastImportAddress)
	assert.Equal(t, "main", f.tf.LastImportId)
	assert.Equal(t, 1, readRemoteSeq(t, f), "successful bind must push remote state")
}

func TestBindRequiresDeclaredResource(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Bind(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	}, phases.BindRequest{Kind: phases.ImportCatalog, Name: "ghost", Key: "ghost"})

	require.True(t, logdiag.HasError(ctx))
	assert.Equal(t, 0, f.tf.ImportCalls)
}

func TestBindDirectEngineSkipsTerraform(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Ucm.Engine = engine.EngineDirect
	f.u.Config.Resources.Catalogs = map[string]*resources.Catalog{
		"main": {CreateCatalog: catalog.CreateCatalog{Name: "main"}},
	}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Bind(ctx, f.u, phases.Options{
		TerraformFactory:    fakeTfFactory(f.tf),
		DirectClientFactory: fakeDirectClientFactory(),
	}, phases.BindRequest{Kind: phases.ImportCatalog, Name: "main", Key: "main"})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 0, f.tf.ImportCalls, "direct engine must not invoke the terraform wrapper")
	assert.Equal(t, -1, readRemoteSeq(t, f), "direct engine must never push remote state")

	// Verify the direct state was actually written with the expected entry —
	// skipping terraform and writing nothing would also satisfy the two
	// assertions above, so the positive assertion is load-bearing.
	state, err := direct.LoadState(direct.StatePath(f.u))
	require.NoError(t, err)
	got := state.Catalogs["main"]
	require.NotNil(t, got, "direct bind must record catalogs[main]")
	assert.Equal(t, "main", got.Name)
}

func TestUnbindTerraformEngineRunsStateRmAndPushes(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Unbind(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	}, phases.UnbindRequest{Kind: phases.ImportCatalog, Key: "main"})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 1, f.tf.StateRmCalls)
	assert.Equal(t, "databricks_catalog.main", f.tf.LastStateRmAddress)
	assert.Equal(t, 1, readRemoteSeq(t, f), "successful unbind must push remote state")
}

func TestUnbindDirectEngineSkipsTerraform(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Ucm.Engine = engine.EngineDirect
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	// Seed direct state with the catalog so Unbind has something to remove.
	f.u.Config.Resources.Catalogs = map[string]*resources.Catalog{
		"main": {CreateCatalog: catalog.CreateCatalog{Name: "main"}},
	}
	phases.Bind(ctx, f.u, phases.Options{
		TerraformFactory:    fakeTfFactory(f.tf),
		DirectClientFactory: fakeDirectClientFactory(),
	}, phases.BindRequest{Kind: phases.ImportCatalog, Name: "main", Key: "main"})
	require.False(t, logdiag.HasError(ctx), "seed bind failed: %v", logdiag.FlushCollected(ctx))

	phases.Unbind(ctx, f.u, phases.Options{
		TerraformFactory:    fakeTfFactory(f.tf),
		DirectClientFactory: fakeDirectClientFactory(),
	}, phases.UnbindRequest{Kind: phases.ImportCatalog, Key: "main"})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 0, f.tf.StateRmCalls, "direct engine must not invoke the terraform wrapper")
	assert.Equal(t, -1, readRemoteSeq(t, f), "direct engine must never push remote state")
}

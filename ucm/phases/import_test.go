package phases_test

import (
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"testing"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/cli/ucm/phases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportTerraformEngineRunsImportAndPushes(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Resources.Catalogs = map[string]*resources.Catalog{
		"main": {CreateCatalog: catalog.CreateCatalog{Name: "main"}},
	}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Import(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	}, phases.ImportRequest{Kind: phases.ImportCatalog, Name: "main", Key: "main"})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 1, f.tf.ImportCalls)
	assert.Equal(t, "databricks_catalog.main", f.tf.LastImportAddress)
	assert.Equal(t, "main", f.tf.LastImportId)
	assert.Equal(t, 1, readRemoteSeq(t, f), "successful import must push remote state")
}

func TestImportRequiresDeclaredResource(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Import(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	}, phases.ImportRequest{Kind: phases.ImportCatalog, Name: "ghost", Key: "ghost"})

	require.True(t, logdiag.HasError(ctx))
	assert.Equal(t, 0, f.tf.ImportCalls)
}

func TestImportDirectEngineSkipsTerraform(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Ucm.Engine = engine.EngineDirect
	f.u.Config.Resources.Catalogs = map[string]*resources.Catalog{
		"main": {CreateCatalog: catalog.CreateCatalog{Name: "main"}},
	}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Import(ctx, f.u, phases.Options{
		TerraformFactory:    fakeTfFactory(f.tf),
		DirectClientFactory: fakeDirectClientFactory(),
	}, phases.ImportRequest{Kind: phases.ImportCatalog, Name: "main", Key: "main"})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 0, f.tf.ImportCalls, "direct engine must not invoke the terraform wrapper")
	assert.Equal(t, -1, readRemoteSeq(t, f), "direct engine must never push remote state")
}

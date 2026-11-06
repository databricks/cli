package generate

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testCatalogSpec = newResourceSpec(resource[*catalog.CatalogInfo, resources.Catalog]{
	LookupFlag:  "existing-catalog-name",
	LookupUsage: "name of the catalog",
	Fetch: func(ctx context.Context, w *databricks.WorkspaceClient, name string) (*catalog.CatalogInfo, error) {
		if name == "missing" {
			return nil, errors.New("catalog not found")
		}
		return &catalog.CatalogInfo{
			Name:        name,
			Comment:     "primary",
			Owner:       "someone@example.com",
			CatalogType: catalog.CatalogTypeManagedCatalog,
			StorageRoot: "s3://bucket",
		}, nil
	},
	SkipFields: []string{"storage_root"},
})

func testBundle(t *testing.T) (context.Context, *bundle.Bundle) {
	t.Helper()
	b := &bundle.Bundle{BundleRootPath: t.TempDir()}
	b.SetWorkpaceClient(mocks.NewMockWorkspaceClient(t).WorkspaceClient)
	ctx := logdiag.InitContext(cmdio.MockDiscard(t.Context()))
	logdiag.SetCollect(ctx, true)
	return ctx, b
}

func TestGenerateResourceWritesConfig(t *testing.T) {
	ctx, b := testBundle(t)

	key, err := generateResource(ctx, b, testCatalogSpec, resourceOptions{lookup: "My Catalog", configDir: "resources"})
	require.NoError(t, err)
	assert.Equal(t, "my_catalog", key)

	data, err := os.ReadFile(filepath.Join(b.BundleRootPath, "resources", "my_catalog.catalog.yml"))
	require.NoError(t, err)
	assert.Equal(t, `resources:
  catalogs:
    my_catalog:
      comment: primary
      name: My Catalog
`, string(data))

	// The bundle has no include patterns, so the file is reported as not included.
	diags := logdiag.FlushCollected(ctx)
	require.Len(t, diags, 1)
	assert.Equal(t, "Generated configuration is not included in the bundle", diags[0].Summary)
}

func TestGenerateResourceHonorsKeyAndConfigDir(t *testing.T) {
	ctx, b := testBundle(t)

	key, err := generateResource(ctx, b, testCatalogSpec, resourceOptions{lookup: "main", key: "k", configDir: "conf"})
	require.NoError(t, err)
	assert.Equal(t, "k", key)
	assert.FileExists(t, filepath.Join(b.BundleRootPath, "conf", "k.catalog.yml"))
}

func TestGenerateResourceReturnsFetchError(t *testing.T) {
	ctx, b := testBundle(t)

	_, err := generateResource(ctx, b, testCatalogSpec, resourceOptions{lookup: "missing", configDir: "resources"})
	assert.EqualError(t, err, "catalog not found")
}

func TestGenerateResourceRefusesToOverwriteWithoutForce(t *testing.T) {
	ctx, b := testBundle(t)
	existing := filepath.Join(b.BundleRootPath, "resources", "main.catalog.yml")
	require.NoError(t, os.MkdirAll(filepath.Dir(existing), 0o755))
	require.NoError(t, os.WriteFile(existing, []byte("keep"), 0o644))

	opts := resourceOptions{lookup: "main", configDir: "resources"}
	_, err := generateResource(ctx, b, testCatalogSpec, opts)
	assert.ErrorContains(t, err, "already exists")

	opts.force = true
	_, err = generateResource(ctx, b, testCatalogSpec, opts)
	require.NoError(t, err)
	data, err := os.ReadFile(existing)
	require.NoError(t, err)
	assert.Contains(t, string(data), "name: main")
}

func TestResourceCommandFlagsAndHelp(t *testing.T) {
	cmd := newResourceCommand(testCatalogSpec)
	assert.Equal(t, "catalog", cmd.Use)
	assert.Equal(t, "Generate bundle configuration for a Catalog", cmd.Short)
	assert.Contains(t, cmd.Long, "--existing-catalog-name NAME --key my_catalog")
	assert.True(t, cmd.Flags().Lookup("bind").Hidden)

	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(nil)
	err := cmd.Execute()
	assert.ErrorContains(t, err, `required flag(s) "existing-catalog-name" not set`)
}

func TestArticle(t *testing.T) {
	assert.Equal(t, "an", article("Experiment"))
	assert.Equal(t, "a", article("Catalog"))
}

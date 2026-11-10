package generate

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testBundle(t *testing.T) (context.Context, *bundle.Bundle, *mocks.MockWorkspaceClient) {
	t.Helper()
	m := mocks.NewMockWorkspaceClient(t)
	b := &bundle.Bundle{BundleRootPath: t.TempDir()}
	b.SetWorkpaceClient(m.WorkspaceClient)
	ctx := logdiag.InitContext(cmdio.MockDiscard(t.Context()))
	logdiag.SetCollect(ctx, true)
	return ctx, b, m
}

func generateCatalog(t *testing.T, opts resourceOptions) (context.Context, *bundle.Bundle, string, error) {
	t.Helper()
	ctx, b, m := testBundle(t)
	m.GetMockCatalogsAPI().EXPECT().GetByName(mock.Anything, opts.id).Return(&catalog.CatalogInfo{
		Name:        opts.id,
		Comment:     "primary",
		Owner:       "someone@example.com",
		CatalogType: catalog.CatalogTypeManagedCatalog,
		StorageRoot: "s3://bucket",
	}, nil).Maybe()
	key, err := generateResource(ctx, b, "catalogs", config.SupportedResources()["catalogs"], opts)
	return ctx, b, key, err
}

func TestGenerateResourceWritesConfig(t *testing.T) {
	ctx, b, key, err := generateCatalog(t, resourceOptions{id: "My Catalog", configDir: "resources"})
	require.NoError(t, err)
	assert.Equal(t, "my_catalog", key)

	data, err := os.ReadFile(filepath.Join(b.BundleRootPath, "resources", "my_catalog.catalog.yml"))
	require.NoError(t, err)
	// Owner and catalog type are not part of the create request and are dropped.
	assert.Equal(t, `resources:
  catalogs:
    my_catalog:
      comment: primary
      name: My Catalog
      storage_root: s3://bucket
`, string(data))

	// The bundle has no include patterns, so the file is reported as not included.
	diags := logdiag.FlushCollected(ctx)
	require.Len(t, diags, 1)
	assert.Equal(t, "Generated configuration is not included in the bundle", diags[0].Summary)
}

func TestGenerateResourceHonorsKeyAndConfigDir(t *testing.T) {
	_, b, key, err := generateCatalog(t, resourceOptions{id: "main", key: "k", configDir: "conf"})
	require.NoError(t, err)
	assert.Equal(t, "k", key)
	assert.FileExists(t, filepath.Join(b.BundleRootPath, "conf", "k.catalog.yml"))
}

func TestGenerateResourceRefusesToOverwriteWithoutForce(t *testing.T) {
	ctx, b, m := testBundle(t)
	m.GetMockCatalogsAPI().EXPECT().GetByName(mock.Anything, "main").Return(&catalog.CatalogInfo{Name: "main"}, nil)
	existing := filepath.Join(b.BundleRootPath, "resources", "main.catalog.yml")
	require.NoError(t, os.MkdirAll(filepath.Dir(existing), 0o755))
	require.NoError(t, os.WriteFile(existing, []byte("keep"), 0o644))
	desc := config.SupportedResources()["catalogs"]

	opts := resourceOptions{id: "main", configDir: "resources"}
	_, err := generateResource(ctx, b, "catalogs", desc, opts)
	assert.ErrorContains(t, err, "already exists")

	opts.force = true
	_, err = generateResource(ctx, b, "catalogs", desc, opts)
	require.NoError(t, err)
	data, err := os.ReadFile(existing)
	require.NoError(t, err)
	assert.Contains(t, string(data), "name: main")
}

func TestGenerateResourceReturnsReadError(t *testing.T) {
	ctx, b, m := testBundle(t)
	m.GetMockCatalogsAPI().EXPECT().GetByName(mock.Anything, "missing").Return(nil, errors.New("catalog not found"))

	_, err := generateResource(ctx, b, "catalogs", config.SupportedResources()["catalogs"], resourceOptions{id: "missing", configDir: "resources"})
	assert.EqualError(t, err, "catalog not found")
}

func TestGenerateResourceUnwrapsExperimentResponse(t *testing.T) {
	ctx, b, m := testBundle(t)
	m.GetMockExperimentsAPI().EXPECT().GetExperiment(mock.Anything, ml.GetExperimentRequest{ExperimentId: "123"}).Return(&ml.GetExperimentResponse{
		Experiment: &ml.Experiment{
			ExperimentId:     "123",
			Name:             "/Users/someone/My Experiment",
			ArtifactLocation: "dbfs:/artifacts",
			LifecycleStage:   "active",
		},
	}, nil)

	key, err := generateResource(ctx, b, "experiments", config.SupportedResources()["experiments"], resourceOptions{id: "123", configDir: "resources"})
	require.NoError(t, err)
	assert.Equal(t, "users_someone_my_experiment", key)

	data, err := os.ReadFile(filepath.Join(b.BundleRootPath, "resources", "users_someone_my_experiment.experiment.yml"))
	require.NoError(t, err)
	assert.Equal(t, `resources:
  experiments:
    users_someone_my_experiment:
      artifact_location: dbfs:/artifacts
      name: /Users/someone/My Experiment
`, string(data))
}

func TestGenerateResourceRejectsStateThatDoesNotMapOntoConfig(t *testing.T) {
	ctx, b, m := testBundle(t)
	// Secret scope state uses "scope" where the bundle resource uses "name".
	m.GetMockSecretsAPI().EXPECT().ListScopesAll(mock.Anything).Return([]workspace.SecretScope{
		{Name: "my_scope", BackendType: workspace.ScopeBackendTypeDatabricks},
	}, nil)

	_, err := generateResource(ctx, b, "secret_scopes", config.SupportedResources()["secret_scopes"], resourceOptions{id: "my_scope", configDir: "resources"})
	assert.ErrorContains(t, err, "generating secret_scopes is not supported yet")
	assert.NoFileExists(t, filepath.Join(b.BundleRootPath, "resources", "my_scope.secret_scope.yml"))
}

func TestNewCommandsCoversEveryResourceType(t *testing.T) {
	names := make(map[string]*cobra.Command)
	for _, cmd := range NewCommands() {
		names[cmd.Name()] = cmd
	}

	// Hand-written generators keep their own command.
	assert.Contains(t, names["job"].Flags().FlagUsages(), "existing-job-id")
	// Generic ones take --existing-id.
	assert.Contains(t, names["catalog"].Flags().FlagUsages(), "existing-id")
	assert.Contains(t, names, "external-location")

	assert.NotContains(t, names, "job-run")
	for name := range names {
		assert.NotContains(t, name, ".")
	}
}

func TestResourceCommandFlagsAndHelp(t *testing.T) {
	cmd := newResourceCommand("external_locations", config.SupportedResources()["external_locations"])
	assert.Equal(t, "external-location", cmd.Use)
	assert.Equal(t, "Generate bundle configuration for an external location", cmd.Short)
	assert.Contains(t, cmd.Long, "generate external-location --existing-id ID --key my_external_location")
	assert.True(t, cmd.Flags().Lookup("bind").Hidden)

	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(nil)
	err := cmd.Execute()
	assert.ErrorContains(t, err, `required flag(s) "existing-id" not set`)
}

func TestArticle(t *testing.T) {
	assert.Equal(t, "an", article("experiment"))
	assert.Equal(t, "a", article("catalog"))
}

package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewGenerateCatalogCommand_Help(t *testing.T) {
	cmd := NewGenerateCatalogCommand()
	assert.Contains(t, cmd.Long, "existing Unity Catalog catalog")
	assert.NotNil(t, cmd.Flag("existing-catalog-name"))
	assert.NotNil(t, cmd.Flag("output-dir"))
	assert.NotNil(t, cmd.Flag("force"))
}

func TestGenerateCatalog_WritesYAML(t *testing.T) {
	work := t.TempDir()
	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockCatalogsAPI().EXPECT().
		GetByName(mock.Anything, "team_alpha").
		Return(&catalog.CatalogInfo{
			Name:       "team_alpha",
			Comment:    "alpha team",
			Properties: map[string]string{"owner": "alpha"},
		}, nil)

	stderr, err := runSubcmd(t, w,
		"catalog",
		"--existing-catalog-name", "team_alpha",
		"--output-dir", work,
	)
	require.NoError(t, err, "stderr=%s", stderr)

	out := filepath.Join(work, "catalogs_team_alpha.yml")
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	contents := string(data)
	assert.Contains(t, contents, "team_alpha")
	assert.Contains(t, contents, "alpha team")
	assert.Contains(t, contents, "catalogs:")
	assert.Contains(t, stderr, "Wrote catalog")
}

func TestGenerateCatalog_KeyOverride(t *testing.T) {
	work := t.TempDir()
	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockCatalogsAPI().EXPECT().
		GetByName(mock.Anything, "raw").
		Return(&catalog.CatalogInfo{Name: "raw"}, nil)

	_, err := runSubcmd(t, w,
		"--key", "raw_alias",
		"catalog",
		"--existing-catalog-name", "raw",
		"--output-dir", work,
	)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(work, "catalogs_raw_alias.yml"))
	require.NoError(t, err)
}

func TestGenerateCatalog_RefusesOverwriteWithoutForce(t *testing.T) {
	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, "catalogs_x.yml"), []byte("old"), 0o644))

	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockCatalogsAPI().EXPECT().
		GetByName(mock.Anything, "x").
		Return(&catalog.CatalogInfo{Name: "x"}, nil)

	_, err := runSubcmd(t, w,
		"catalog",
		"--existing-catalog-name", "x",
		"--output-dir", work,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestGenerateCatalog_CreatesMissingOutputDir(t *testing.T) {
	work := t.TempDir()
	nested := filepath.Join(work, "does", "not", "exist")

	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockCatalogsAPI().EXPECT().
		GetByName(mock.Anything, "team_alpha").
		Return(&catalog.CatalogInfo{Name: "team_alpha"}, nil)

	_, err := runSubcmd(t, w,
		"catalog",
		"--existing-catalog-name", "team_alpha",
		"--output-dir", nested,
	)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(nested, "catalogs_team_alpha.yml"))
	require.NoError(t, err)
}

func TestGenerateCatalog_ForceOverwrites(t *testing.T) {
	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, "catalogs_x.yml"), []byte("old"), 0o644))

	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockCatalogsAPI().EXPECT().
		GetByName(mock.Anything, "x").
		Return(&catalog.CatalogInfo{Name: "x", Comment: "fresh"}, nil)

	_, err := runSubcmd(t, w,
		"catalog",
		"--existing-catalog-name", "x",
		"--output-dir", work,
		"--force",
	)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(work, "catalogs_x.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "fresh")
	assert.NotContains(t, string(data), "old")
}

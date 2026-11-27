package skills

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWithoutDir(t *testing.T) {
	src, err := Resolve(env.Set(t.Context(), "DATABRICKS_SKILLS_DIR", ""))
	assert.ErrorIs(t, err, ErrNoSource)
	assert.Nil(t, src)
}

// The folder is built with the host's separator, so on Windows the label is
// converted from a backslash path. On other hosts it is the same path.
func TestResolveWithDir(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_SKILLS_DIR", filepath.Join("testdata", "release"))
	src, err := Resolve(ctx)
	require.NoError(t, err)

	catalog, err := src.Catalog(ctx)
	require.NoError(t, err)
	assert.Equal(t, "1.2.0", catalog.Version)
	assert.Equal(t, "testdata/release", catalog.Label)
}

func TestLocalCatalogWithoutDir(t *testing.T) {
	catalog, err := LocalCatalog(env.Set(t.Context(), "DATABRICKS_SKILLS_DIR", ""))
	assert.NoError(t, err)
	assert.Nil(t, catalog)
}

// The acceptance fixture is the release that later acceptance tests point
// DATABRICKS_SKILLS_DIR at, so this checks that it stays a valid release.
func TestLocalCatalogFromAcceptanceFixture(t *testing.T) {
	dir := filepath.Join("..", "..", "acceptance", "cmd", "skills", "release")
	catalog, err := LocalCatalog(env.Set(t.Context(), "DATABRICKS_SKILLS_DIR", dir))
	require.NoError(t, err)
	require.Len(t, catalog.Skills, 3)
	assert.Equal(t, "demo-skill", catalog.Skills[0].Name)
	assert.Equal(t, "future-skill", catalog.Skills[1].Name)
	assert.Equal(t, "guide-skill", catalog.Skills[2].Name)
}

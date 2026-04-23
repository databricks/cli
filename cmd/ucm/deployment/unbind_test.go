package deployment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnbindResourceDirect_RemovesStateAndLeavesConfigUntouched(t *testing.T) {
	u := setupUcmFixture(t)
	before, err := os.ReadFile(filepath.Join(u.RootPath, "ucm.yml"))
	require.NoError(t, err)

	// Seed a recorded state so unbind has something to remove.
	client := newFakeDirectClient()
	client.catalogs["team_alpha"] = &catalog.CatalogInfo{Name: "team_alpha"}
	require.NoError(t, bindResourceDirect(t.Context(), u, client, kindCatalog, "my_catalog", "team_alpha"))

	require.NoError(t, unbindResourceDirect(u, kindCatalog, "my_catalog"))

	state, err := direct.LoadState(direct.StatePath(u))
	require.NoError(t, err)
	_, ok := state.Catalogs["my_catalog"]
	assert.False(t, ok, "catalog should be removed from state")

	// ucm.yml must remain untouched.
	after, err := os.ReadFile(filepath.Join(u.RootPath, "ucm.yml"))
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

func TestUnbindResourceDirect_ErrorsWhenKeyNotBound(t *testing.T) {
	u := setupUcmFixture(t)

	err := unbindResourceDirect(u, kindCatalog, "my_catalog")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no bound")
	assert.Contains(t, err.Error(), "my_catalog")
}

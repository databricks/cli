package direct_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadState_MissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resources.json")
	s, err := direct.LoadState(path)
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Empty(t, s.Catalogs)
	assert.Empty(t, s.Schemas)
	assert.Empty(t, s.Grants)
}

func TestLoadState_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resources.json")
	in := direct.NewState()
	in.Catalogs["main"] = &direct.CatalogState{Name: "main", Comment: "prod"}
	in.Schemas["raw"] = &direct.SchemaState{Name: "raw", Catalog: "main"}
	in.Grants["analysts"] = &direct.GrantState{
		SecurableType: "schema",
		SecurableName: "main.raw",
		Principal:     "analysts",
		Privileges:    []string{"SELECT"},
	}

	require.NoError(t, direct.SaveState(path, in))

	out, err := direct.LoadState(path)
	require.NoError(t, err)
	assert.Equal(t, in.Catalogs, out.Catalogs)
	assert.Equal(t, in.Schemas, out.Schemas)
	assert.Equal(t, in.Grants, out.Grants)
}

func TestLoadState_RejectsFutureVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resources.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"version":99}`), 0o644))

	_, err := direct.LoadState(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version 99")
}

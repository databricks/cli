package deployment

import (
	"testing"

	"github.com/databricks/cli/ucm/phases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveBindable(t *testing.T) {
	u := setupUcmFixture(t)

	cases := []struct {
		key     string
		want    phases.ImportKind
		wantErr string
	}{
		{"my_catalog", phases.ImportCatalog, ""},
		{"my_schema", phases.ImportSchema, ""},
		{"my_sc", phases.ImportStorageCredential, ""},
		{"my_loc", phases.ImportExternalLocation, ""},
		{"my_vol", phases.ImportVolume, ""},
		{"my_conn", phases.ImportConnection, ""},
		{"grant_a", "", "grants are not bindable"},
		{"does_not_exist", "", "no bindable resource"},
	}

	for _, c := range cases {
		got, err := resolveBindable(u, c.key)
		if c.wantErr != "" {
			require.Error(t, err, c.key)
			assert.Contains(t, err.Error(), c.wantErr, c.key)
			continue
		}
		require.NoError(t, err, c.key)
		assert.Equal(t, c.want, got, c.key)
	}
}

package direct_test

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnbindReturnsRemovedKeys(t *testing.T) {
	for _, tc := range []struct {
		name    string
		keys    []string
		history bool
	}{
		{"resource", []string{"resources.jobs.foo"}, true},
		{"children", []string{"resources.jobs.foo", "resources.jobs.foo.permissions", "resources.jobs.foo.grants"}, true},
		{"children_without_parent", []string{"resources.jobs.foo.permissions"}, true},
		{"missing", nil, true},
		{"history_disabled", []string{"resources.jobs.foo"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			statePath := filepath.Join(t.TempDir(), "resources.json")
			var b direct.DeploymentBundle
			require.NoError(t, b.StateDB.Open(ctx, statePath, dstate.ReadingFile, false, dstate.OpenDmsArgs{}))
			require.NoError(t, b.StateDB.UpgradeToWrite())
			for _, key := range append(tc.keys, "resources.jobs.foobar") {
				require.NoError(t, b.StateDB.SaveState(ctx, key, "id", map[string]string{"name": key}, nil))
			}
			_, err := b.StateDB.Finalize(ctx)
			require.NoError(t, err)

			root := &config.Root{Experimental: &config.Experimental{DeploymentHistory: &tc.history}}
			metadata, err := b.Unbind(ctx, root, statePath, "resources.jobs.foo")
			require.NoError(t, err)
			if tc.history {
				require.NotNil(t, metadata)
				assert.ElementsMatch(t, tc.keys, metadata.ResourceKeys)
			} else {
				assert.Nil(t, metadata)
			}

			require.NoError(t, b.StateDB.Open(ctx, statePath, dstate.ReadingFile, false, dstate.OpenDmsArgs{}))
			assert.Len(t, b.StateDB.Data.State, 1)
			if metadata != nil {
				assert.Equal(t, b.StateDB.Data.Serial, metadata.Serial)
			}
			assert.Equal(t, "id", b.StateDB.GetResourceID("resources.jobs.foobar"))
			_, err = b.StateDB.Finalize(ctx)
			require.NoError(t, err)
		})
	}
}

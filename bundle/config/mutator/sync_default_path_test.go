package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncDefaultPath_DefaultIfUnset(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: "/tmp/some/dir",
		Config:         config.Root{},
	}

	ctx := t.Context()
	diags := bundle.Apply(ctx, b, mutator.SyncDefaultPath())
	require.NoError(t, diags.Error())
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)
}

func TestSyncDefaultPath_SkipIfSet(t *testing.T) {
	tcases := []struct {
		name   string
		paths  any
		expect []string
	}{
		{
			name:   "nil",
			paths:  nil,
			expect: []string{"."},
		},
		{
			name:   "empty sequence",
			paths:  []string{},
			expect: []string{},
		},
		{
			name:   "non-empty sequence",
			paths:  []string{"something"},
			expect: []string{"something"},
		},
	}

	for _, tcase := range tcases {
		t.Run(tcase.name, func(t *testing.T) {
			b := &bundle.Bundle{
				BundleRootPath: "/tmp/some/dir",
				Config:         config.Root{},
			}

			ctx := logdiag.InitContext(t.Context())

			bundle.ApplyFuncContext(ctx, b, func(ctx context.Context, b *bundle.Bundle) {
				err := b.Config.Set(structpath.MustParsePath("sync.paths"), tcase.paths)
				require.NoError(t, err)
			})
			require.False(t, logdiag.HasError(ctx))

			diags := bundle.Apply(ctx, b, mutator.SyncDefaultPath())
			require.NoError(t, diags.Error())

			// If the sync paths field is already set, do nothing.
			assert.Equal(t, tcase.expect, b.Config.Sync.Paths)
		})
	}
}

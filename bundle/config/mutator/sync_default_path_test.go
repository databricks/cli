package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncDefaultPath_DefaultIfUnset(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: "/tmp/some/dir",
		Config:         config.Root{},
	}

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, mutator.SyncDefaultPath())
	require.NoError(t, diags.Error())
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)
}

func TestSyncDefaultPath_SkipIfSet(t *testing.T) {
	tcases := []struct {
		name   string
		paths  dyn.Value
		expect []string
	}{
		{
			name:   "nil",
			paths:  dyn.V(nil),
			expect: nil,
		},
		{
			name:   "empty sequence",
			paths:  dyn.V([]dyn.Value{}),
			expect: []string{},
		},
		{
			name:   "non-empty sequence",
			paths:  dyn.V([]dyn.Value{dyn.V("something")}),
			expect: []string{"something"},
		},
	}

	for _, tcase := range tcases {
		t.Run(tcase.name, func(t *testing.T) {
			b := &bundle.Bundle{
				BundleRootPath: "/tmp/some/dir",
				Config:         config.Root{},
			}

			ctx := logdiag.InitContext(context.Background())

			bundle.ApplyFuncContext(ctx, b, func(ctx context.Context, b *bundle.Bundle) {
				err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
					v, err := dyn.Set(v, "sync", dyn.V(dyn.NewMapping()))
					if err != nil {
						return dyn.InvalidValue, err
					}
					v, err = dyn.Set(v, "sync.paths", tcase.paths)
					if err != nil {
						return dyn.InvalidValue, err
					}
					return v, nil
				})
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

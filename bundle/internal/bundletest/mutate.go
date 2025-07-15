package bundletest

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/require"
)

func Mutate(t *testing.T, b *bundle.Bundle, f func(v dyn.Value) (dyn.Value, error)) {
	bundle.ApplyFuncContext(context.Background(), b, func(ctx context.Context, b *bundle.Bundle) {
		err := b.Config.Mutate(f)
		require.NoError(t, err)
	})
}

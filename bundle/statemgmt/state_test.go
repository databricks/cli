package statemgmt

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/require"
)

// identityFiler returns a FilerFactory that always returns the provided filer.
func identityFiler(f filer.Filer) deploy.FilerFactory {
	return func(_ *bundle.Bundle) (filer.Filer, error) {
		return f, nil
	}
}

func localStateFile(t *testing.T, ctx context.Context, b *bundle.Bundle) string {
	dir, err := b.CacheDir(ctx, "terraform")
	require.NoError(t, err)
	return filepath.Join(dir, b.StateFileName())
}

func readLocalState(t *testing.T, ctx context.Context, b *bundle.Bundle) map[string]any {
	f, err := os.Open(localStateFile(t, ctx, b))
	require.NoError(t, err)
	defer f.Close()

	var contents map[string]any
	dec := json.NewDecoder(f)
	err = dec.Decode(&contents)
	require.NoError(t, err)
	return contents
}

func writeLocalState(t *testing.T, ctx context.Context, b *bundle.Bundle, contents map[string]any) {
	f, err := os.Create(localStateFile(t, ctx, b))
	require.NoError(t, err)
	defer f.Close()

	enc := json.NewEncoder(f)
	err = enc.Encode(contents)
	require.NoError(t, err)
}

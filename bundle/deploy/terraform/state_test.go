package terraform

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/stretchr/testify/require"
)

func localStateFile(t *testing.T, ctx context.Context, b *bundle.Bundle) string {
	dir, err := Dir(ctx, b)
	require.NoError(t, err)
	return filepath.Join(dir, TerraformStateFileName)
}

func readLocalState(t *testing.T, ctx context.Context, b *bundle.Bundle) map[string]int {
	f, err := os.Open(localStateFile(t, ctx, b))
	require.NoError(t, err)
	defer f.Close()

	var contents map[string]int
	dec := json.NewDecoder(f)
	err = dec.Decode(&contents)
	require.NoError(t, err)
	return contents
}

func writeLocalState(t *testing.T, ctx context.Context, b *bundle.Bundle, contents map[string]int) {
	f, err := os.Create(localStateFile(t, ctx, b))
	require.NoError(t, err)
	defer f.Close()

	enc := json.NewEncoder(f)
	err = enc.Encode(contents)
	require.NoError(t, err)
}

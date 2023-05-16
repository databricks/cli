package mutator_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessInclude(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Workspace: config.Workspace{
				Host: "foo",
			},
		},
	}

	relPath := "./file.yml"
	fullPath := filepath.Join(bundle.Config.Path, relPath)
	f, err := os.Create(fullPath)
	require.NoError(t, err)
	fmt.Fprint(f, "workspace:\n  host: bar\n")
	f.Close()

	assert.Equal(t, "foo", bundle.Config.Workspace.Host)
	_, err = mutator.ProcessInclude(fullPath, relPath).Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "bar", bundle.Config.Workspace.Host)
}

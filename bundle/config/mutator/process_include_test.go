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
	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Workspace: config.Workspace{
				Host: "foo",
			},
		},
	}

	relPath := "./file.yml"
	fullPath := filepath.Join(b.Config.Path, relPath)
	f, err := os.Create(fullPath)
	require.NoError(t, err)
	fmt.Fprint(f, "workspace:\n  host: bar\n")
	f.Close()

	assert.Equal(t, "foo", b.Config.Workspace.Host)
	err = mutator.ProcessInclude(fullPath, relPath).Apply(context.Background(), b)
	require.NoError(t, err)
	assert.Equal(t, "bar", b.Config.Workspace.Host)
}

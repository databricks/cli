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
		RootPath: t.TempDir(),
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "foo",
			},
		},
	}

	relPath := "./file.yml"
	fullPath := filepath.Join(b.RootPath, relPath)
	f, err := os.Create(fullPath)
	require.NoError(t, err)
	fmt.Fprint(f, "workspace:\n  host: bar\n")
	f.Close()

	assert.Equal(t, "foo", b.Config.Workspace.Host)
	diags := bundle.Apply(context.Background(), b, mutator.ProcessInclude(fullPath, relPath))
	require.NoError(t, diags.Error())
	assert.Equal(t, "bar", b.Config.Workspace.Host)
}

package mutator_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessInclude(t *testing.T) {
	root := &config.Root{
		Path: t.TempDir(),
		Workspace: config.Workspace{
			Host: "foo",
		},
	}

	relPath := "./file.yml"
	fullPath := filepath.Join(root.Path, relPath)
	f, err := os.Create(fullPath)
	require.NoError(t, err)
	fmt.Fprint(f, "workspace:\n  host: bar\n")
	f.Close()

	assert.Equal(t, "foo", root.Workspace.Host)
	_, err = mutator.ProcessInclude(fullPath, relPath).Apply(root)
	require.NoError(t, err)
	assert.Equal(t, "bar", root.Workspace.Host)
}

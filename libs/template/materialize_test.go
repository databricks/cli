package template

import (
	"context"
	"os"
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaterializeForNonTemplateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{})
	require.NoError(t, err)
	ctx := auth.SetWorkspaceClient(context.Background(), w)

	// Try to materialize a non-template directory.
	err = Materialize(ctx, "", os.DirFS(tmpDir), nil)
	assert.EqualError(t, err, "not a bundle template: expected to find a template schema file at "+schemaFileName)
}

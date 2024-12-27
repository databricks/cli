package template

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaterializeForNonTemplateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{})
	require.NoError(t, err)
	ctx := root.SetWorkspaceClient(context.Background(), w)

	tmpl := Template{
		TemplateOpts: TemplateOpts{
			ConfigFilePath: "",
			TemplateFS:     os.DirFS(tmpDir),
			OutputFiler:    nil,
		},
	}

	// Try to materialize a non-template directory.
	err = tmpl.Materialize(ctx)
	assert.EqualError(t, err, fmt.Sprintf("not a bundle template: expected to find a template schema file at %s", schemaFileName))
}

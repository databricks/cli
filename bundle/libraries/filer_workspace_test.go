package libraries

import (
	"path"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilerForWorkspace(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/Workspace/Users/shreyas.goenka@databricks.com/a/b/c",
			},
		},
	}

	client, uploadPath, diags := filerForWorkspace(b)
	require.NoError(t, diags.Error())
	assert.Equal(t, path.Join("/Workspace/Users/shreyas.goenka@databricks.com/a/b/c/.internal"), uploadPath)
	assert.IsType(t, &filer.WorkspaceFilesClient{}, client)
}

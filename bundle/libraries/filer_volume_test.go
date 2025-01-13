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

func TestFilerForVolume(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/Volumes/main/my_schema/my_volume/abc",
			},
		},
	}

	client, uploadPath, diags := filerForVolume(b)
	require.NoError(t, diags.Error())
	assert.Equal(t, path.Join("/Volumes/main/my_schema/my_volume/abc/.internal"), uploadPath)
	assert.IsType(t, &filer.FilesClient{}, client)
}

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

func TestFilerForVolumeWithInvalidPrefix(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/Volume/main/my_schema/my_volume",
			},
		},
	}

	_, _, diags := filerForVolume(b)
	require.EqualError(t, diags.Error(), "expected artifact_path to start with /Volumes/, got /Volume/main/my_schema/my_volume")
}

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

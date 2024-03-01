package artifacts

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/testfile"
	"github.com/stretchr/testify/require"
)

type noop struct{}

func (n *noop) Apply(context.Context, *bundle.Bundle) error {
	return nil
}

func (n *noop) Name() string {
	return "noop"
}

func TestExpandGlobFilesSource(t *testing.T) {
	rootPath := t.TempDir()
	err := os.Mkdir(filepath.Join(rootPath, "test"), 0755)
	require.NoError(t, err)

	testfile.CreateFile(t, filepath.Join(rootPath, "test", "myjar1.jar"))
	testfile.CreateFile(t, filepath.Join(rootPath, "test", "myjar2.jar"))

	b := &bundle.Bundle{
		Config: config.Root{
			Path: rootPath,
			Artifacts: map[string]*config.Artifact{
				"test": {
					Type: "custom",
					Files: []config.ArtifactFile{
						{
							Source: filepath.Join(".", "test", "*.jar"),
						},
					},
				},
			},
		},
	}
	u := &upload{"test"}
	uploadMutators[config.ArtifactType("custom")] = func(name string) bundle.Mutator {
		return &noop{}
	}

	err = bundle.Apply(context.Background(), b, u)
	require.NoError(t, err)

	require.Equal(t, 2, len(b.Config.Artifacts["test"].Files))
	require.Equal(t, filepath.Join(rootPath, "test", "myjar1.jar"), b.Config.Artifacts["test"].Files[0].Source)
	require.Equal(t, filepath.Join(rootPath, "test", "myjar2.jar"), b.Config.Artifacts["test"].Files[1].Source)
}

func TestExpandGlobFilesSourceWithNoMatches(t *testing.T) {
	rootPath := t.TempDir()
	err := os.Mkdir(filepath.Join(rootPath, "test"), 0755)
	require.NoError(t, err)

	b := &bundle.Bundle{
		Config: config.Root{
			Path: rootPath,
			Artifacts: map[string]*config.Artifact{
				"test": {
					Type: "custom",
					Files: []config.ArtifactFile{
						{
							Source: filepath.Join(".", "test", "myjar.jar"),
						},
					},
				},
			},
		},
	}
	u := &upload{"test"}
	uploadMutators[config.ArtifactType("custom")] = func(name string) bundle.Mutator {
		return &noop{}
	}

	err = bundle.Apply(context.Background(), b, u)
	require.NoError(t, err)

	// We expect to have the same path as it was provided in the source
	require.Equal(t, 1, len(b.Config.Artifacts["test"].Files))
	require.Equal(t, filepath.Join(rootPath, "test", "myjar.jar"), b.Config.Artifacts["test"].Files[0].Source)
}

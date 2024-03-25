package artifacts

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/bundletest"
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

	t1 := testfile.CreateFile(t, filepath.Join(rootPath, "test", "myjar1.jar"))
	t1.Close(t)

	t2 := testfile.CreateFile(t, filepath.Join(rootPath, "test", "myjar2.jar"))
	t2.Close(t)

	b := &bundle.Bundle{
		Config: config.Root{
			Path: rootPath,
			Artifacts: map[string]*config.Artifact{
				"test": {
					Type: "custom",
					Files: []config.ArtifactFile{
						{
							Source: filepath.Join("..", "test", "*.jar"),
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, ".", filepath.Join(rootPath, "resources", "artifacts.yml"))

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
							Source: filepath.Join("..", "test", "myjar.jar"),
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, ".", filepath.Join(rootPath, "resources", "artifacts.yml"))

	u := &upload{"test"}
	uploadMutators[config.ArtifactType("custom")] = func(name string) bundle.Mutator {
		return &noop{}
	}

	err = bundle.Apply(context.Background(), b, u)
	require.ErrorContains(t, err, "no files found for")
}

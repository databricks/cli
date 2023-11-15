package mutator_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func touch(t *testing.T, path, file string) {
	f, err := os.Create(filepath.Join(path, file))
	require.NoError(t, err)
	f.Close()
}

func TestProcessRootIncludesEmpty(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Path: ".",
		},
	}
	err := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.NoError(t, err)
}

func TestProcessRootIncludesAbs(t *testing.T) {
	// remove this once equivalent tests for windows have been set up
	// or this test has been fixed for windows
	// date: 28 Nov 2022
	if runtime.GOOS == "windows" {
		t.Skip("skipping temperorilty to make windows unit tests green")
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: ".",
			Include: []string{
				"/tmp/*.yml",
			},
		},
	}
	err := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be relative paths")
}

func TestProcessRootIncludesSingleGlob(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Include: []string{
				"*.yml",
			},
		},
	}

	touch(t, b.Config.Path, "databricks.yml")
	touch(t, b.Config.Path, "a.yml")
	touch(t, b.Config.Path, "b.yml")

	err := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.NoError(t, err)

	assert.Equal(t, []string{"a.yml", "b.yml"}, b.Config.Include)
}

func TestProcessRootIncludesMultiGlob(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Include: []string{
				"a*.yml",
				"b*.yml",
			},
		},
	}

	touch(t, b.Config.Path, "a1.yml")
	touch(t, b.Config.Path, "b1.yml")

	err := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.NoError(t, err)

	assert.Equal(t, []string{"a1.yml", "b1.yml"}, b.Config.Include)
}

func TestProcessRootIncludesRemoveDups(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Include: []string{
				"*.yml",
				"*.yml",
			},
		},
	}

	touch(t, b.Config.Path, "a.yml")

	err := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.NoError(t, err)
	assert.Equal(t, []string{"a.yml"}, b.Config.Include)
}

func TestProcessRootIncludesNotExists(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Include: []string{
				"notexist.yml",
			},
		},
	}
	err := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notexist.yml defined in 'include' section does not match any files")
}

func TestProcessRootIncludesExtrasFromEnvVar(t *testing.T) {
	// rootPath := t.TempDir()
	// testYamlName := "extra_include_path.yml"
	// touch(t, rootPath, testYamlName)
	// t.Setenv(env.IncludesVariable, path.Join(rootPath, testYamlName))

	b := &bundle.Bundle{
		Config: config.Root{
			Path: rootPath,
		},
	}

	err := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.NoError(t, err)
	assert.Contains(t, b.Config.Include, testYamlName)
}

func TestProcessRootIncludesDedupExtrasFromEnvVar(t *testing.T) {
	// rootPath := t.TempDir()
	// testYamlName := "extra_include_path.yml"
	// touch(t, rootPath, testYamlName)
	// t.Setenv(env.IncludesVariable, strings.Join(
	// 	[]string{
	// 		path.Join(rootPath, testYamlName),
	// 		path.Join(rootPath, testYamlName),
	// 	},
	// 	string(os.PathListSeparator),
	// ))

	b := &bundle.Bundle{
		Config: config.Root{
			Path: rootPath,
		},
	}

	err := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.NoError(t, err)
	assert.Equal(t, []string{testYamlName}, b.Config.Include)
}

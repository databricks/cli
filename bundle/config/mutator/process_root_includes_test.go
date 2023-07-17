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
	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: ".",
		},
	}
	err := mutator.ProcessRootIncludes().Apply(context.Background(), bundle)
	require.NoError(t, err)
}

func TestProcessRootIncludesAbs(t *testing.T) {
	// remove this once equivalent tests for windows have been set up
	// or this test has been fixed for windows
	// date: 28 Nov 2022
	if runtime.GOOS == "windows" {
		t.Skip("skipping temperorilty to make windows unit tests green")
	}

	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: ".",
			Include: []string{
				"/tmp/*.yml",
			},
		},
	}
	err := mutator.ProcessRootIncludes().Apply(context.Background(), bundle)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be relative paths")
}

func TestProcessRootIncludesSingleGlob(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Include: []string{
				"*.yml",
			},
		},
	}

	touch(t, bundle.Config.Path, "databricks.yaml")
	touch(t, bundle.Config.Path, "a.yml")
	touch(t, bundle.Config.Path, "b.yml")

	err := mutator.ProcessRootIncludes().Apply(context.Background(), bundle)
	require.NoError(t, err)

	assert.Equal(t, []string{"a.yml", "b.yml"}, bundle.Config.Include)
}

func TestProcessRootIncludesMultiGlob(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Include: []string{
				"a*.yml",
				"b*.yml",
			},
		},
	}

	touch(t, bundle.Config.Path, "a1.yml")
	touch(t, bundle.Config.Path, "b1.yml")

	err := mutator.ProcessRootIncludes().Apply(context.Background(), bundle)
	require.NoError(t, err)

	assert.Equal(t, []string{"a1.yml", "b1.yml"}, bundle.Config.Include)
}

func TestProcessRootIncludesRemoveDups(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Include: []string{
				"*.yml",
				"*.yml",
			},
		},
	}

	touch(t, bundle.Config.Path, "a.yml")

	err := mutator.ProcessRootIncludes().Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, []string{"a.yml"}, bundle.Config.Include)
}

func TestProcessRootIncludesNotExists(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Include: []string{
				"notexist.yml",
			},
		},
	}
	err := mutator.ProcessRootIncludes().Apply(context.Background(), bundle)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notexist.yml defined in 'include' section does not match any files")
}

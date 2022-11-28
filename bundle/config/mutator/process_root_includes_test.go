package mutator_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/mutator"
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
	_, err := mutator.ProcessRootIncludes().Apply(context.Background(), bundle)
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
	_, err := mutator.ProcessRootIncludes().Apply(context.Background(), bundle)
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

	touch(t, bundle.Config.Path, "bundle.yml")
	touch(t, bundle.Config.Path, "a.yml")
	touch(t, bundle.Config.Path, "b.yml")

	ms, err := mutator.ProcessRootIncludes().Apply(context.Background(), bundle)
	require.NoError(t, err)

	var names []string
	for _, m := range ms {
		names = append(names, m.Name())
	}

	assert.NotContains(t, names, "ProcessInclude(bundle.yml)")
	assert.Contains(t, names, "ProcessInclude(a.yml)")
	assert.Contains(t, names, "ProcessInclude(b.yml)")
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

	ms, err := mutator.ProcessRootIncludes().Apply(context.Background(), bundle)
	require.NoError(t, err)

	var names []string
	for _, m := range ms {
		names = append(names, m.Name())
	}

	assert.Contains(t, names, "ProcessInclude(a1.yml)")
	assert.Contains(t, names, "ProcessInclude(b1.yml)")
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

	ms, err := mutator.ProcessRootIncludes().Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Len(t, ms, 1)
	assert.Equal(t, "ProcessInclude(a.yml)", ms[0].Name())
}

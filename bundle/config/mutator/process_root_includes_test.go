package mutator_test

import (
	"context"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessRootIncludesEmpty(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Path: ".",
		},
	}
	diags := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.Empty(t, diags)

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
	diags := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.True(t, diags.HasError())
	assert.ErrorContains(t, diags.Error(), "must be relative paths")
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

	testutil.Touch(t, b.Config.Path, "databricks.yml")
	testutil.Touch(t, b.Config.Path, "a.yml")
	testutil.Touch(t, b.Config.Path, "b.yml")

	diags := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.Empty(t, diags)

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

	testutil.Touch(t, b.Config.Path, "a1.yml")
	testutil.Touch(t, b.Config.Path, "b1.yml")

	diags := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.Empty(t, diags)

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

	testutil.Touch(t, b.Config.Path, "a.yml")

	diags := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.Empty(t, diags)

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
	diags := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.True(t, diags.HasError())
	assert.ErrorContains(t, diags.Error(), "notexist.yml defined in 'include' section does not match any files")
}

func TestProcessRootIncludesExtrasFromEnvVar(t *testing.T) {
	rootPath := t.TempDir()
	testYamlName := "extra_include_path.yml"
	testutil.Touch(t, rootPath, testYamlName)
	t.Setenv(env.IncludesVariable, path.Join(rootPath, testYamlName))

	b := &bundle.Bundle{
		Config: config.Root{
			Path: rootPath,
		},
	}

	diags := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.Empty(t, diags)

	assert.Contains(t, b.Config.Include, testYamlName)
}

func TestProcessRootIncludesDedupExtrasFromEnvVar(t *testing.T) {
	rootPath := t.TempDir()
	testYamlName := "extra_include_path.yml"
	testutil.Touch(t, rootPath, testYamlName)
	t.Setenv(env.IncludesVariable, strings.Join(
		[]string{
			path.Join(rootPath, testYamlName),
			path.Join(rootPath, testYamlName),
		},
		string(os.PathListSeparator),
	))

	b := &bundle.Bundle{
		Config: config.Root{
			Path: rootPath,
		},
	}

	diags := bundle.Apply(context.Background(), b, mutator.ProcessRootIncludes())
	require.Empty(t, diags)

	assert.Equal(t, []string{testYamlName}, b.Config.Include)
}

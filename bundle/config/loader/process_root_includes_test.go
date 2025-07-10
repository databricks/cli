package loader_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/loader"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessRootIncludesEmpty(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: ".",
	}
	diags := bundle.Apply(context.Background(), b, loader.ProcessRootIncludes())
	require.NoError(t, diags.Error())
}

func TestProcessRootIncludesAbs(t *testing.T) {
	// remove this once equivalent tests for windows have been set up
	// or this test has been fixed for windows
	// date: 28 Nov 2022
	if runtime.GOOS == "windows" {
		t.Skip("skipping temperorilty to make windows unit tests green")
	}

	b := &bundle.Bundle{
		BundleRootPath: ".",
		Config: config.Root{
			Include: []string{
				"/tmp/*.yml",
			},
		},
	}
	diags := bundle.Apply(context.Background(), b, loader.ProcessRootIncludes())
	require.True(t, diags.HasError())
	assert.ErrorContains(t, diags.Error(), "must be relative paths")
}

func TestProcessRootIncludesSingleGlob(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Include: []string{
				"*.yml",
			},
		},
	}

	testutil.Touch(t, b.BundleRootPath, "databricks.yml")
	testutil.Touch(t, b.BundleRootPath, "a.yml")
	testutil.Touch(t, b.BundleRootPath, "b.yml")

	diags := bundle.Apply(context.Background(), b, loader.ProcessRootIncludes())
	require.NoError(t, diags.Error())
	assert.Equal(t, []string{"a.yml", "b.yml"}, b.Config.Include)
}

func TestProcessRootIncludesMultiGlob(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Include: []string{
				"a*.yml",
				"b*.yml",
			},
		},
	}

	testutil.Touch(t, b.BundleRootPath, "a1.yml")
	testutil.Touch(t, b.BundleRootPath, "b1.yml")

	diags := bundle.Apply(context.Background(), b, loader.ProcessRootIncludes())
	require.NoError(t, diags.Error())
	assert.Equal(t, []string{"a1.yml", "b1.yml"}, b.Config.Include)
}

func TestProcessRootIncludesRemoveDups(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Include: []string{
				"*.yml",
				"*.yml",
			},
		},
	}

	testutil.Touch(t, b.BundleRootPath, "a.yml")

	diags := bundle.Apply(context.Background(), b, loader.ProcessRootIncludes())
	require.NoError(t, diags.Error())
	assert.Equal(t, []string{"a.yml"}, b.Config.Include)
}

func TestProcessRootIncludesNotExists(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Include: []string{
				"notexist.yml",
			},
		},
	}
	diags := bundle.Apply(context.Background(), b, loader.ProcessRootIncludes())
	require.True(t, diags.HasError())
	assert.ErrorContains(t, diags.Error(), "notexist.yml defined in 'include' section does not match any files")
}

func TestProcessRootIncludesGlobInRootPath(t *testing.T) {
	tests := []struct {
		name string
		root string
		diag diag.Diagnostic
	}{
		{
			name: "star",
			root: "foo/a*",
			diag: diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Bundle root path contains glob pattern characters",
				Detail:   `The path to the bundle root foo/a* contains glob pattern character "*". Please remove the character from this path to use bundle commands.`,
			},
		},
		{
			name: "question mark",
			root: "bar/?b",
			diag: diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Bundle root path contains glob pattern characters",
				Detail:   `The path to the bundle root bar/?b contains glob pattern character "?". Please remove the character from this path to use bundle commands.`,
			},
		},
		{
			name: "left bracket",
			root: "[ab",
			diag: diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Bundle root path contains glob pattern characters",
				Detail:   `The path to the bundle root [ab contains glob pattern character "[". Please remove the character from this path to use bundle commands.`,
			},
		},
		{
			name: "right bracket",
			root: "ab]/bax",
			diag: diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Bundle root path contains glob pattern characters",
				Detail:   `The path to the bundle root ab]/bax contains glob pattern character "]". Please remove the character from this path to use bundle commands.`,
			},
		},
		{
			name: "hat",
			root: "ab^bax",
			diag: diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Bundle root path contains glob pattern characters",
				Detail:   `The path to the bundle root ab^bax contains glob pattern character "^". Please remove the character from this path to use bundle commands.`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := &bundle.Bundle{
				BundleRootPath: test.root,
			}

			diags := bundle.Apply(context.Background(), b, loader.ProcessRootIncludes())
			require.True(t, diags.HasError())
			assert.Len(t, diags, 1)
			assert.Equal(t, test.diag, diags[0])
		})
	}
}

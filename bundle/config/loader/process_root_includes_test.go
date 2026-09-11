package loader_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/loader"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessRootIncludesEmpty(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: ".",
	}
	diags := bundle.Apply(t.Context(), b, loader.ProcessRootIncludes())
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
	diags := bundle.Apply(t.Context(), b, loader.ProcessRootIncludes())
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

	diags := bundle.Apply(t.Context(), b, loader.ProcessRootIncludes())
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

	diags := bundle.Apply(t.Context(), b, loader.ProcessRootIncludes())
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

	diags := bundle.Apply(t.Context(), b, loader.ProcessRootIncludes())
	require.NoError(t, diags.Error())
	assert.Equal(t, []string{"a.yml"}, b.Config.Include)
}

// The expanded include list must be visible in both the typed and the dynamic
// configuration: the per-file includes are applied without their own mutator scope, so
// nothing converts the typed field back into the dynamic tree afterwards.
func TestProcessRootIncludesUpdatesDynamicValue(t *testing.T) {
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

	diags := bundle.Apply(t.Context(), b, loader.ProcessRootIncludes())
	require.NoError(t, diags.Error())
	assert.Equal(t, []string{"a.yml"}, b.Config.Include)

	assert.Equal(t, []any{"a.yml"}, b.Config.Value().Get("include").AsAny())
}

// An empty include list must stay absent from the dynamic tree: the typed field is
// omitempty, so writing [] would add an empty "include" to `bundle validate -o json`.
func TestProcessRootIncludesEmptyOmitsDynamicValue(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Include: []string{
				"*.yml",
			},
		},
	}

	testutil.Touch(t, b.BundleRootPath, "databricks.yml")

	diags := bundle.Apply(t.Context(), b, loader.ProcessRootIncludes())
	require.NoError(t, diags.Error())
	assert.Empty(t, b.Config.Include)
	assert.Equal(t, dyn.KindInvalid, b.Config.Value().Get("include").Kind())
}

// Merge semantics across included files must be unaffected by how the per-file includes
// are applied: maps merge per key with the later file winning, sequences concatenate, and
// locations accumulate (UniqueResourceKeys reports duplicates by counting locations).
func TestProcessRootIncludesMergesAcrossFiles(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Include: []string{
				"*.yml",
			},
		},
	}

	testutil.WriteFile(t, filepath.Join(b.BundleRootPath, "a.yml"), `
resources:
  jobs:
    shared:
      max_concurrent_runs: 1
      tags:
        from_a: yes_a
      tasks:
        - task_key: task_a
`)

	testutil.WriteFile(t, filepath.Join(b.BundleRootPath, "b.yml"), `
resources:
  jobs:
    shared:
      tags:
        from_b: yes_b
      tasks:
        - task_key: task_b
`)

	diags := bundle.Apply(t.Context(), b, loader.ProcessRootIncludes())
	require.NoError(t, diags.Error())

	job := b.Config.Value().Get("resources").Get("jobs").Get("shared")

	// Set only in a.yml: a per-key map merge must not drop it.
	assert.Equal(t, int64(1), job.Get("max_concurrent_runs").MustInt())

	// Maps merge per key across both files.
	assert.Equal(t, map[string]any{"from_a": "yes_a", "from_b": "yes_b"}, job.Get("tags").AsAny())

	// Sequences concatenate rather than overwrite.
	assert.Equal(t, []any{
		map[string]any{"task_key": "task_a"},
		map[string]any{"task_key": "task_b"},
	}, job.Get("tasks").AsAny())

	// Both definitions must remain visible, otherwise duplicate keys go unreported.
	assert.Len(t, job.Locations(), 2)
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
	diags := bundle.Apply(t.Context(), b, loader.ProcessRootIncludes())
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

			diags := bundle.Apply(t.Context(), b, loader.ProcessRootIncludes())
			require.True(t, diags.HasError())
			assert.Len(t, diags, 1)
			assert.Equal(t, test.diag, diags[0])
		})
	}
}

package template

import (
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleRoots(t *testing.T) {
	tcases := []struct {
		name     string
		files    []string
		expected []string
	}{
		{
			name:     "no bundle configuration file",
			files:    []string{"README.md", "src/main.py"},
			expected: nil,
		},
		{
			// The nested and top-level shapes are covered end to end by the
			// nested-output and standard-output-json acceptance tests; this pins
			// the ordering and the "." normalization for a root that is the
			// output directory itself.
			name: "multiple bundle configuration files are sorted",
			files: []string{
				"projects/beta/qq/databricks.yml",
				"projects/alpha/yz/databricks.yml",
				"databricks.yml",
			},
			expected: []string{".", "projects/alpha/yz", "projects/beta/qq"},
		},
		{
			name: "all bundle configuration file names are recognized",
			files: []string{
				"a/databricks.yml",
				"b/databricks.yaml",
				"c/bundle.yml",
				"d/bundle.yaml",
			},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name: "multiple configuration files in one directory are reported once",
			files: []string{
				"a/databricks.yml",
				"a/bundle.yml",
			},
			expected: []string{"a"},
		},
		{
			name:     "a file only named like a bundle configuration file is ignored",
			files:    []string{"src/not-databricks.yml", "src/databricks.yml.tmpl"},
			expected: nil,
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, bundleRoots(tc.files))
		})
	}
}

// A databricks.yml removed by a {{skip}} directive must not be reported as a
// bundle root. persistToDisk only records the files that survive the skip
// patterns, so this asserts the roots are derived from the filtered list.
func TestBundleRootsSkippedFileIsNotReported(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := t.Context()

	r := &renderer{
		ctx:          ctx,
		skipPatterns: []string{"projects/alpha/yz/databricks.yml"},
		files: []file{
			&inMemoryFile{
				perm:    0o444,
				relPath: "projects/alpha/yz/databricks.yml",
				content: []byte("bundle:\n  name: yz\n"),
			},
			&inMemoryFile{
				perm:    0o444,
				relPath: "projects/beta/qq/databricks.yml",
				content: []byte("bundle:\n  name: qq\n"),
			},
		},
	}

	out, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)
	err = r.persistToDisk(ctx, out)
	require.NoError(t, err)

	assert.Equal(t, []string{"projects/beta/qq"}, bundleRoots(r.persistedPaths))
}

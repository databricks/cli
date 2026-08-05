package template

import (
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The outputs are sorted, and a file removed by a {{skip}} directive is not
// reported: persistToDisk only records the files that survive the skip patterns.
func TestInitResultOutputs(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := t.Context()

	w := &defaultWriter{
		renderer: &renderer{
			ctx:          ctx,
			skipPatterns: []string{"projects/alpha/yz/yz.sql"},
			files: []file{
				&inMemoryFile{
					perm:    0o444,
					relPath: "projects/alpha/yz/src/yz.py",
					content: []byte("print(1)\n"),
				},
				&inMemoryFile{
					perm:    0o444,
					relPath: "projects/alpha/yz/databricks.yml",
					content: []byte("bundle:\n  name: yz\n"),
				},
				&inMemoryFile{
					perm:    0o444,
					relPath: "projects/alpha/yz/yz.sql",
					content: []byte("SELECT 1\n"),
				},
			},
		},
	}

	out, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)
	require.NoError(t, w.renderer.persistToDisk(ctx, out))

	assert.Equal(t, &InitResult{Outputs: []string{
		"projects/alpha/yz/databricks.yml",
		"projects/alpha/yz/src/yz.py",
	}}, w.InitResult())
}

// InitResult is nil before Materialize has run, so the command does not render
// an empty payload.
func TestInitResultBeforeMaterialize(t *testing.T) {
	w := &defaultWriter{}
	assert.Nil(t, w.InitResult())
}

package zip

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZipExtract(t *testing.T) {
	tmpDir := t.TempDir()
	var err error

	err = Extract("./testdata/dir.zip", tmpDir)
	assert.NoError(t, err)

	assert.DirExists(t, filepath.Join(tmpDir, "dir"))

	b, err := os.ReadFile(filepath.Join(tmpDir, "dir/a"))
	assert.NoError(t, err)
	assert.Equal(t, "hello a\n", string(b))

	b, err = os.ReadFile(filepath.Join(tmpDir, "dir/b"))
	assert.NoError(t, err)
	assert.Equal(t, "hello b\n", string(b))
}

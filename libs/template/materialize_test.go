package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupConfig(t *testing.T, config string) string {
	// create target directory with the input config
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "config.json"))
	require.NoError(t, err)
	_, err = f.WriteString(config)
	f.Close()
	require.NoError(t, err)
	return tmp
}

func TestMaterializeEmptyDirsAreNotGenerated(t *testing.T) {
	tmp := setupConfig(t, `
	{
		"a": "dir-with-file",
		"b": "foo",
		"c": "dir-with-skipped-file",
		"d": "skipping"
	}`)
	err := Materialize("./testdata/skip_dir", tmp, filepath.Join(tmp, "config.json"))
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(tmp, "dir-with-file"))
	assert.FileExists(t, filepath.Join(tmp, "dir-with-file/.gitkeep"))
	assert.NoDirExists(t, filepath.Join(tmp, "empty-dir"))
	assert.NoDirExists(t, filepath.Join(tmp, "dir-with-skipped-file"))

	tmp2 := setupConfig(t, `
	{
		"a": "dir-with-file",
		"b": "foo",
		"c": "dir-not-skipped-this-time",
		"d": "not-skipping"
	}`)
	err = Materialize("./testdata/skip_dir", tmp2, filepath.Join(tmp2, "config.json"))
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(tmp2, "dir-with-file"))
	assert.FileExists(t, filepath.Join(tmp2, "dir-with-file/.gitkeep"))
	assert.DirExists(t, filepath.Join(tmp2, "dir-not-skipped-this-time"))
	assert.FileExists(t, filepath.Join(tmp2, "dir-not-skipped-this-time/foo"))
}

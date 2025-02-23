package notebook

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/fakefs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectSource(t *testing.T) {
	var nb bool
	var lang workspace.Language
	var err error

	nb, lang, err = Detect("./testdata/py_source.py")
	require.NoError(t, err)
	assert.True(t, nb)
	assert.Equal(t, workspace.LanguagePython, lang)

	nb, lang, err = Detect("./testdata/r_source.r")
	require.NoError(t, err)
	assert.True(t, nb)
	assert.Equal(t, workspace.LanguageR, lang)

	nb, lang, err = Detect("./testdata/scala_source.scala")
	require.NoError(t, err)
	assert.True(t, nb)
	assert.Equal(t, workspace.LanguageScala, lang)

	nb, lang, err = Detect("./testdata/sql_source.sql")
	require.NoError(t, err)
	assert.True(t, nb)
	assert.Equal(t, workspace.LanguageSql, lang)

	nb, lang, err = Detect("./testdata/txt.txt")
	require.NoError(t, err)
	assert.False(t, nb)
	assert.Equal(t, workspace.Language(""), lang)
}

func TestDetectCallsDetectJupyter(t *testing.T) {
	nb, lang, err := Detect("./testdata/py_ipynb.ipynb")
	require.NoError(t, err)
	assert.True(t, nb)
	assert.Equal(t, workspace.LanguagePython, lang)
}

func TestDetectUnknownExtension(t *testing.T) {
	_, _, err := Detect("./testdata/doesntexist.foobar")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	nb, _, err := Detect("./testdata/unknown_extension.foobar")
	require.NoError(t, err)
	assert.False(t, nb)
}

func TestDetectNoExtension(t *testing.T) {
	_, _, err := Detect("./testdata/doesntexist")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	nb, _, err := Detect("./testdata/no_extension")
	require.NoError(t, err)
	assert.False(t, nb)
}

func TestDetectFileDoesNotExists(t *testing.T) {
	_, _, err := Detect("./testdata/doesntexist.py")
	require.Error(t, err)
}

func TestDetectEmptyFile(t *testing.T) {
	// Create empty file.
	dir := t.TempDir()
	path := filepath.Join(dir, "file.py")
	err := os.WriteFile(path, nil, 0o644)
	require.NoError(t, err)

	// No contents means not a notebook.
	nb, _, err := Detect(path)
	require.NoError(t, err)
	assert.False(t, nb)
}

func TestDetectFileWithLongHeader(t *testing.T) {
	// Create 128kb garbage file.
	dir := t.TempDir()
	path := filepath.Join(dir, "file.py")
	buf := make([]byte, 128*1024)
	err := os.WriteFile(path, buf, 0o644)
	require.NoError(t, err)

	// Garbage contents means not a notebook.
	nb, _, err := Detect(path)
	require.NoError(t, err)
	assert.False(t, nb)
}

type fileInfoWithWorkspaceInfo struct {
	fakefs.FileInfo

	oi workspace.ObjectInfo
}

func (f fileInfoWithWorkspaceInfo) WorkspaceObjectInfo() workspace.ObjectInfo {
	return f.oi
}

func TestDetectWithObjectInfo(t *testing.T) {
	fakefs := fakefs.FS{
		"file.py": fakefs.File{
			FileInfo: fileInfoWithWorkspaceInfo{
				oi: workspace.ObjectInfo{
					ObjectType: workspace.ObjectTypeNotebook,
					Language:   workspace.LanguagePython,
				},
			},
		},
	}

	nb, lang, err := DetectWithFS(fakefs, "file.py")
	require.NoError(t, err)
	assert.True(t, nb)
	assert.Equal(t, workspace.LanguagePython, lang)
}

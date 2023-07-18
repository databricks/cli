package template

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
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

func assertFilePerm(t *testing.T, path string, perm fs.FileMode) {
	stat, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, perm, stat.Mode().Perm())
}

func assertFileContent(t *testing.T, path string, content string) {
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(b))
}

func TestMaterializeEmptyDirsAreNotGenerated(t *testing.T) {
	tmp := setupConfig(t, `
	{
		"a": "this directory is created because it contains a file",
		"b": "this variable is not used anywhere",
		"c": "this directory will be skipped if d=foo",
		"d": "foo"
	}`)
	err := Materialize("./testdata/skip_dir", tmp, filepath.Join(tmp, "config.json"))
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(tmp, "this directory is created because it contains a file"))
	assert.FileExists(t, filepath.Join(tmp, "this directory is created because it contains a file/.gitkeep"))
	assert.NoDirExists(t, filepath.Join(tmp, "this directory will be skipped if d=foo"))

	tmp2 := setupConfig(t, `
	{
		"a": "this directory is created because it contains a file",
		"b": "this variable is not used anywhere",
		"c": "this directory will be skipped if d=foo",
		"d": "bar"
	}`)
	err = Materialize("./testdata/skip_dir", tmp2, filepath.Join(tmp2, "config.json"))
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(tmp2, "this directory is created because it contains a file"))
	assert.FileExists(t, filepath.Join(tmp2, "this directory is created because it contains a file/.gitkeep"))
	assert.DirExists(t, filepath.Join(tmp2, "this directory will be skipped if d=foo"))
	assert.FileExists(t, filepath.Join(tmp2, "this directory will be skipped if d=foo/abc"))
}

func TestMaterializeFilePermissionsAreCopiedForUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}

	tmp := t.TempDir()

	// setup template in temp directory
	err := os.Mkdir(filepath.Join(tmp, "my_tmpl"), 0777)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmp, "my_tmpl", "databricks_template_schema.json"), []byte(`
	{
		"properties": {
			"a": {
				"type": "string"
			},
			"b": {
				"type": "string"
			}
		}
	}`), 0644)
	require.NoError(t, err)
	err = os.Mkdir(filepath.Join(tmp, "my_tmpl", "template"), 0777)
	require.NoError(t, err)

	// A normal file with the executable bit not flipped
	err = os.WriteFile(filepath.Join(tmp, "my_tmpl", "template", "{{.a}}"), []byte("abc"), 0600)
	require.NoError(t, err)

	// A read only file
	err = os.WriteFile(filepath.Join(tmp, "my_tmpl", "template", "{{.b}}"), []byte("def"), 0400)
	require.NoError(t, err)

	// A read only executable file
	err = os.WriteFile(filepath.Join(tmp, "my_tmpl", "template", "foo"), []byte("ghi"), 0500)
	require.NoError(t, err)

	// An executable script, accessable by non user access classes
	err = os.WriteFile(filepath.Join(tmp, "my_tmpl", "template", "bar"), []byte("ghi"), 0755)
	require.NoError(t, err)

	// create config.json file
	err = os.Mkdir(filepath.Join(tmp, "config"), 0777)
	require.NoError(t, err)
	configPath := filepath.Join(tmp, "config", "config.json")
	err = os.WriteFile(configPath, []byte(`
	{
		"a": "Amsterdam",
		"b": "Hague"
	}`), 0644)
	require.NoError(t, err)

	// create directory to initialize the template in
	instanceRoot := filepath.Join(tmp, "instance")
	err = os.Mkdir(instanceRoot, 0777)
	require.NoError(t, err)

	// materialize the template
	err = Materialize(filepath.Join(tmp, "my_tmpl"), instanceRoot, configPath)
	require.NoError(t, err)

	// assert template files have the correct permission bits set
	assertFilePerm(t, filepath.Join(instanceRoot, "Amsterdam"), 0600)
	assertFilePerm(t, filepath.Join(instanceRoot, "Hague"), 0400)
	assertFilePerm(t, filepath.Join(instanceRoot, "foo"), 0500)
	assertFilePerm(t, filepath.Join(instanceRoot, "bar"), 0755)
}

func TestMaterializeFilePermissionsAreCopiedForWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	tmp := t.TempDir()

	// create template in temp directory
	err := os.Mkdir(filepath.Join(tmp, "my_tmpl"), 0777)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmp, "my_tmpl", "databricks_template_schema.json"), []byte(`
	{
		"properties": {
			"a": {
				"type": "string"
			},
			"b": {
				"type": "string"
			}
		}
	}`), 0644)
	require.NoError(t, err)

	// A normal file with the executable bit not flipped
	err = os.Mkdir(filepath.Join(tmp, "my_tmpl", "template"), 0777)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmp, "my_tmpl", "template", "{{.a}}"), []byte("abc"), 0666)
	require.NoError(t, err)

	// A read only file
	err = os.WriteFile(filepath.Join(tmp, "my_tmpl", "template", "{{.b}}"), []byte("def"), 0444)
	require.NoError(t, err)

	// create config.json file
	err = os.Mkdir(filepath.Join(tmp, "config"), 0777)
	require.NoError(t, err)
	configPath := filepath.Join(tmp, "config", "config.json")
	err = os.WriteFile(configPath, []byte(`
	{
		"a": "Amsterdam",
		"b": "Hague"
	}`), 0644)
	require.NoError(t, err)

	// create directory to initialize the template in
	instanceRoot := filepath.Join(tmp, "instance")
	err = os.Mkdir(instanceRoot, 0777)
	require.NoError(t, err)

	// materialize the template
	err = Materialize(filepath.Join(tmp, "my_tmpl"), instanceRoot, configPath)
	require.NoError(t, err)

	// assert template files have the correct permission bits set
	assertFilePerm(t, filepath.Join(instanceRoot, "Amsterdam"), 0666)
	assertFilePerm(t, filepath.Join(instanceRoot, "Hague"), 0444)
}

package template

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRendererVariableRead(t *testing.T) {
	r, err := newRenderer(nil, "./testdata/email/library")
	require.NoError(t, err)

	tmpDir := t.TempDir()

	err = walkFileTree(r, "./testdata/email/template", tmpDir)
	require.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(tmpDir, "my_email"))
	require.NoError(t, err)

	assert.Equal(t, "shreyas.goenka@databricks.com\n", string(b))
}

func TestRendererUrlParseUsageInFunction(t *testing.T) {
	r, err := newRenderer(nil, "./testdata/get_host/library")
	require.NoError(t, err)

	tmpDir := t.TempDir()

	err = walkFileTree(r, "./testdata/get_host/template", tmpDir)
	require.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(tmpDir, "my_host"))
	require.NoError(t, err)

	assert.Equal(t, "https://www.host.com\n", string(b))
}

func TestRendererRegexpCheckFailing(t *testing.T) {
	r, err := newRenderer(nil, "./testdata/is_https/library")
	require.NoError(t, err)

	tmpDir := t.TempDir()

	err = walkFileTree(r, "./testdata/is_https/template_not_https", tmpDir)
	require.NoError(t, err)
}

func TestRendererRegexpCheckPassing(t *testing.T) {
	r, err := newRenderer(nil, "./testdata/is_https/library")
	require.NoError(t, err)

	tmpDir := t.TempDir()

	err = walkFileTree(r, "./testdata/is_https/template_is_https", tmpDir)
	require.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(tmpDir, "my_check"))
	require.NoError(t, err)

	assert.Equal(t, "this file is created if validation passes\n", string(b))
}

func TestExecuteTemplate(t *testing.T) {
	templateText :=
		`"{{.count}} items are made of {{.Material}}".
{{if eq .Animal "sheep" }}
Sheep wool is the best!
{{else}}
{{.Animal}} wool is not too bad...
{{end}}
My email is {{template "email"}}
`

	r := renderer{
		config: map[string]any{
			"Material": "wool",
			"count":    1,
			"Animal":   "sheep",
		},
		baseTemplate: template.Must(template.New("base").Parse(`{{define "email"}}shreyas.goenka@databricks.com{{end}}`)),
	}

	statement, err := r.executeTemplate(templateText)
	require.NoError(t, err)
	assert.Contains(t, statement, `"1 items are made of wool"`)
	assert.NotContains(t, statement, `cat wool is not too bad.."`)
	assert.Contains(t, statement, "Sheep wool is the best!")
	assert.Contains(t, statement, `My email is shreyas.goenka@databricks.com`)

	r = renderer{
		config: map[string]any{
			"Material": "wool",
			"count":    1,
			"Animal":   "cat",
		},
		baseTemplate: template.Must(template.New("base").Parse(`{{define "email"}}hrithik.roshan@databricks.com{{end}}`)),
	}

	statement, err = r.executeTemplate(templateText)
	require.NoError(t, err)
	assert.Contains(t, statement, `"1 items are made of wool"`)
	assert.Contains(t, statement, `cat wool is not too bad...`)
	assert.NotContains(t, statement, "Sheep wool is the best!")
	assert.Contains(t, statement, `My email is hrithik.roshan@databricks.com`)
}

func TestGenerateFile(t *testing.T) {
	tmp := t.TempDir()

	pathTemplate := filepath.Join(tmp, "{{.Animal}}", "{{.Material}}", "foo", "{{.count}}.txt")
	contentTemplate := `"{{.count}} items are made of {{.Material}}".
	{{if eq .Animal "sheep" }}
	Sheep wool is the best!
	{{else}}
	{{.Animal}} wool is not too bad...
	{{end}}
	`

	r := renderer{
		config: map[string]any{
			"Material": "wool",
			"count":    1,
			"Animal":   "cat",
		},
		baseTemplate: template.New("base"),
	}
	f, err := r.generateFile(pathTemplate, contentTemplate, 0444)
	require.NoError(t, err)

	// assert file content
	assert.Equal(t, "\"1 items are made of wool\".\n\t\n\tcat wool is not too bad...\n\t\n\t", f.content)

	// assert file permissions are correctly assigned
	assert.Equal(t, fs.FileMode(0444), f.perm)

	// assert file path
	assert.Equal(t, filepath.Join(tmp, "cat", "wool", "foo", "1.txt"), f.path)
}

func TestDeleteSkippedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	inputFiles := map[*inMemoryFile]any{
		{
			path:    filepath.Join(tmpDir, "aaa"),
			content: "one",
			perm:    0444,
		}: nil,
		{
			path:    filepath.Join(tmpDir, "abb"),
			content: "two",
			perm:    0444,
		}: nil,
		{
			path:    filepath.Join(tmpDir, "bbb"),
			content: "three",
			perm:    0666,
		}: nil,
	}

	err := deleteSkippedFiles(inputFiles, []string{"aaa", "abb"})
	require.NoError(t, err)

	assert.Len(t, inputFiles, 1)
	for v := range inputFiles {
		assert.Equal(t, inMemoryFile{
			path:    filepath.Join(tmpDir, "bbb"),
			content: "three",
			perm:    0666,
		}, *v)
	}
}

func TestDeleteSkippedFilesWithGlobPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	inputFiles := map[*inMemoryFile]any{
		{
			path:    filepath.Join(tmpDir, "aaa"),
			content: "one",
			perm:    0444,
		}: nil,
		{
			path:    filepath.Join(tmpDir, "abb"),
			content: "two",
			perm:    0444,
		}: nil,
		{
			path:    filepath.Join(tmpDir, "bbb"),
			content: "three",
			perm:    0666,
		}: nil,
		{
			path:    filepath.Join(tmpDir, "ddd"),
			content: "four",
			perm:    0666,
		}: nil,
	}

	err := deleteSkippedFiles(inputFiles, []string{"a*"})
	require.NoError(t, err)

	files := make([]inMemoryFile, 0)
	for v := range inputFiles {
		files = append(files, *v)
	}
	assert.Len(t, files, 2)
	assert.Contains(t, files, inMemoryFile{
		path:    filepath.Join(tmpDir, "bbb"),
		content: "three",
		perm:    0666,
	})
	assert.Contains(t, files, inMemoryFile{
		path:    filepath.Join(tmpDir, "ddd"),
		content: "four",
		perm:    0666,
	})
}

func TestSkipAllFiles(t *testing.T) {
	tmpDir := t.TempDir()
	inputFiles := map[*inMemoryFile]any{
		{
			path:    filepath.Join(tmpDir, "aaa"),
			content: "one",
			perm:    0444,
		}: nil,
		{
			path:    filepath.Join(tmpDir, "abb"),
			content: "two",
			perm:    0444,
		}: nil,
		{
			path:    filepath.Join(tmpDir, "bbb"),
			content: "three",
			perm:    0666,
		}: nil,
	}

	err := deleteSkippedFiles(inputFiles, []string{"*"})
	require.NoError(t, err)
	assert.Len(t, inputFiles, 0)
}

func TestTemplateMaterializeFiles(t *testing.T) {
	tmpDir := t.TempDir()
	inputFiles := map[*inMemoryFile]any{
		{
			path:    filepath.Join(tmpDir, "aaa"),
			content: "one",
			perm:    0444,
		}: nil,
		{
			path:    filepath.Join(tmpDir, "abb"),
			content: "two",
			perm:    0444,
		}: nil,
		{
			path:    filepath.Join(tmpDir, "bbb"),
			content: "three",
			perm:    0666,
		}: nil,
	}

	err := materializeFiles(inputFiles)
	assert.NoError(t, err)

	path := filepath.Join(tmpDir, "aaa")
	assertFilePerm(t, path, 0444)
	assertFileContent(t, path, "one")

	path = filepath.Join(tmpDir, "abb")
	assertFilePerm(t, path, 0444)
	assertFileContent(t, path, "two")

	path = filepath.Join(tmpDir, "bbb")
	assertFilePerm(t, path, 0666)
	assertFileContent(t, path, "three")
}

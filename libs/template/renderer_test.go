package template

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertFileContent(t *testing.T, path string, content string) {
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(b))
}

func TestRendererWithAssociatedTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	r, err := newRenderer(context.Background(), nil, "./testdata/email/library", tmpDir, "./testdata/email/template")
	require.NoError(t, err)

	err = r.walk()
	require.NoError(t, err)

	err = r.persistToDisk()
	require.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(tmpDir, "my_email"))
	require.NoError(t, err)
	assert.Equal(t, "shreyas.goenka@databricks.com\n", string(b))
}

func TestRendererExecuteTemplate(t *testing.T) {
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

func TestRendererIsSkipped(t *testing.T) {
	r := renderer{
		skipPatterns: []string{"a*", "*yz", "def", "a/b/*"},
	}

	// skipped paths
	isSkipped, err := r.isSkipped("abc")
	require.NoError(t, err)
	assert.True(t, isSkipped)

	isSkipped, err = r.isSkipped("abcd")
	require.NoError(t, err)
	assert.True(t, isSkipped)

	isSkipped, err = r.isSkipped("a")
	require.NoError(t, err)
	assert.True(t, isSkipped)

	isSkipped, err = r.isSkipped("xxyz")
	require.NoError(t, err)
	assert.True(t, isSkipped)

	isSkipped, err = r.isSkipped("yz")
	require.NoError(t, err)
	assert.True(t, isSkipped)

	isSkipped, err = r.isSkipped("a/b/c")
	require.NoError(t, err)
	assert.True(t, isSkipped)

	// NOT skipped paths
	isSkipped, err = r.isSkipped(".")
	require.NoError(t, err)
	assert.False(t, isSkipped)

	isSkipped, err = r.isSkipped("y")
	require.NoError(t, err)
	assert.False(t, isSkipped)

	isSkipped, err = r.isSkipped("z")
	require.NoError(t, err)
	assert.False(t, isSkipped)

	isSkipped, err = r.isSkipped("defg")
	require.NoError(t, err)
	assert.False(t, isSkipped)

	isSkipped, err = r.isSkipped("cat")
	require.NoError(t, err)
	assert.False(t, isSkipped)

	isSkipped, err = r.isSkipped("a/b/c/d")
	require.NoError(t, err)
	assert.False(t, isSkipped)
}

// TODO: have a test that directories matching glob patterns are skipped, and not generated in the first place
// TODO: make glob patterns work for windows too. PR test runner should be enough to test this

func TestRendererPersistToDisk(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	instanceFiler, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)

	r := &renderer{
		ctx:           ctx,
		instanceFiler: instanceFiler,
		skipPatterns:  []string{"a/b/c", "mn*"},
		files: []*inMemoryFile{
			{
				path:    "a/b/c",
				content: nil,
			},
			{
				path:    "mno",
				content: nil,
			},
			{
				path:    "a/b/d",
				content: []byte("123"),
			},
			{
				path:    "mmnn",
				content: []byte("456"),
			},
		},
	}

	err = r.persistToDisk()
	require.NoError(t, err)

	assert.NoFileExists(t, filepath.Join(tmpDir, "a", "b", "c"))
	assert.NoFileExists(t, filepath.Join(tmpDir, "mno"))
	assertFileContent(t, filepath.Join(tmpDir, "a", "b", "d"), "123")
	assertFileContent(t, filepath.Join(tmpDir, "mmnn"), "456")
}

// func TestGenerateFile(t *testing.T) {
// 	tmp := t.TempDir()

// 	pathTemplate := filepath.Join(tmp, "{{.Animal}}", "{{.Material}}", "foo", "{{.count}}.txt")
// 	contentTemplate := `"{{.count}} items are made of {{.Material}}".
// 	{{if eq .Animal "sheep" }}
// 	Sheep wool is the best!
// 	{{else}}
// 	{{.Animal}} wool is not too bad...
// 	{{end}}
// 	`

// 	r := renderer{
// 		config: map[string]any{
// 			"Material": "wool",
// 			"count":    1,
// 			"Animal":   "cat",
// 		},
// 		baseTemplate: template.New("base"),
// 	}
// 	f, err := r.generateFile(pathTemplate, contentTemplate, 0444)
// 	require.NoError(t, err)

// 	// assert file content
// 	assert.Equal(t, "\"1 items are made of wool\".\n\t\n\tcat wool is not too bad...\n\t\n\t", f.content)

// 	// assert file permissions are correctly assigned
// 	assert.Equal(t, fs.FileMode(0444), f.perm)

// 	// assert file path
// 	assert.Equal(t, filepath.Join(tmp, "cat", "wool", "foo", "1.txt"), f.relPath)
// }

// func TestDeleteSkippedFiles(t *testing.T) {
// 	tmpDir := t.TempDir()
// 	inputFiles := map[*inMemoryFile]any{
// 		{
// 			relPath: filepath.Join(tmpDir, "aaa"),
// 			content: "one",
// 			perm:    0444,
// 		}: nil,
// 		{
// 			relPath: filepath.Join(tmpDir, "abb"),
// 			content: "two",
// 			perm:    0444,
// 		}: nil,
// 		{
// 			relPath: filepath.Join(tmpDir, "bbb"),
// 			content: "three",
// 			perm:    0666,
// 		}: nil,
// 	}

// 	err := deleteSkippedFiles(inputFiles, []string{"aaa", "abb"})
// 	require.NoError(t, err)

// 	assert.Len(t, inputFiles, 1)
// 	for v := range inputFiles {
// 		assert.Equal(t, inMemoryFile{
// 			relPath: filepath.Join(tmpDir, "bbb"),
// 			content: "three",
// 			perm:    0666,
// 		}, *v)
// 	}
// }

// func TestDeleteSkippedFilesWithGlobPatterns(t *testing.T) {
// 	tmpDir := t.TempDir()
// 	inputFiles := map[*inMemoryFile]any{
// 		{
// 			relPath: filepath.Join(tmpDir, "aaa"),
// 			content: "one",
// 			perm:    0444,
// 		}: nil,
// 		{
// 			relPath: filepath.Join(tmpDir, "abb"),
// 			content: "two",
// 			perm:    0444,
// 		}: nil,
// 		{
// 			relPath: filepath.Join(tmpDir, "bbb"),
// 			content: "three",
// 			perm:    0666,
// 		}: nil,
// 		{
// 			relPath: filepath.Join(tmpDir, "ddd"),
// 			content: "four",
// 			perm:    0666,
// 		}: nil,
// 	}

// 	err := deleteSkippedFiles(inputFiles, []string{"a*"})
// 	require.NoError(t, err)

// 	files := make([]inMemoryFile, 0)
// 	for v := range inputFiles {
// 		files = append(files, *v)
// 	}
// 	assert.Len(t, files, 2)
// 	assert.Contains(t, files, inMemoryFile{
// 		relPath: filepath.Join(tmpDir, "bbb"),
// 		content: "three",
// 		perm:    0666,
// 	})
// 	assert.Contains(t, files, inMemoryFile{
// 		relPath: filepath.Join(tmpDir, "ddd"),
// 		content: "four",
// 		perm:    0666,
// 	})
// }

// func TestSkipAllFiles(t *testing.T) {
// 	tmpDir := t.TempDir()
// 	inputFiles := map[*inMemoryFile]any{
// 		{
// 			relPath: filepath.Join(tmpDir, "aaa"),
// 			content: "one",
// 			perm:    0444,
// 		}: nil,
// 		{
// 			relPath: filepath.Join(tmpDir, "abb"),
// 			content: "two",
// 			perm:    0444,
// 		}: nil,
// 		{
// 			relPath: filepath.Join(tmpDir, "bbb"),
// 			content: "three",
// 			perm:    0666,
// 		}: nil,
// 	}

// 	err := deleteSkippedFiles(inputFiles, []string{"*"})
// 	require.NoError(t, err)
// 	assert.Len(t, inputFiles, 0)
// }

// func TestTemplateMaterializeFiles(t *testing.T) {
// 	tmpDir := t.TempDir()
// 	inputFiles := map[*inMemoryFile]any{
// 		{
// 			relPath: filepath.Join(tmpDir, "aaa"),
// 			content: "one",
// 			perm:    0444,
// 		}: nil,
// 		{
// 			relPath: filepath.Join(tmpDir, "abb"),
// 			content: "two",
// 			perm:    0444,
// 		}: nil,
// 		{
// 			relPath: filepath.Join(tmpDir, "bbb"),
// 			content: "three",
// 			perm:    0666,
// 		}: nil,
// 	}

// 	err := materializeFiles(inputFiles)
// 	assert.NoError(t, err)

// 	path := filepath.Join(tmpDir, "aaa")
// 	assertFilePerm(t, path, 0444)
// 	assertFileContent(t, path, "one")

// 	path = filepath.Join(tmpDir, "abb")
// 	assertFilePerm(t, path, 0444)
// 	assertFileContent(t, path, "two")

// 	path = filepath.Join(tmpDir, "bbb")
// 	assertFilePerm(t, path, 0666)
// 	assertFileContent(t, path, "three")
// }

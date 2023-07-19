package template

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
	assert.Equal(t, "shreyas.goenka@databricks.com", strings.Trim(string(b), "\n\r"))
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
// TODO: add test for skip all files from current directory
// TODO: add test for "fail" method
// TODO: test for skip patterns being relatively evaluated

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

func TestRendererWalk(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	r, err := newRenderer(ctx, nil, "./testdata/walk/library", tmpDir, "./testdata/walk/template")
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	getContent := func(r *renderer, path string) string {
		for _, f := range r.files {
			if f.path == path {
				return strings.Trim(string(f.content), "\r\n")
			}
		}
		require.FailNow(t, "file is absent: "+path)
		return ""
	}

	assert.Len(t, r.files, 4)
	assert.Equal(t, "file one", getContent(r, "file1"))
	assert.Equal(t, "file two", getContent(r, "file2"))
	assert.Equal(t, "file three", getContent(r, "dir1/dir3/file3"))
	assert.Equal(t, "file four", getContent(r, "dir2/file4"))
}

func TestRendererFailFunction(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	r, err := newRenderer(ctx, nil, "./testdata/fail/library", tmpDir, "./testdata/fail/template")
	require.NoError(t, err)

	err = r.walk()
	assert.Equal(t, "I am a error message", err.Error())
}

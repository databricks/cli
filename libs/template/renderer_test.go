package template

import (
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
	err := r.generateFile(pathTemplate, contentTemplate, 0444)
	require.NoError(t, err)

	// assert file exists
	assert.FileExists(t, filepath.Join(tmp, "cat", "wool", "foo", "1.txt"))

	// assert file content is created correctly
	b, err := os.ReadFile(filepath.Join(tmp, "cat", "wool", "foo", "1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "\"1 items are made of wool\".\n\t\n\tcat wool is not too bad...\n\t\n\t", string(b))

	// assert file permissions are correctly assigned
	stat, err := os.Stat(filepath.Join(tmp, "cat", "wool", "foo", "1.txt"))
	require.NoError(t, err)
	assert.Equal(t, uint(0444), uint(stat.Mode().Perm()))
}

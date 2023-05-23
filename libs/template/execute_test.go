package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteTemplate(t *testing.T) {
	templateText :=
		`"{{.count}} items are made of {{.Material}}".
{{if eq .Animal "sheep" }}
Sheep wool is the best!
{{else}}
{{.Animal}} wool is not too bad...
{{end}}
`
	statement, err := executeTemplate(map[string]any{
		"Material": "wool",
		"count":    1,
		"Animal":   "sheep",
	}, templateText)
	require.NoError(t, err)
	assert.Equal(t, "\"1 items are made of wool\".\n\nSheep wool is the best!\n\n", statement)

	statement, err = executeTemplate(map[string]any{
		"Material": "wool",
		"count":    1,
		"Animal":   "cat",
	}, templateText)
	require.NoError(t, err)
	assert.Equal(t, "\"1 items are made of wool\".\n\ncat wool is not too bad...\n\n", statement)
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
	err := generateFile(map[string]any{
		"Material": "wool",
		"count":    1,
		"Animal":   "cat",
	}, pathTemplate, contentTemplate, 0444)
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

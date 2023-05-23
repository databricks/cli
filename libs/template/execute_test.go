package template

import (
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

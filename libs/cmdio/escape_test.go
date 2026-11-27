package cmdio

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type escapeTestRow struct {
	ID    int64
	Name  string
	Tags  map[string]string
	inner string
}

func (r escapeTestRow) Inner() string { return r.inner }

func TestExecuteTemplateEscapesControlCharacters(t *testing.T) {
	// Colors on, so that the template's own colors are in the output.
	ctx := InContext(t.Context(), NewTestIO(nil, &bytes.Buffer{}, &bytes.Buffer{}))
	require.True(t, colorEnabled(ctx))

	tmpl, err := template.New("t").Funcs(renderFuncMap(ctx)).Parse(
		`{{range .}}{{green "%d" .ID}}	{{.Name | bold}}	{{.Inner}}{{range $k, $v := .Tags}}	{{$k}}={{$v}}{{end}}	{{.Name | b64_encode | b64_decode}}
{{end}}`)
	require.NoError(t, err)

	rows := []any{
		escapeTestRow{ID: 1, Name: "a\x1b]0;title\x07b", Tags: map[string]string{"k\r": "v\x1b[2J"}, inner: "in\x1bner"},
		// A value the CLI styled before it got here keeps its colors.
		&escapeTestRow{ID: 2, Name: Green(ctx, "●") + " plain\r\n"},
	}
	var out bytes.Buffer
	require.NoError(t, executeTemplate(tmpl, &out, rows, colorEnabled(ctx)))

	assert.Equal(t,
		"\x1b[32m1\x1b[0m\t\x1b[1m"+`a\u001b]0;title\u0007b`+"\x1b[0m\t"+`in\u001bner`+"\t"+`k\r=v\u001b[2J`+"\t"+`a\u001b]0;title\u0007b`+"\n"+
			"\x1b[32m2\x1b[0m\t\x1b[1m\x1b[32m●\x1b[0m plain\n\x1b[0m\t\t\x1b[32m●\x1b[0m plain\n\n",
		out.String())
	// The data is not modified.
	assert.Equal(t, "a\x1b]0;title\x07b", rows[0].(escapeTestRow).Name)
}

func TestExecuteTemplateKeepsNoColorsWhenColorsAreOff(t *testing.T) {
	ctx, _ := NewTestContextWithStdout(t.Context())
	require.False(t, colorEnabled(ctx))

	tmpl, err := template.New("t").Funcs(renderFuncMap(ctx)).Parse("{{green `%d` .ID}}\t{{.Name}}\t{{.Inner}}\n")
	require.NoError(t, err)

	// With colors off the template adds none, so a color in a value is the value's.
	var out bytes.Buffer
	require.NoError(t, executeTemplate(tmpl, &out, escapeTestRow{ID: 1, Name: "a\x1b[8mb", inner: "c"}, colorEnabled(ctx)))
	assert.Equal(t, "1\t"+`a\u001b[8mb`+"\tc\n", out.String())
}

func TestExecuteTemplateEndsAStyleLeftOnAtItsCell(t *testing.T) {
	tmpl, err := template.New("t").Parse("{{.Name}}\t{{.Inner}}\n")
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, executeTemplate(tmpl, &out, escapeTestRow{Name: "a\x1b[8mb", inner: "c"}, true))
	assert.Equal(t, "a\x1b[8mb\x1b[0m\tc\n", out.String())
}

func TestExecuteTemplateWritesOutputRenderedBeforeAnError(t *testing.T) {
	tmpl, err := template.New("t").Funcs(staticTemplateFuncs).Parse("before\r{{`not base64` | b64_decode}}after")
	require.NoError(t, err)

	var out bytes.Buffer
	err = executeTemplate(tmpl, &out, nil, false)
	require.Error(t, err)
	assert.Equal(t, `before\r`, out.String())
}

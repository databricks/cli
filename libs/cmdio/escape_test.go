package cmdio

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/listing"
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
			"\x1b[32m2\x1b[0m\t\x1b[1m\x1b[32m●\x1b[0m plain\x1b[0m\n\t\t\x1b[32m●\x1b[0m plain\n\n",
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

// Every renderer decides by itself whether colors are kept, so each is run
// with colors on and off: the template's own color and a color in a value are
// both kept with colors on, and with colors off the value's color is escaped.
func TestRenderersKeepColorsOnlyWhenColorsAreOn(t *testing.T) {
	const name = "a\x1b[8mb"
	const rowTemplate = "{{green `%d` .ID}} {{.Name}}\n"
	const rowsTemplate = "{{range .}}" + rowTemplate + "{{end}}"

	// 21 rows: the iterator renderer flushes a full batch of 20, then the rest.
	// No ID is the end of another one, so that a line is found only where it is.
	var rows []escapeTestRow
	var ids []string
	for id := 101; id <= 121; id++ {
		rows = append(rows, escapeTestRow{ID: int64(id), Name: name})
		ids = append(ids, strconv.Itoa(id))
	}
	iterator := func() listing.Iterator[escapeTestRow] {
		it := listing.SliceIterator[escapeTestRow](rows)
		return &it
	}

	renderers := []struct {
		name     string
		template string
		render   func(ctx context.Context, stdout, stderr *bytes.Buffer) error
		// Each line starts with text the CLI colors itself, then has the name.
		ownText   []string
		ownColor  string
		separator string
	}{
		{
			name:     "Render",
			template: rowTemplate,
			render: func(ctx context.Context, stdout, stderr *bytes.Buffer) error {
				return Render(ctx, rows[0])
			},
			ownText: ids[:1], ownColor: ansiGreen, separator: " ",
		},
		{
			name:     "RenderIterator",
			template: rowsTemplate,
			render: func(ctx context.Context, stdout, stderr *bytes.Buffer) error {
				return RenderIterator(ctx, iterator())
			},
			ownText: ids, ownColor: ansiGreen, separator: " ",
		},
		{
			name: "paged iterator",
			render: func(ctx context.Context, stdout, stderr *bytes.Buffer) error {
				return renderIteratorPagedTemplateCore(ctx, iterator(), strings.NewReader("\r"), stdout, "", rowsTemplate, 5)
			},
			ownText: ids, ownColor: ansiGreen, separator: " ",
		},
		{
			name: "RenderDiagnostics",
			render: func(ctx context.Context, stdout, stderr *bytes.Buffer) error {
				err := RenderDiagnostics(ctx, diag.Diagnostics{{Severity: diag.Error, Summary: name}})
				stdout.Write(stderr.Bytes())
				return err
			},
			ownText: []string{"Error"}, ownColor: ansiRed, separator: ": ",
		},
	}

	for _, r := range renderers {
		for _, colors := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/colors=%v", r.name, colors), func(t *testing.T) {
				// Only stdout is a terminal, so RenderIterator does not page.
				var stdout, stderr bytes.Buffer
				ctx := InContext(t.Context(), &cmdIO{
					capabilities: Capabilities{stdoutIsTTY: true, noColor: !colors},
					outputFormat: flags.OutputText,
					template:     r.template,
					out:          &stdout,
					err:          &stderr,
				})
				require.Equal(t, colors, colorEnabled(ctx))
				require.NoError(t, r.render(ctx, &stdout, &stderr))

				// The pager ends its lines with CRLF.
				out := strings.ReplaceAll(stdout.String(), "\r\n", "\n")
				for _, own := range r.ownText {
					if colors {
						// The style the value turns on ends with its line.
						assert.Contains(t, out, r.ownColor+own+ansiReset+r.separator+name+ansiReset+"\n")
					} else {
						assert.Contains(t, out, own+r.separator+`a\u001b[8mb`+"\n")
					}
				}
				// The value's own color is in every line with colors on, and in none with colors off.
				raw := 0
				if colors {
					raw = len(r.ownText)
				}
				assert.Equal(t, raw, strings.Count(out, name))
			})
		}
	}
}

func TestLogStringKeepsColorsWhenEitherStreamHasColors(t *testing.T) {
	// The color helpers follow stdout, so a line they colored can go to a
	// stderr that is not a terminal.
	const line = "\x1b[32mdone\x1b[0m"
	cases := []struct {
		name         string
		capabilities Capabilities
		expected     string
	}{
		{"stdout", Capabilities{stdoutIsTTY: true}, line + "\n"},
		{"stderr", Capabilities{stderrIsTTY: true}, line + "\n"},
		{"neither", Capabilities{}, `\u001b[32mdone\u001b[0m` + "\n"},
		{"NO_COLOR", Capabilities{stdoutIsTTY: true, stderrIsTTY: true, noColor: true}, `\u001b[32mdone\u001b[0m` + "\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stderr bytes.Buffer
			ctx := InContext(t.Context(), &cmdIO{capabilities: c.capabilities, err: &stderr})
			LogString(ctx, line)
			assert.Equal(t, c.expected, stderr.String())
		})
	}
}

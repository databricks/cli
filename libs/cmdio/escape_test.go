package cmdio

import (
	"bytes"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type escapeTestState string

type escapeTestInner struct {
	Name    string
	private string
}

type escapeTestOuter struct {
	ID      int64
	Name    string
	State   escapeTestState
	Inner   *escapeTestInner
	Nil     *escapeTestInner
	Tags    map[string]string
	Items   []escapeTestInner
	Array   [2]string
	Any     any
	Created time.Time
	Bytes   []byte
}

func TestEscapeControlInData(t *testing.T) {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	original := escapeTestOuter{
		ID:      1,
		Name:    "a\x1b[2Jb",
		State:   "RUN\rNING",
		Inner:   &escapeTestInner{Name: "in\x07ner", private: "pri\x1bvate"},
		Tags:    map[string]string{"k\x1b": "v\x1b", "plain": "plain"},
		Items:   []escapeTestInner{{Name: "plain"}, {Name: "it\x1bem"}},
		Array:   [2]string{"plain", "arr\x1b"},
		Any:     []any{"any\x1b", 2},
		Created: created,
		Bytes:   []byte("by\x1btes"),
	}
	inner := original.Inner

	got := escapeControlInData(&original).(*escapeTestOuter)

	assert.Equal(t, &escapeTestOuter{
		ID:      1,
		Name:    `a\x1b[2Jb`,
		State:   `RUN\rNING`,
		Inner:   &escapeTestInner{Name: `in\aner`, private: "pri\x1bvate"},
		Tags:    map[string]string{`k\x1b`: `v\x1b`, "plain": "plain"},
		Items:   []escapeTestInner{{Name: "plain"}, {Name: `it\x1bem`}},
		Array:   [2]string{"plain", `arr\x1b`},
		Any:     []any{`any\x1b`, 2},
		Created: created,
		Bytes:   []byte("by\x1btes"),
	}, got)

	// The input is not modified.
	assert.Equal(t, "a\x1b[2Jb", original.Name)
	assert.Same(t, inner, original.Inner)
	assert.Equal(t, "in\x07ner", inner.Name)
	assert.Equal(t, map[string]string{"k\x1b": "v\x1b", "plain": "plain"}, original.Tags)
	assert.Equal(t, "it\x1bem", original.Items[1].Name)
	assert.Equal(t, []any{"any\x1b", 2}, original.Any)
}

func TestEscapeControlInDataReturnsCleanDataAsIs(t *testing.T) {
	clean := &escapeTestOuter{
		Name:  "multi\nline\ttext",
		Inner: &escapeTestInner{Name: "inner"},
		Tags:  map[string]string{"k": "v"},
		Items: []escapeTestInner{{Name: "item"}},
		Any:   []any{"any"},
	}
	assert.Same(t, clean, escapeControlInData(clean))

	items := []any{clean, "plain", nil}
	assert.Equal(t, &items[0], &escapeControlInData(items).([]any)[0])

	assert.Nil(t, escapeControlInData(nil))
	assert.Equal(t, 5, escapeControlInData(5))
}

func TestExecuteTemplateEscapesData(t *testing.T) {
	ctx, _ := NewTestContextWithStdout(t.Context())
	tmpl, err := template.New("t").Funcs(renderFuncMap(ctx)).Parse(`{{range .}}{{.ID | printf "%d" | green}}	{{.Name | bold}}	{{.State}}
{{end}}`)
	require.NoError(t, err)

	var out bytes.Buffer
	err = executeTemplate(tmpl, &out, []any{
		&escapeTestOuter{ID: 1, Name: "a\x1b]0;title\x07b", State: "ok\r"},
	})
	require.NoError(t, err)
	assert.Equal(t, "1\t"+`a\x1b]0;title\ab`+"\t"+`ok\r`+"\n", out.String())
}

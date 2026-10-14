package cmdio

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type templateDataInner struct {
	Name *string
}

type templateDataBase struct {
	Base *string
}

type templateDataOuter struct {
	templateDataBase
	Flag   *bool
	Count  *int64
	Unset  *string
	Inner  *templateDataInner
	Nested []templateDataInner
	ByKey  map[string]templateDataInner
	Plain  string
}

func renderTemplateData(t *testing.T, tmpl string, data any) string {
	t.Helper()
	tt := template.Must(template.New("t").Funcs(template.FuncMap{
		"plus1": func(v int64) int64 { return v + 1 },
	}).Parse(tmpl))
	var buf bytes.Buffer
	require.NoError(t, tt.Execute(&buf, templateData(data)))
	return buf.String()
}

func TestTemplateDataDereferencesScalars(t *testing.T) {
	data := templateDataOuter{
		templateDataBase: templateDataBase{Base: new("base")},
		Flag:             new(false),
		Count:            new(int64(41)),
		Inner:            &templateDataInner{Name: new("in")},
		Nested:           []templateDataInner{{Name: new("a")}, {}},
		ByKey:            map[string]templateDataInner{"k": {Name: new("v")}},
		Plain:            "p",
	}
	got := renderTemplateData(t, `{{if .Flag}}T{{else}}F{{end}} {{plus1 .Count}} [{{.Unset}}] {{.Inner.Name}} {{range .Nested}}<{{.Name}}>{{end}} {{(index .ByKey "k").Name}} {{.Plain}} {{.Base}}`, data)
	assert.Equal(t, "F 42 [] in <a><> v p base", got)
}

func TestTemplateDataNilStructPointerIsFalsy(t *testing.T) {
	got := renderTemplateData(t, `{{if .Inner}}set{{else}}unset{{end}}`, &templateDataOuter{})
	assert.Equal(t, "unset", got)
}

type templateDataWithMethod struct {
	Name string
}

func (v templateDataWithMethod) Upper() string { return "UP-" + v.Name }

func TestTemplateDataKeepsTypesWithoutPointers(t *testing.T) {
	got := renderTemplateData(t, `{{.Upper}}`, templateDataWithMethod{Name: "x"})
	assert.Equal(t, "UP-x", got)
}

package main

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templates embed.FS

// ErrSkipThisFile is panicked by the skipThisFile template helper and recovered
// by callers to suppress generation of a file. Mirrors genkit's render package.
var ErrSkipThisFile = errors.New("skip generating this file")

// helperFuncs holds the template helpers actually referenced by the cliv0
// templates (templates/*.tmpl). It derives from genkit's render.HelperFuncs but
// drops every helper no template uses, so the set must stay in sync with the
// templates: an unreferenced helper here is dead, and a referenced-but-missing
// one fails rendering loudly.
var helperFuncs = template.FuncMap{
	"lower": strings.ToLower,
	"trimSuffix": func(right, left string) string {
		return strings.TrimSuffix(left, right)
	},
	"without": func(left, right string) string {
		return strings.ReplaceAll(right, left, "")
	},
	"skipThisFile": func() error {
		// The error is rendered as a string in the resulting file, so we must
		// panic and let callers handle it via errors.Is(err, ErrSkipThisFile).
		panic(ErrSkipThisFile)
	},
	"list": func(l ...any) []any {
		return l
	},
	"in": func(haystack []any, needle string) bool {
		for _, v := range haystack {
			if needle == fmt.Sprint(v) {
				return true
			}
		}
		return false
	},
	"dict": func(args ...any) map[string]any {
		if len(args)%2 != 0 {
			panic("number of arguments to dict is not even")
		}
		result := map[string]any{}
		for i := 0; i < len(args); i += 2 {
			k := fmt.Sprint(args[i])
			v := args[i+1]
			result[k] = v
		}
		return result
	},
}

func parseTemplate(name, path string) *template.Template {
	t := template.New(name).Funcs(helperFuncs)
	return template.Must(t.ParseFS(templates, path))
}

// renderToFile renders the named template with data to fileName. It mirrors
// genkit's render.RenderToFile: a skipThisFile panic surfaces as an error that
// callers match with errors.Is(err, ErrSkipThisFile).
func renderToFile(data any, t *template.Template, templateName, fileName string) error {
	sb := strings.Builder{}
	if err := t.ExecuteTemplate(&sb, templateName, data); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fileName), 0o755); err != nil {
		return err
	}
	return os.WriteFile(fileName, []byte(sb.String()), 0o644)
}

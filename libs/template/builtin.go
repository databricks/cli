package template

import (
	"embed"
	"io/fs"
)

//go:embed all:templates
var builtinTemplates embed.FS

// BuiltinTemplate represents a template that is built into the CLI.
type BuiltinTemplate struct {
	Name string
	FS   fs.FS
}

// Builtin returns the list of all built-in templates.
// TODO: Make private?
func Builtin() ([]BuiltinTemplate, error) {
	templates, err := fs.Sub(builtinTemplates, "templates")
	if err != nil {
		return nil, err
	}

	entries, err := fs.ReadDir(templates, ".")
	if err != nil {
		return nil, err
	}

	var out []BuiltinTemplate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		templateFS, err := fs.Sub(templates, entry.Name())
		if err != nil {
			return nil, err
		}

		out = append(out, BuiltinTemplate{
			Name: entry.Name(),
			FS:   templateFS,
		})
	}

	return out, nil
}

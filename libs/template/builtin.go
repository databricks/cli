package template

import (
	"embed"
	"io/fs"
)

//go:embed all:templates
var builtinTemplates embed.FS

// builtinTemplate represents a template that is built into the CLI.
type builtinTemplate struct {
	Name string
	FS   fs.FS
}

// builtin returns the list of all built-in templates.
func builtin() ([]builtinTemplate, error) {
	templates, err := fs.Sub(builtinTemplates, "templates")
	if err != nil {
		return nil, err
	}

	entries, err := fs.ReadDir(templates, ".")
	if err != nil {
		return nil, err
	}

	var out []builtinTemplate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		templateFS, err := fs.Sub(templates, entry.Name())
		if err != nil {
			return nil, err
		}

		out = append(out, builtinTemplate{
			Name: entry.Name(),
			FS:   templateFS,
		})
	}

	return out, nil
}

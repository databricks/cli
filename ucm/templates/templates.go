// Package templates embeds the ucm init starter templates and exposes them via
// the shared libs/template Reader interface so cmd/ucm/init can dispatch on a
// name the same way cmd/bundle/init dispatches on databricks templates.
package templates

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/databricks/cli/libs/jsonschema"
	"github.com/databricks/cli/libs/template"
)

//go:embed all:default all:brownfield all:multienv
var builtinFS embed.FS

const schemaFileName = "databricks_template_schema.json"

// Name identifies a built-in ucm template.
type Name string

const (
	Default    Name = "default"
	Brownfield Name = "brownfield"
	Multienv   Name = "multienv"
)

type builtin struct {
	name        Name
	description string
}

var builtins = []builtin{
	{Default, "Minimal ucm.yml with one catalog, one schema, one grant"},
	{Brownfield, "Stub ucm.yml plus instructions for 'ucm generate' to populate"},
	{Multienv, "dev/staging/prod targets with shared includes"},
}

// List returns the built-in ucm templates with their descriptions.
func List() []struct {
	Name        string
	Description string
} {
	out := make([]struct {
		Name        string
		Description string
	}, 0, len(builtins))
	for _, b := range builtins {
		out = append(out, struct {
			Name        string
			Description string
		}{string(b.name), b.description})
	}
	return out
}

// HelpDescriptions returns the bullet list of templates used in --help text.
func HelpDescriptions() string {
	out := ""
	for i, b := range builtins {
		if i > 0 {
			out += "\n"
		}
		out += fmt.Sprintf("- %s: %s", b.name, b.description)
	}
	return out
}

// Lookup returns a Reader for the built-in template with the given name, or
// nil if name does not match any built-in.
func Lookup(name string) template.Reader {
	for _, b := range builtins {
		if string(b.name) == name {
			return &reader{name: string(b.name)}
		}
	}
	return nil
}

// reader adapts the embedded FS to template.Reader.
type reader struct {
	name string
}

func (r *reader) LoadSchemaAndTemplateFS(_ context.Context) (*jsonschema.Schema, fs.FS, error) {
	templateFS, err := fs.Sub(builtinFS, r.name)
	if err != nil {
		return nil, nil, err
	}
	schema, err := jsonschema.LoadFS(templateFS, schemaFileName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("not a ucm template: expected to find a template schema file at %s", schemaFileName)
		}
		return nil, nil, fmt.Errorf("failed to load schema for template %s: %w", r.name, err)
	}
	return schema, templateFS, nil
}

func (r *reader) SchemaFS(_ context.Context) (fs.FS, error) {
	return fs.Sub(builtinFS, r.name)
}

func (r *reader) Cleanup(_ context.Context) {}

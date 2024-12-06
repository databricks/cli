package main

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/ghodss/yaml"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/jsonschema"
)

type annotation struct {
	Description         string `json:"description,omitempty"`
	MarkdownDescription string `json:"markdown_description,omitempty"`
	Title               string `json:"title,omitempty"`
	Default             any    `json:"default,omitempty"`
	Enum                []any  `json:"enum,omitempty"`
}

type annotationHandler struct {
	ref   map[string]annotation
	empty map[string]annotation
}

const Placeholder = "PLACEHOLDER"

// Adds annotations to the JSON schema reading from the annotation files.
// More details https://json-schema.org/understanding-json-schema/reference/annotations
func newAnnotationHandler(sources []string) (*annotationHandler, error) {
	prev := dyn.NilValue
	for _, path := range sources {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		generated, err := yamlloader.LoadYAML(path, bytes.NewBuffer(b))
		if err != nil {
			return nil, err
		}
		prev, err = merge.Merge(prev, generated)
		if err != nil {
			return nil, err
		}
	}

	var data map[string]annotation

	err := convert.ToTyped(&data, prev)
	if err != nil {
		return nil, err
	}

	d := &annotationHandler{}
	d.ref = data
	d.empty = map[string]annotation{}
	return d, nil
}

func (d *annotationHandler) addAnnotations(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	refPath := jsonschema.TypePath(typ)
	items := map[string]*jsonschema.Schema{}
	items[refPath] = &s

	for k, v := range s.Properties {
		itemRefPath := fmt.Sprintf("%s.%s", refPath, k)
		items[itemRefPath] = v
	}

	for k, v := range items {
		// Skipping all default types like int, string, etc.
		shouldHandle := strings.HasPrefix(refPath, "github.com")
		if !shouldHandle {
			continue
		}

		item, ok := d.ref[k]
		if !ok {
			item = annotation{}
		}
		if item.Description == "" {
			item.Description = Placeholder
			d.empty[k] = item
		}
		if item.Description != Placeholder {
			v.Description = item.Description
		}

		if item.Default != nil {
			v.Default = item.Default
		}
		v.MarkdownDescription = item.MarkdownDescription
		v.Title = item.Title
		v.Enum = item.Enum
	}
	return s
}

// Adds empty annotations with placeholders to the annotation file
func (d *annotationHandler) sync(outputPath string) error {
	file, err := os.ReadFile(outputPath)
	if err != nil {
		return err
	}

	existing := map[string]annotation{}
	err = yaml.Unmarshal(file, &existing)

	for k, v := range d.empty {
		existing[k] = v
	}
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(existing)
	if err != nil {
		return err
	}
	err = os.WriteFile(outputPath, b, 0644)
	if err != nil {
		return err
	}
	return nil
}

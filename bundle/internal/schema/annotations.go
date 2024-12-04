package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/ghodss/yaml"

	"github.com/databricks/cli/libs/jsonschema"
)

type annotation struct {
	Description         string `json:"description,omitempty"`
	MarkdownDescription string `json:"markdown_description,omitempty"`
	Title               string `json:"title,omitempty"`
	Default             any    `json:"default,omitempty"`
}

type annotationHandler struct {
	filePath string
	op       *openapiParser
	ref      map[string]annotation
}

const Placeholder = "PLACEHOLDER"

// Adds annotations to the JSON schema reading from the annotations file.
// More details https://json-schema.org/understanding-json-schema/reference/annotations
func newAnnotationHandler(path string, op *openapiParser) (*annotationHandler, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	data := map[string]annotation{}
	err = yaml.Unmarshal(b, &data)
	if err != nil {
		return nil, err
	}

	if data == nil {
		data = map[string]annotation{}
	}

	d := &annotationHandler{}
	d.ref = data
	d.op = op
	d.filePath = path
	return d, nil
}

func (d *annotationHandler) addAnnotations(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	// Skips the type if it is a part of OpenAPI spec, as it is already annotated.
	_, ok := d.op.findRef(typ)
	if ok {
		return s
	}

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
			d.ref[k] = annotation{
				Description: Placeholder,
			}
			continue
		}
		if item.Description == Placeholder {
			continue
		}

		v.Description = item.Description
		if item.Default != nil {
			v.Default = item.Default
		}
		v.MarkdownDescription = item.MarkdownDescription
		v.Title = item.Title
	}
	return s
}

func (d *annotationHandler) overwrite() error {
	data, err := yaml.Marshal(d.ref)
	if err != nil {
		return err
	}

	err = os.WriteFile(d.filePath, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

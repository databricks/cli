package main

import (
	"bytes"
	"os"
	"reflect"
	"strings"

	"github.com/ghodss/yaml"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
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
	ref   annotationFile
	empty annotationFile
}

type annotationFile map[string]map[string]annotation

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

	var data annotationFile

	err := convert.ToTyped(&data, prev)
	if err != nil {
		return nil, err
	}

	d := &annotationHandler{}
	d.ref = data
	d.empty = annotationFile{}
	return d, nil
}

func (d *annotationHandler) addAnnotations(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	refPath := getPath(typ)
	shouldHandle := strings.HasPrefix(refPath, "github.com")
	if !shouldHandle {
		return s
	}

	annotations := d.ref[refPath]
	if annotations == nil {
		annotations = map[string]annotation{}
	}

	rootTypeAnnotation, ok := annotations[RootTypeKey]
	if ok {
		assingAnnotation(&s, rootTypeAnnotation)
	}

	for k, v := range s.Properties {
		item, ok := annotations[k]
		if !ok {
			item = annotation{}
		}
		if item.Description == "" {
			item.Description = Placeholder

			emptyAnnotations := d.empty[refPath]
			if emptyAnnotations == nil {
				emptyAnnotations = map[string]annotation{}
				d.empty[refPath] = emptyAnnotations
			}
			emptyAnnotations[k] = item
		}
		assingAnnotation(v, item)
	}
	return s
}

// Adds empty annotations with placeholders to the annotation file
func (d *annotationHandler) sync(outputPath string) error {
	file, err := os.ReadFile(outputPath)
	if err != nil {
		return err
	}

	existing, err := yamlloader.LoadYAML(outputPath, bytes.NewBuffer(file))
	if err != nil {
		return err
	}
	emptyB, err := yaml.Marshal(d.empty)
	if err != nil {
		return err
	}

	empty, err := yamlloader.LoadYAML("", bytes.NewBuffer(emptyB))
	if err != nil {
		return err
	}
	mergedFile, err := merge.Merge(existing, empty)
	if err != nil {
		return err
	}

	saver := yamlsaver.NewSaver()
	config, _ := mergedFile.AsMap()
	err = saver.SaveAsYAML(config, outputPath, true)
	if err != nil {
		return err
	}
	return nil
}

func getPath(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}

func assingAnnotation(s *jsonschema.Schema, a annotation) {
	if a.Description != Placeholder {
		s.Description = a.Description
	}

	if a.Default != nil {
		s.Default = a.Default
	}
	s.MarkdownDescription = a.MarkdownDescription
	s.Title = a.Title
	s.Enum = a.Enum
}

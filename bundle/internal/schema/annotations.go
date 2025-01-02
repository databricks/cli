package main

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	yaml3 "gopkg.in/yaml.v3"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/jsonschema"
)

type annotationHandler struct {
	// Annotations read from all annotation files including all overrides
	parsedAnnotations annotation.File
	// Missing annotations for fields that are found in config that need to be added to the annotation file
	missingAnnotations annotation.File
}

// Adds annotations to the JSON schema reading from the annotation files.
// More details https://json-schema.org/understanding-json-schema/reference/annotations
func newAnnotationHandler(sources []string) (*annotationHandler, error) {
	data, err := annotation.LoadAndMerge(sources)
	if err != nil {
		return nil, err
	}
	d := &annotationHandler{}
	d.parsedAnnotations = data
	d.missingAnnotations = annotation.File{}
	return d, nil
}

func (d *annotationHandler) addAnnotations(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	refPath := getPath(typ)
	shouldHandle := strings.HasPrefix(refPath, "github.com")
	if !shouldHandle {
		return s
	}

	annotations := d.parsedAnnotations[refPath]
	if annotations == nil {
		annotations = map[string]annotation.Descriptor{}
	}

	rootTypeAnnotation, ok := annotations[RootTypeKey]
	if ok {
		assignAnnotation(&s, rootTypeAnnotation)
	}

	for k, v := range s.Properties {
		item := annotations[k]
		if item.Description == "" {
			item.Description = annotation.Placeholder

			emptyAnnotations := d.missingAnnotations[refPath]
			if emptyAnnotations == nil {
				emptyAnnotations = map[string]annotation.Descriptor{}
				d.missingAnnotations[refPath] = emptyAnnotations
			}
			emptyAnnotations[k] = item
		}
		assignAnnotation(v, item)
	}
	return s
}

// Writes missing annotations with placeholder values back to the annotation file
func (d *annotationHandler) syncWithMissingAnnotations(outputPath string) error {
	existingFile, err := os.ReadFile(outputPath)
	if err != nil {
		return err
	}
	existing, err := yamlloader.LoadYAML("", bytes.NewBuffer(existingFile))
	if err != nil {
		return err
	}
	missingAnnotations, err := convert.FromTyped(&d.missingAnnotations, dyn.NilValue)
	if err != nil {
		return err
	}

	output, err := merge.Merge(existing, missingAnnotations)
	if err != nil {
		return err
	}

	err = saveYamlWithStyle(outputPath, output)
	if err != nil {
		return err
	}
	return nil
}

func getPath(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}

func assignAnnotation(s *jsonschema.Schema, a annotation.Descriptor) {
	if a.Description != annotation.Placeholder {
		s.Description = a.Description
	}

	if a.Default != nil {
		s.Default = a.Default
	}
	s.MarkdownDescription = convertLinksToAbsoluteUrl(a.MarkdownDescription)
	s.Title = a.Title
	s.Enum = a.Enum
}

func saveYamlWithStyle(outputPath string, input dyn.Value) error {
	style := map[string]yaml3.Style{}
	file, _ := input.AsMap()
	for _, v := range file.Keys() {
		style[v.MustString()] = yaml3.LiteralStyle
	}

	saver := yamlsaver.NewSaverWithStyle(style)
	err := saver.SaveAsYAML(file, outputPath, true)
	if err != nil {
		return err
	}
	return nil
}

func convertLinksToAbsoluteUrl(s string) string {
	if s == "" {
		return s
	}
	base := "https://docs.databricks.com"
	referencePage := "/dev-tools/bundles/reference.html"

	// Regular expression to match Markdown-style links like [_](link)
	re := regexp.MustCompile(`\[_\]\(([^)]+)\)`)
	result := re.ReplaceAllStringFunc(s, func(match string) string {
		matches := re.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		link := matches[1]
		var text, absoluteURL string

		if strings.HasPrefix(link, "#") {
			text = strings.TrimPrefix(link, "#")
			absoluteURL = fmt.Sprintf("%s%s%s", base, referencePage, link)

			// Handle relative paths like /dev-tools/bundles/resources.html#dashboard
		} else if strings.HasPrefix(link, "/") {
			absoluteURL = strings.ReplaceAll(fmt.Sprintf("%s%s", base, link), ".md", ".html")
			if strings.Contains(link, "#") {
				parts := strings.Split(link, "#")
				text = parts[1]
			} else {
				text = "link"
			}
		} else {
			return match
		}

		return fmt.Sprintf("[%s](%s)", text, absoluteURL)
	})

	return result
}

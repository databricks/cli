package main

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"

	yaml3 "gopkg.in/yaml.v3"

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
	// Annotations read from all annotation files including all overrides
	parsedAnnotations annotationFile
	// Missing annotations for fields that are found in config that need to be added to the annotation file
	missingAnnotations annotationFile
}

/**
 * Parsed file with annotations, expected format:
 * github.com/databricks/cli/bundle/config.Bundle:
 *  	cluster_id:
 *      description: "Description"
 */
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
	d.parsedAnnotations = data
	d.missingAnnotations = annotationFile{}
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
		annotations = map[string]annotation{}
	}

	rootTypeAnnotation, ok := annotations[RootTypeKey]
	if ok {
		assignAnnotation(&s, rootTypeAnnotation)
	}

	for k, v := range s.Properties {
		item := annotations[k]
		if item.Description == "" {
			item.Description = Placeholder

			emptyAnnotations := d.missingAnnotations[refPath]
			if emptyAnnotations == nil {
				emptyAnnotations = map[string]annotation{}
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

	for k := range d.missingAnnotations {
		if !isCliPath(k) {
			delete(d.missingAnnotations, k)
			fmt.Printf("Missing annotations for `%s` that are not in CLI package, try to fetch latest OpenAPI spec and regenerate annotations", k)
		}
	}

	missingAnnotations, err := convert.FromTyped(d.missingAnnotations, dyn.NilValue)
	if err != nil {
		return err
	}

	output, err := merge.Merge(existing, missingAnnotations)
	if err != nil {
		return err
	}

	var outputTyped annotationFile
	err = convert.ToTyped(&outputTyped, output)
	if err != nil {
		return err
	}

	err = saveYamlWithStyle(outputPath, outputTyped)
	if err != nil {
		return err
	}
	return nil
}

func getPath(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}

func assignAnnotation(s *jsonschema.Schema, a annotation) {
	if a.Description != Placeholder {
		s.Description = a.Description
	}

	if a.Default != nil {
		s.Default = a.Default
	}
	s.MarkdownDescription = convertLinksToAbsoluteUrl(a.MarkdownDescription)
	s.Title = a.Title
	s.Enum = a.Enum
}

func saveYamlWithStyle(outputPath string, annotations annotationFile) error {
	annotationOrder := yamlsaver.NewOrder([]string{"description", "markdown_description", "title", "default", "enum"})
	style := map[string]yaml3.Style{}

	order := getAlphabeticalOrder(annotations)
	dynMap := map[string]dyn.Value{}
	for k, v := range annotations {
		style[k] = yaml3.LiteralStyle

		properties := map[string]dyn.Value{}
		propertiesOrder := getAlphabeticalOrder(v)
		for key, value := range v {
			d, err := convert.FromTyped(value, dyn.NilValue)
			if d.Kind() == dyn.KindNil || err != nil {
				properties[key] = dyn.NewValue(map[string]dyn.Value{}, []dyn.Location{{Line: propertiesOrder.Get(key)}})
				continue
			}
			val, err := yamlsaver.ConvertToMapValue(value, annotationOrder, []string{}, map[string]dyn.Value{})
			if err != nil {
				return err
			}
			properties[key] = val.WithLocations([]dyn.Location{{Line: propertiesOrder.Get(key)}})
		}

		dynMap[k] = dyn.NewValue(properties, []dyn.Location{{Line: order.Get(k)}})
	}

	saver := yamlsaver.NewSaverWithStyle(style)
	err := saver.SaveAsYAML(dynMap, outputPath, true)
	if err != nil {
		return err
	}
	return nil
}

func getAlphabeticalOrder[T any](mapping map[string]T) *yamlsaver.Order {
	order := []string{}
	for k := range mapping {
		order = append(order, k)
	}
	slices.Sort(order)
	return yamlsaver.NewOrder(order)
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

func isCliPath(path string) bool {
	return !strings.HasPrefix(path, "github.com/databricks/databricks-sdk-go")
}

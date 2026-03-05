package main

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"

	yaml3 "go.yaml.in/yaml/v3"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/jsonschema"
)

type annotationHandler struct {
	// Annotations read from the annotations.yml override file
	parsedAnnotations annotation.File
	// OpenAPI parser for reading descriptions directly from the spec
	openapi *openapiParser
	// Missing annotations for fields that are found in config that need to be added to the annotation file
	missingAnnotations annotation.File
	// Path mapping for converting between Go type paths and bundle paths
	pathMap *pathMapping
}

// Adds annotations to the JSON schema reading from the OpenAPI spec and
// the annotations.yml override file. OpenAPI descriptions are used as the base,
// and annotations.yml entries override them.
func newAnnotationHandler(annotationsPath string, openapi *openapiParser) (*annotationHandler, error) {
	data, err := annotation.LoadAndMerge([]string{annotationsPath})
	if err != nil {
		return nil, err
	}

	pathMap := buildPathMapping()

	// Convert bundle path keys in annotations to Go type path keys.
	resolved := annotation.File{}
	for key, fields := range data {
		typePath := key
		// If the key is a bundle path, resolve it to a Go type path.
		if tp, ok := pathMap.bundlePathToType[key]; ok {
			typePath = tp
		}
		resolved[typePath] = fields
	}

	return &annotationHandler{
		parsedAnnotations:  resolved,
		openapi:            openapi,
		missingAnnotations: annotation.File{},
		pathMap:            pathMap,
	}, nil
}

func (d *annotationHandler) addAnnotations(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	refPath := getPath(typ)
	shouldHandle := strings.HasPrefix(refPath, "github.com")
	if !shouldHandle {
		return s
	}

	// Step 1: Get base annotations from the OpenAPI spec
	openapiAnnotations := d.getOpenApiAnnotations(typ, s)

	// Step 2: Get override annotations from annotations.yml
	overrideAnnotations := d.parsedAnnotations[refPath]
	if overrideAnnotations == nil {
		overrideAnnotations = map[string]annotation.Descriptor{}
	}

	// Step 3: Merge. Start with OpenAPI base, apply overrides on top.
	merged := map[string]annotation.Descriptor{}
	for k, v := range openapiAnnotations {
		merged[k] = v
	}
	for k, v := range overrideAnnotations {
		if existing, ok := merged[k]; ok {
			merged[k] = mergeDescriptor(existing, v)
		} else {
			merged[k] = v
		}
	}

	rootTypeAnnotation, ok := merged[RootTypeKey]
	if ok {
		assignAnnotation(&s, rootTypeAnnotation)
	}

	for k, v := range s.Properties {
		item := merged[k]
		if item.Description == "" {
			item.Description = annotation.Placeholder

			// Only track missing annotations for CLI types, and only for
			// fields that don't have OpenAPI descriptions. Fields with
			// OpenAPI descriptions are handled at runtime and don't need
			// entries in annotations.yml.
			if isCliPath(refPath) {
				if _, hasOpenApi := openapiAnnotations[k]; !hasOpenApi {
					emptyAnnotations := d.missingAnnotations[refPath]
					if emptyAnnotations == nil {
						emptyAnnotations = map[string]annotation.Descriptor{}
						d.missingAnnotations[refPath] = emptyAnnotations
					}
					emptyAnnotations[k] = annotation.Descriptor{
						Description: annotation.Placeholder,
					}
				}
			}
		}
		assignAnnotation(v, item)
	}
	return s
}

// getOpenApiAnnotations reads annotations for the given type directly from
// the OpenAPI spec. This replaces the previous approach of pre-extracting
// annotations into a separate YAML file.
func (d *annotationHandler) getOpenApiAnnotations(typ reflect.Type, s jsonschema.Schema) map[string]annotation.Descriptor {
	result := map[string]annotation.Descriptor{}

	if d.openapi == nil {
		return result
	}

	ref, ok := d.openapi.findRef(typ)
	if !ok {
		return result
	}

	// Root type annotation
	preview := ref.Preview
	if preview == "PUBLIC" {
		preview = ""
	}
	outputOnly := isOutputOnly(ref)
	if ref.Description != "" || ref.Enum != nil || ref.Deprecated || ref.DeprecationMessage != "" || preview != "" || outputOnly != nil {
		if ref.Deprecated && ref.DeprecationMessage == "" {
			ref.DeprecationMessage = "This field is deprecated"
		}
		result[RootTypeKey] = annotation.Descriptor{
			Description:        ref.Description,
			Enum:               ref.Enum,
			DeprecationMessage: ref.DeprecationMessage,
			Preview:            preview,
			OutputOnly:         outputOnly,
		}
	}

	// Property annotations
	for k := range s.Properties {
		if refProp, ok := ref.Properties[k]; ok {
			propPreview := refProp.Preview
			if propPreview == "PUBLIC" {
				propPreview = ""
			}
			if refProp.Deprecated && refProp.DeprecationMessage == "" {
				refProp.DeprecationMessage = "This field is deprecated"
			}

			description := refProp.Description

			// If the field doesn't have a description, try to find the referenced type
			// and use its description.
			if description == "" && refProp.Reference != nil {
				refRefPath := *refProp.Reference
				refTypeName := strings.TrimPrefix(refRefPath, "#/components/schemas/")
				if refType, ok := d.openapi.ref[refTypeName]; ok {
					description = refType.Description
				}
			}

			result[k] = annotation.Descriptor{
				Description:        description,
				Enum:               refProp.Enum,
				Preview:            propPreview,
				DeprecationMessage: refProp.DeprecationMessage,
				OutputOnly:         isOutputOnly(*refProp),
			}
		}
	}

	return result
}

// mergeDescriptor merges an override descriptor on top of a base descriptor.
// Non-empty, non-PLACEHOLDER fields in the override take precedence.
func mergeDescriptor(base, override annotation.Descriptor) annotation.Descriptor {
	result := base
	if override.Description != "" && override.Description != annotation.Placeholder {
		result.Description = override.Description
	}
	if override.MarkdownDescription != "" {
		result.MarkdownDescription = override.MarkdownDescription
	}
	if override.Title != "" {
		result.Title = override.Title
	}
	if override.Default != nil {
		result.Default = override.Default
	}
	if override.Enum != nil {
		result.Enum = override.Enum
	}
	if override.MarkdownExamples != "" {
		result.MarkdownExamples = override.MarkdownExamples
	}
	if override.DeprecationMessage != "" {
		result.DeprecationMessage = override.DeprecationMessage
	}
	if override.Preview != "" {
		result.Preview = override.Preview
	}
	if override.OutputOnly != nil {
		result.OutputOnly = override.OutputOnly
	}
	return result
}

// Writes missing annotations with placeholder values back to the annotation file.
// Missing annotations are stored with Go type path keys internally, so we
// convert them to bundle path keys before writing.
func (d *annotationHandler) syncWithMissingAnnotations(outputPath string) error {
	existingFile, err := os.ReadFile(outputPath)
	if err != nil {
		return err
	}
	existing, err := yamlloader.LoadYAML("", bytes.NewBuffer(existingFile))
	if err != nil {
		return err
	}

	// Convert missing annotations from Go type paths to bundle paths.
	converted := annotation.File{}
	for k, v := range d.missingAnnotations {
		if !isCliPath(k) {
			fmt.Printf("Missing annotations for `%s` that are not in CLI package, try to fetch latest OpenAPI spec and regenerate annotations\n", k)
			continue
		}
		key := k
		if bundlePath, ok := d.pathMap.typeToBundlePath[k]; ok {
			key = bundlePath
		}
		converted[key] = v
	}

	missingAnnotations, err := convert.FromTyped(converted, dyn.NilValue)
	if err != nil {
		return err
	}

	output, err := merge.Merge(existing, missingAnnotations)
	if err != nil {
		return err
	}

	var outputTyped annotation.File
	err = convert.ToTyped(&outputTyped, output)
	if err != nil {
		return err
	}

	return saveYamlWithStyle(outputPath, outputTyped)
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

	if a.DeprecationMessage != "" {
		s.Deprecated = true
		s.DeprecationMessage = a.DeprecationMessage
	}

	if a.Preview == "PRIVATE" {
		s.DoNotSuggest = true
		s.Preview = a.Preview
	}

	if a.OutputOnly != nil && *a.OutputOnly {
		s.FieldBehaviors = []string{"OUTPUT_ONLY"}
	}

	s.MarkdownDescription = convertLinksToAbsoluteUrl(a.MarkdownDescription)
	s.Title = a.Title
	s.Enum = a.Enum
}

func saveYamlWithStyle(outputPath string, annotations annotation.File) error {
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
	var order []string
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
	re := regexp.MustCompile(`\[\\?(.*?)\]\((.*?)\)`)
	result := re.ReplaceAllStringFunc(s, func(match string) string {
		matches := re.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		originalText := matches[1]
		link := matches[2]

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

		if originalText != "_" {
			text = originalText
		}

		return fmt.Sprintf("[%s](%s)", text, absoluteURL)
	})

	return result
}

func isCliPath(path string) bool {
	return !strings.HasPrefix(path, "github.com/databricks/databricks-sdk-go")
}

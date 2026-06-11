package main

import (
	"bytes"
	"fmt"
	"maps"
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

	for k := range d.missingAnnotations {
		if !isCliPath(k) {
			delete(d.missingAnnotations, k)
			fmt.Printf("Missing annotations for `%s` that are not in CLI package, try to refresh .codegen/cli.json and regenerate annotations\n", k)
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

	var outputTyped annotation.File
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

	// Surface the launch stage as the raw x-databricks-launch-stage field (and,
	// below, as a human-readable prefix on the description). cli.json makes
	// launch_stage the single source of truth: x-databricks-preview above is
	// derivable from it (PRIVATE iff PRIVATE_PREVIEW) and is kept only for
	// backwards compatibility, so PRIVATE_PREVIEW must hide the field here too.
	//
	// OPEN QUESTION (confirm with the team before finalizing this PR): is
	// either x-databricks-preview or x-databricks-launch-stage consumed by
	// anything downstream of the published schema? If x-databricks-preview is
	// unused we can stop emitting it; if the raw x-databricks-launch-stage isn't
	// wanted in the published schema, drop s.LaunchStage and keep only the
	// description prefix below.
	if a.LaunchStage != "" {
		s.LaunchStage = a.LaunchStage
		if a.LaunchStage == "PRIVATE_PREVIEW" {
			s.DoNotSuggest = true
		}
	}

	if a.OutputOnly != nil && *a.OutputOnly {
		s.FieldBehaviors = []string{"OUTPUT_ONLY"}
	}

	s.MarkdownDescription = convertLinksToAbsoluteUrl(a.MarkdownDescription)
	s.Title = a.Title
	s.Enum = a.Enum
	s.EnumDescriptions = buildEnumDescriptions(a.Enum, a.EnumLaunchStages, a.EnumDescriptions)

	// Surface launch stage in hover tooltips. Editors generally only render the
	// standard description field, so we tag it directly rather than rely on
	// consumers reading x-databricks-launch-stage.
	if tag := annotation.PreviewTag(a.LaunchStage); tag != "" {
		s.Description = prefixWithPreviewTag(s.Description, tag)
	}
}

// buildEnumDescriptions produces the parallel enumDescriptions array VSCode
// renders next to each enum value. Each entry combines the launch-stage tag
// and the per-value description text. Returns nil when every entry would be
// empty so the field is omitted from the schema. The enum slice is the same
// one assigned to s.Enum, so the arrays stay index-aligned.
func buildEnumDescriptions(enum []any, launchStages, descriptions map[string]string) []string {
	if len(enum) == 0 || (len(launchStages) == 0 && len(descriptions) == 0) {
		return nil
	}
	result := make([]string, len(enum))
	hasContent := false
	for i, v := range enum {
		key, ok := v.(string)
		if !ok {
			continue
		}
		result[i] = prefixWithPreviewTag(descriptions[key], annotation.PreviewTag(launchStages[key]))
		if result[i] != "" {
			hasContent = true
		}
	}
	if !hasContent {
		return nil
	}
	return result
}

// prefixWithPreviewTag prepends the launch-stage tag to a description while
// guarding against double-tagging — if the description already starts with
// the tag, it is returned unchanged. An empty tag (GA) also takes the
// HasPrefix path, returning the description as-is.
func prefixWithPreviewTag(description, tag string) string {
	if description == "" {
		return tag
	}
	if strings.HasPrefix(description, tag) {
		return description
	}
	return tag + " " + description
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
	return yamlsaver.NewOrder(slices.Sorted(maps.Keys(mapping)))
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
		if after, ok := strings.CutPrefix(link, "#"); ok {
			text = after
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

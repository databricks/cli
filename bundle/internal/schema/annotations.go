package main

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/jsonschema"
)

type annotationHandler struct {
	// Annotations from cli.json merged with the annotations file.
	parsedAnnotations annotation.File
	// Annotations from the annotations file only: the content the CLI owns
	// and rewrites during sync.
	fileAnnotations annotation.File
	// Missing annotations for fields that are found in config that need to be added to the annotation file
	missingAnnotations annotation.File
}

// Adds annotations to the JSON schema reading from the annotation files.
// More details https://json-schema.org/understanding-json-schema/reference/annotations
func newAnnotationHandler(extracted, fromFile annotation.File) (*annotationHandler, error) {
	dropShadowingPlaceholders(fromFile, extracted)
	merged, err := mergeAnnotationFiles(extracted, fromFile)
	if err != nil {
		return nil, err
	}
	return &annotationHandler{
		parsedAnnotations:  merged,
		fileAnnotations:    fromFile,
		missingAnnotations: annotation.File{},
	}, nil
}

// dropShadowingPlaceholders removes PLACEHOLDER descriptions from fromFile for
// fields that cli.json documents. A PLACEHOLDER marks a field with no
// documentation anywhere, so once upstream gains a description the marker is
// stale and would otherwise shadow it in the merge, leaving the schema with no
// description at all. fromFile is mutated in place: the sync step then rewrites
// the annotations file without the stale markers. Entries that carry other
// hand-authored fields (e.g. deprecation_message) lose only the placeholder.
func dropShadowingPlaceholders(fromFile, extracted annotation.File) {
	for typePath, props := range fromFile {
		for key, d := range props {
			if d.Description != annotation.Placeholder {
				continue
			}
			if extracted[typePath][key].Description == "" {
				// Genuinely undocumented: keep the TODO marker.
				continue
			}
			d.Description = ""
			if isEmptyDescriptor(d) {
				delete(props, key)
			} else {
				props[key] = d
			}
		}
		if len(props) == 0 {
			delete(fromFile, typePath)
		}
	}
}

func isEmptyDescriptor(d annotation.Descriptor) bool {
	return d.Description == "" &&
		d.MarkdownDescription == "" &&
		d.Title == "" &&
		d.Default == nil &&
		d.Enum == nil &&
		d.MarkdownExamples == "" &&
		d.DeprecationMessage == "" &&
		d.Preview == "" &&
		d.OutputOnly == nil
}

// mergeAnnotationFiles merges later layers over earlier ones with the same
// semantics the on-disk annotation files used to be merged with: maps merge
// recursively, scalars take the later value, sequences concatenate.
func mergeAnnotationFiles(files ...annotation.File) (annotation.File, error) {
	prev := dyn.NilValue
	for _, f := range files {
		v, err := convert.FromTyped(f, dyn.NilValue)
		if err != nil {
			return nil, err
		}
		prev, err = merge.Merge(prev, v)
		if err != nil {
			return nil, err
		}
	}

	var data annotation.File
	err := convert.ToTyped(&data, prev)
	if err != nil {
		return nil, err
	}
	return data, nil
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
			emptyAnnotations[k] = annotation.Descriptor{Description: annotation.Placeholder}
		}
		assignAnnotation(v, item)
	}
	return s
}

// Writes the annotations file back in canonical form, adding placeholder
// descriptions for fields that have no documentation anywhere. Entries for
// fields that no longer exist in the config are dropped with a warning.
func (d *annotationHandler) syncWithMissingAnnotations(outputPath string, g *typeGraph) error {
	updated, err := mergeAnnotationFiles(d.fileAnnotations, d.missingAnnotations)
	if err != nil {
		return err
	}

	detached, err := saveAnnotationsFile(outputPath, updated, g)
	if err != nil {
		return err
	}
	for _, k := range detached {
		fmt.Printf("Dropping annotation for `%s`: no such field in the bundle configuration\n", k)
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

	if a.OutputOnly != nil && *a.OutputOnly {
		s.FieldBehaviors = []string{"OUTPUT_ONLY"}
	}

	s.MarkdownDescription = convertLinksToAbsoluteUrl(a.MarkdownDescription)
	s.Title = a.Title
	s.Enum = a.Enum
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

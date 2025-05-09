package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/libs/jsonschema"
)

const (
	rootFileName      = "reference.md"
	resourcesFileName = "resources.md"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run main.go <annotation-file> <output-file>")
		os.Exit(1)
	}

	annotationDir := os.Args[1]
	docsDir := os.Args[2]
	outputDir := path.Join(docsDir, "output")
	templatesDir := path.Join(docsDir, "templates")

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			log.Fatal(err)
		}
	}

	rootHeader, err := os.ReadFile(path.Join(templatesDir, rootFileName))
	if err != nil {
		log.Fatal(err)
	}
	err = generateDocs(
		[]string{path.Join(annotationDir, "annotations.yml")},
		path.Join(outputDir, rootFileName),
		reflect.TypeOf(config.Root{}),
		fillTemplateVariables(string(rootHeader)),
	)
	if err != nil {
		log.Fatal(err)
	}
	resourcesHeader, err := os.ReadFile(path.Join(templatesDir, resourcesFileName))
	if err != nil {
		log.Fatal(err)
	}
	err = generateDocs(
		[]string{path.Join(annotationDir, "annotations_openapi.yml"), path.Join(annotationDir, "annotations_openapi_overrides.yml"), path.Join(annotationDir, "annotations.yml")},
		path.Join(outputDir, resourcesFileName),
		reflect.TypeOf(config.Resources{}),
		fillTemplateVariables(string(resourcesHeader)),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func generateDocs(inputPaths []string, outputPath string, rootType reflect.Type, header string) error {
	annotations, err := annotation.LoadAndMerge(inputPaths)
	if err != nil {
		log.Fatal(err)
	}

	// schemas is used to resolve references to schemas
	schemas := map[string]*jsonschema.Schema{}
	// ownFields is used to track fields that are defined in the annotation file and should be included in the docs page
	ownFields := map[string]bool{}

	s, err := jsonschema.FromType(rootType, []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		func(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
			_, isOwnField := annotations[jsonschema.TypePath(typ)]
			if isOwnField {
				ownFields[jsonschema.TypePath(typ)] = true
			}

			refPath := getPath(typ)
			shouldHandle := strings.HasPrefix(refPath, "github.com")
			if !shouldHandle {
				schemas[jsonschema.TypePath(typ)] = &s
				return s
			}

			a := annotations[refPath]
			if a == nil {
				a = map[string]annotation.Descriptor{}
			}

			rootTypeAnnotation, ok := a["_"]
			if ok {
				assignAnnotation(&s, rootTypeAnnotation)
			}

			for k, v := range s.Properties {
				assignAnnotation(v, a[k])
			}

			schemas[jsonschema.TypePath(typ)] = &s
			return s
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	nodes := buildNodes(s, schemas, ownFields)
	err = buildMarkdown(nodes, outputPath, header)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func getPath(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}

func assignAnnotation(s *jsonschema.Schema, a annotation.Descriptor) {
	if a.Description != "" && a.Description != annotation.Placeholder {
		s.Description = a.Description
	}
	if a.MarkdownDescription != "" {
		s.MarkdownDescription = a.MarkdownDescription
	}
	if a.MarkdownExamples != "" {
		s.Examples = []string{a.MarkdownExamples}
	}
	if a.DeprecationMessage != "" {
		s.Deprecated = true
		s.DeprecationMessage = a.DeprecationMessage
	}
	if a.ForceNotDeprecated {
		s.Deprecated = false
		s.DeprecationMessage = ""
	}
	if a.Preview == "PRIVATE" {
		s.DoNotSuggest = true
		s.Preview = a.Preview
	}
}

func fillTemplateVariables(s string) string {
	currentDate := time.Now().Format("2006-01-02")
	return strings.ReplaceAll(s, "{{update_date}}", currentDate)
}

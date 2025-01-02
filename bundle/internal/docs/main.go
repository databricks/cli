package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/libs/jsonschema"
)

const (
	rootFileName = "reference.md"
	rootHeader   = `---
description: Configuration reference for databricks.yml
---

# Configuration reference

This article provides reference for keys supported by <DABS> configuration (YAML). See [\_](/dev-tools/bundles/index.md).
`
)

const (
	resourcesFileName = "resources-reference.md"
	resourcesHeader   = `---
description: Resources references for databricks.yml
---

# Resources reference

This article provides reference for keys supported by <DABS> configuration (YAML). See [\_](/dev-tools/bundles/index.md).
`
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run main.go <annotation-file> <output-file>")
		os.Exit(1)
	}

	annotationDir := os.Args[1]
	docsDir := os.Args[2]
	outputDir := path.Join(docsDir, "output")

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			log.Fatal(err)
		}
	}

	err := generateDocs(
		[]string{path.Join(annotationDir, "annotations.yml")},
		path.Join(outputDir, rootFileName),
		reflect.TypeOf(config.Root{}),
		rootHeader,
	)
	if err != nil {
		log.Fatal(err)
	}
	err = generateDocs(
		[]string{path.Join(annotationDir, "annotations_openapi.yml"), path.Join(annotationDir, "annotations_openapi_overrides.yml")},
		path.Join(outputDir, resourcesFileName),
		reflect.TypeOf(config.Resources{}),
		resourcesHeader,
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

	schemas := map[string]jsonschema.Schema{}
	customFields := map[string]bool{}

	s, err := jsonschema.FromType(rootType, []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		func(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
			_, isCustomField := annotations[jsonschema.TypePath(typ)]
			if isCustomField {
				customFields[jsonschema.TypePath(typ)] = true
			}
			schemas[jsonschema.TypePath(typ)] = s

			refPath := getPath(typ)
			shouldHandle := strings.HasPrefix(refPath, "github.com")
			if !shouldHandle {
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

			return s
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	nodes := getNodes(s, schemas, customFields)
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
		s.Examples = []any{a.MarkdownExamples}
	}
}

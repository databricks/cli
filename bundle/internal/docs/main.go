package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/libs/jsonschema"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run main.go <annotation-file> <output-file>")
		os.Exit(1)
	}

	annotationFile := os.Args[1]
	outputFile := os.Args[2]

	err := generateDocs(annotationFile, outputFile)
	if err != nil {
		log.Fatal(err)
	}
}

func generateDocs(workdir, outputPath string) error {
	annotationsPath := filepath.Join(workdir, "annotations.yml")

	annotations, err := annotation.LoadAndMerge([]string{annotationsPath})
	if err != nil {
		log.Fatal(err)
	}

	schemas := map[string]jsonschema.Schema{}
	customFields := map[string]bool{}

	s, err := jsonschema.FromType(reflect.TypeOf(config.Root{}), []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
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
	err = buildMarkdown(nodes, outputPath)
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

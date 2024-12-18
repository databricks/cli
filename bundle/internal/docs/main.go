package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

const Placeholder = "PLACEHOLDER"

func removeJobsFields(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	switch typ {
	case reflect.TypeOf(resources.Job{}):
		// This field has been deprecated in jobs API v2.1 and is always set to
		// "MULTI_TASK" in the backend. We should not expose it to the user.
		delete(s.Properties, "format")

		// These fields are only meant to be set by the DABs client (ie the CLI)
		// and thus should not be exposed to the user. These are used to annotate
		// jobs that were created by DABs.
		delete(s.Properties, "deployment")
		delete(s.Properties, "edit_mode")

	case reflect.TypeOf(jobs.GitSource{}):
		// These fields are readonly and are not meant to be set by the user.
		delete(s.Properties, "job_source")
		delete(s.Properties, "git_snapshot")

	default:
		// Do nothing
	}

	return s
}

// While volume_type is required in the volume create API, DABs automatically sets
// it's value to "MANAGED" if it's not provided. Thus, we make it optional
// in the bundle schema.
func makeVolumeTypeOptional(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	if typ != reflect.TypeOf(resources.Volume{}) {
		return s
	}

	res := []string{}
	for _, r := range s.Required {
		if r != "volume_type" {
			res = append(res, r)
		}
	}
	s.Required = res
	return s
}

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

type annotationFile map[string]map[string]annotation

type annotation struct {
	Description         string `json:"description,omitempty"`
	MarkdownDescription string `json:"markdown_description,omitempty"`
	Title               string `json:"title,omitempty"`
	Default             any    `json:"default,omitempty"`
	Enum                []any  `json:"enum,omitempty"`
}

func generateDocs(workdir, outputPath string) error {
	annotationsPath := filepath.Join(workdir, "annotations.yml")
	annotationsOpenApiPath := filepath.Join(workdir, "annotations_openapi.yml")
	annotationsOpenApiOverridesPath := filepath.Join(workdir, "annotations_openapi_overrides.yml")

	annotations, err := LoadAndMergeAnnotations([]string{annotationsPath, annotationsOpenApiPath, annotationsOpenApiOverridesPath})
	if err != nil {
		log.Fatal(err)
	}

	schemas := map[string]jsonschema.Schema{}

	s, err := jsonschema.FromType(reflect.TypeOf(config.Root{}), []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		removeJobsFields,
		makeVolumeTypeOptional,

		func(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
			schemas[jsonschema.TypePath(typ)] = s

			refPath := getPath(typ)
			shouldHandle := strings.HasPrefix(refPath, "github.com")
			if !shouldHandle {
				return s
			}

			a := annotations[refPath]
			if a == nil {
				a = map[string]annotation{}
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

	nodes := getNodes(s, schemas, annotations)
	err = buildMarkdown(nodes, outputPath)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func getPath(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}

func assignAnnotation(s *jsonschema.Schema, a annotation) {
	if a.Description != "" && a.Description != Placeholder {
		s.Description = a.Description
	}
	if a.MarkdownDescription != "" {
		s.MarkdownDescription = a.MarkdownDescription
	}
}

func LoadAndMergeAnnotations(sources []string) (annotationFile, error) {
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
	return data, nil
}

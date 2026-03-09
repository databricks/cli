package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/jsonschema"
)

type Components struct {
	Schemas map[string]jsonschema.Schema `json:"schemas,omitempty"`
}

type Specification struct {
	Components Components `json:"components"`
}

type openapiParser struct {
	ref map[string]jsonschema.Schema
}

const RootTypeKey = "_"

func newParser(path string) (*openapiParser, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	spec := Specification{}
	err = json.Unmarshal(b, &spec)
	if err != nil {
		return nil, err
	}

	p := &openapiParser{}
	p.ref = spec.Components.Schemas
	return p, nil
}

// This function checks if the input type:
// 1. Is a Databricks Go SDK type.
// 2. Has a Databricks Go SDK type embedded in it.
//
// If the above conditions are met, the function returns the JSON schema
// corresponding to the Databricks Go SDK type from the OpenAPI spec.
func (p *openapiParser) findRef(typ reflect.Type) (jsonschema.Schema, bool) {
	typs := []reflect.Type{typ}

	// Check for embedded Databricks Go SDK types.
	if typ.Kind() == reflect.Struct {
		for i := range typ.NumField() {
			if !typ.Field(i).Anonymous {
				continue
			}

			// Deference current type if it's a pointer.
			ctyp := typ.Field(i).Type
			for ctyp.Kind() == reflect.Ptr {
				ctyp = ctyp.Elem()
			}

			typs = append(typs, ctyp)
		}
	}

	for _, ctyp := range typs {
		// Skip if it's not a Go SDK type.
		if !strings.HasPrefix(ctyp.PkgPath(), "github.com/databricks/databricks-sdk-go") {
			continue
		}

		pkgName := path.Base(ctyp.PkgPath())
		k := fmt.Sprintf("%s.%s", pkgName, ctyp.Name())

		// Skip if the type is not in the openapi spec.
		_, ok := p.ref[k]
		if !ok {
			k = mapIncorrectTypNames(k)
			_, ok = p.ref[k]
			if !ok {
				continue
			}
		}

		// Return the first Go SDK type found in the openapi spec.
		return p.ref[k], true
	}

	return jsonschema.Schema{}, false
}

// Fix inconsistent type names between the Go SDK and the OpenAPI spec.
// E.g. "serving.PaLmConfig" in the Go SDK is "serving.PaLMConfig" in the OpenAPI spec.
func mapIncorrectTypNames(ref string) string {
	switch ref {
	case "serving.PaLmConfig":
		return "serving.PaLMConfig"
	case "serving.OpenAiConfig":
		return "serving.OpenAIConfig"
	case "serving.GoogleCloudVertexAiConfig":
		return "serving.GoogleCloudVertexAIConfig"
	case "serving.Ai21LabsConfig":
		return "serving.AI21LabsConfig"
	default:
		return ref
	}
}

func isOutputOnly(s jsonschema.Schema) string {
	if s.FieldBehaviors == nil || !slices.Contains(s.FieldBehaviors, "OUTPUT_ONLY") {
		return ""
	}
	return "true"
}

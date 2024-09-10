package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
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
		for i := 0; i < typ.NumField(); i++ {
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
			continue
		}

		// Return the first Go SDK type found in the openapi spec.
		return p.ref[k], true
	}

	return jsonschema.Schema{}, false
}

// Use the OpenAPI spec to load descriptions for the given type.
func (p *openapiParser) addDescriptions(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	ref, ok := p.findRef(typ)
	if !ok {
		return s
	}

	s.Description = ref.Description
	for k, v := range s.Properties {
		if refProp, ok := ref.Properties[k]; ok {
			v.Description = refProp.Description
		}
	}

	return s
}

// Use the OpenAPI spec add enum values for the given type.
func (p *openapiParser) addEnums(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	ref, ok := p.findRef(typ)
	if !ok {
		return s
	}

	s.Enum = append(s.Enum, ref.Enum...)
	for k, v := range s.Properties {
		if refProp, ok := ref.Properties[k]; ok {
			v.Enum = append(v.Enum, refProp.Enum...)
		}
	}

	return s
}

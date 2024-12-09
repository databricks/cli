package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/ghodss/yaml"

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
func (p *openapiParser) extractAnnotations(typ reflect.Type, outputPath, overridesPath string) error {
	annotations := map[string]annotation{}
	overrides := map[string]annotation{}

	b, err := os.ReadFile(overridesPath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(b, &overrides)
	if err != nil {
		return err
	}
	if overrides == nil {
		overrides = map[string]annotation{}
	}

	_, err = jsonschema.FromType(typ, []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		func(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
			ref, ok := p.findRef(typ)
			if !ok {
				return s
			}

			basePath := getPath(typ)
			annotations[basePath] = annotation{
				Description: ref.Description,
				Enum:        ref.Enum,
			}
			if ref.Description == "" {
				addEmptyOverride(basePath, overrides)
			}

			for k := range s.Properties {
				itemPath := fmt.Sprintf("%s.%s", basePath, k)

				if refProp, ok := ref.Properties[k]; ok {
					annotations[itemPath] = annotation{Description: refProp.Description, Enum: refProp.Enum}
					if refProp.Description == "" {
						addEmptyOverride(itemPath, overrides)
					}
				} else {
					addEmptyOverride(itemPath, overrides)
				}
			}
			return s
		},
	})

	if err != nil {
		return err
	}

	b, err = yaml.Marshal(overrides)
	if err != nil {
		return err
	}
	err = os.WriteFile(overridesPath, b, 0644)
	if err != nil {
		return err
	}

	b, err = yaml.Marshal(annotations)
	if err != nil {
		return err
	}
	b = bytes.Join([][]byte{[]byte("# This file is auto-generated. DO NOT EDIT."), b}, []byte("\n"))
	err = os.WriteFile(outputPath, b, 0644)
	if err != nil {
		return err
	}

	return nil
}

func addEmptyOverride(path string, overrides map[string]annotation) {
	if overrides[path].Description == "" {
		overrides[path] = annotation{Description: Placeholder}
	}

	a, ok := overrides[path]
	if !ok {
		a = annotation{}
	}
	if a.Description == "" {
		a.Description = Placeholder
	}
	overrides[path] = a
}

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

	"github.com/databricks/cli/libs/dyn/yamlloader"
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
	annotations := annotationFile{}
	overrides := annotationFile{}

	b, err := os.ReadFile(overridesPath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(b, &overrides)
	if err != nil {
		return err
	}
	if overrides == nil {
		overrides = annotationFile{}
	}

	_, err = jsonschema.FromType(typ, []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		func(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
			ref, ok := p.findRef(typ)
			if !ok {
				return s
			}

			basePath := getPath(typ)
			pkg := map[string]annotation{}
			annotations[basePath] = pkg

			if ref.Description != "" || ref.Enum != nil {
				pkg[RootTypeKey] = annotation{Description: ref.Description, Enum: ref.Enum}
			}

			for k := range s.Properties {
				if refProp, ok := ref.Properties[k]; ok {
					pkg[k] = annotation{Description: refProp.Description, Enum: refProp.Enum}
					if refProp.Description == "" {
						addEmptyOverride(k, basePath, overrides)
					}
				} else {
					addEmptyOverride(k, basePath, overrides)
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
	o, err := yamlloader.LoadYAML("", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	err = saveYamlWithStyle(overridesPath, o)
	if err != nil {
		return err
	}
	b, err = yaml.Marshal(annotations)
	if err != nil {
		return err
	}
	b = bytes.Join([][]byte{[]byte("# This file is auto-generated. DO NOT EDIT."), b}, []byte("\n"))
	err = os.WriteFile(outputPath, b, 0o644)
	if err != nil {
		return err
	}

	return nil
}

func addEmptyOverride(key, pkg string, overridesFile annotationFile) {
	if overridesFile[pkg] == nil {
		overridesFile[pkg] = map[string]annotation{}
	}

	overrides := overridesFile[pkg]
	if overrides[key].Description == "" {
		overrides[key] = annotation{Description: Placeholder}
	}

	a, ok := overrides[key]
	if !ok {
		a = annotation{}
	}
	if a.Description == "" {
		a.Description = Placeholder
	}
	overrides[key] = a
}

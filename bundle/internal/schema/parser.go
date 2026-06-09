package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/jsonschema"
)

type annotationParser struct {
	ref map[string]cliJSONSchema
}

const RootTypeKey = "_"

// deprecationMessage is the message emitted for any field or type that the spec
// marks as deprecated. The spec (.codegen/cli.json) carries only a deprecated
// flag, not a message, so we synthesize a fixed one here.
const deprecationMessage = "This field is deprecated"

// deprecationMessageFor returns the deprecation message for a deprecated field
// or type, or an empty string when it is not deprecated.
func deprecationMessageFor(deprecated bool) string {
	if deprecated {
		return deprecationMessage
	}
	return ""
}

func newParser(schemas map[string]cliJSONSchema) *annotationParser {
	return &annotationParser{ref: schemas}
}

// This function checks if the input type:
// 1. Is a Databricks Go SDK type.
// 2. Has a Databricks Go SDK type embedded in it.
//
// If the above conditions are met, the function returns the schema
// corresponding to the Databricks Go SDK type from the spec.
func (p *annotationParser) findRef(typ reflect.Type) (cliJSONSchema, bool) {
	typs := []reflect.Type{typ}

	// Check for embedded Databricks Go SDK types.
	if typ.Kind() == reflect.Struct {
		for field := range typ.Fields() {
			if !field.Anonymous {
				continue
			}

			// Deference current type if it's a pointer.
			ctyp := field.Type
			for ctyp.Kind() == reflect.Pointer {
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

		// Skip if the type is not in the spec.
		_, ok := p.ref[k]
		if !ok {
			k = mapIncorrectTypNames(k)
			_, ok = p.ref[k]
			if !ok {
				continue
			}
		}

		// Return the first Go SDK type found in the spec.
		return p.ref[k], true
	}

	return cliJSONSchema{}, false
}

// Fix inconsistent type names between the Go SDK and the spec.
// E.g. "serving.PaLmConfig" in the Go SDK is "serving.PaLMConfig" in the spec.
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

func isOutputOnly(behaviors []string) *bool {
	if !slices.Contains(behaviors, "OUTPUT_ONLY") {
		return nil
	}
	res := true
	return &res
}

// Use the spec to load descriptions for the given type.
func (p *annotationParser) extractAnnotations(typ reflect.Type, outputPath, overridesPath string) error {
	annotations := annotation.File{}
	overrides := annotation.File{}

	b, err := os.ReadFile(overridesPath)
	if err != nil {
		return err
	}
	overridesDyn, err := yamlloader.LoadYAML(overridesPath, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	err = convert.ToTyped(&overrides, overridesDyn)
	if err != nil {
		return err
	}
	if overrides == nil {
		overrides = annotation.File{}
	}

	_, err = jsonschema.FromType(typ, []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		func(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
			ref, ok := p.findRef(typ)
			if !ok {
				return s
			}

			basePath := getPath(typ)
			pkg := map[string]annotation.Descriptor{}
			annotations[basePath] = pkg
			preview := ref.Preview
			if preview == "PUBLIC" {
				preview = ""
			}
			if ref.Description != "" || ref.Enum != nil || ref.Deprecated || preview != "" {
				pkg[RootTypeKey] = annotation.Descriptor{
					Description:        ref.Description,
					Enum:               ref.Enum,
					DeprecationMessage: deprecationMessageFor(ref.Deprecated),
					Preview:            preview,
				}
			}

			for k := range s.Properties {
				if refProp, ok := ref.Fields[k]; ok {
					preview = refProp.Preview
					if preview == "PUBLIC" {
						preview = ""
					}

					description := refProp.Description

					// If the field doesn't have a description, try to find the referenced type
					// and use its description. This handles cases where the field references
					// a type that has a description but the field itself doesn't.
					if description == "" && refProp.Ref != "" {
						if refType, ok := p.ref[refProp.Ref]; ok {
							description = refType.Description
						}
					}

					pkg[k] = annotation.Descriptor{
						Description:        description,
						Enum:               refProp.Enum,
						Preview:            preview,
						DeprecationMessage: deprecationMessageFor(refProp.Deprecated),
						OutputOnly:         isOutputOnly(refProp.Behaviors),
					}
					if description == "" {
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

	err = saveYamlWithStyle(overridesPath, overrides)
	if err != nil {
		return err
	}
	err = saveYamlWithStyle(outputPath, annotations)
	if err != nil {
		return err
	}
	err = prependCommentToFile(outputPath, "# This file is auto-generated. DO NOT EDIT.\n")
	if err != nil {
		return err
	}
	return nil
}

func prependCommentToFile(outputPath, comment string) error {
	b, err := os.ReadFile(outputPath)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(comment)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	return err
}

func addEmptyOverride(key, pkg string, overridesFile annotation.File) {
	if overridesFile[pkg] == nil {
		overridesFile[pkg] = map[string]annotation.Descriptor{}
	}

	overrides := overridesFile[pkg]
	if overrides[key].Description == "" {
		overrides[key] = annotation.Descriptor{Description: annotation.Placeholder}
	}

	a, ok := overrides[key]
	if !ok {
		a = annotation.Descriptor{}
	}
	if a.Description == "" {
		a.Description = annotation.Placeholder
	}
	overrides[key] = a
}

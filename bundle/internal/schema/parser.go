package main

import (
	"fmt"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/internal/clijson"
	"github.com/databricks/cli/libs/jsonschema"
)

type annotationParser struct {
	ref map[string]*clijson.SchemaJSON
}

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

func newParser(schemas map[string]*clijson.SchemaJSON) *annotationParser {
	return &annotationParser{ref: schemas}
}

// This function checks if the input type:
// 1. Is a Databricks Go SDK type.
// 2. Has a Databricks Go SDK type embedded in it.
//
// If the above conditions are met, the function returns the schema
// corresponding to the Databricks Go SDK type from the spec.
func (p *annotationParser) findRef(typ reflect.Type) (*clijson.SchemaJSON, bool) {
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
		if _, ok := p.ref[k]; !ok {
			continue
		}

		// Return the first Go SDK type found in the spec.
		return p.ref[k], true
	}

	return nil, false
}

// previewFromLaunchStage maps a launch stage to the preview marker the bundle
// uses to hide a field or type from completions. cli.json no longer carries a
// separate preview flag — launch_stage is the single source of truth — so a
// private-preview stage is the only one that should not be suggested. Every
// other stage (GA, public preview, ...) yields an empty marker.
func previewFromLaunchStage(launchStage string) string {
	if launchStage == "PRIVATE_PREVIEW" {
		return "PRIVATE"
	}
	return ""
}

// normalizeLaunchStage drops the GA stage so it isn't persisted in the
// annotation file. GA is the implicit default for any field that isn't in a
// preview, so storing it would add a stage to thousands of entries for no
// benefit. cli.json is already filtered at min-stage=PRIVATE_PREVIEW upstream,
// so the only other values are the preview stages, which we keep verbatim.
func normalizeLaunchStage(launchStage string) string {
	if launchStage == "GA" {
		return ""
	}
	return launchStage
}

// notableEnumLaunchStages keeps only the enum values whose launch stage is
// worth surfacing (i.e. not GA), so the annotation file isn't polluted with a
// stage for every value of a GA enum. Returns nil when nothing remains.
func notableEnumLaunchStages(stages map[string]string) map[string]string {
	result := map[string]string{}
	for value, stage := range stages {
		if ls := normalizeLaunchStage(stage); ls != "" {
			result[value] = ls
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// nonEmptyEnumDescriptions drops blank per-value descriptions so the annotation
// file stays clean. Returns nil when nothing remains.
func nonEmptyEnumDescriptions(descriptions map[string]string) map[string]string {
	result := map[string]string{}
	for value, desc := range descriptions {
		if desc != "" {
			result[value] = desc
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// enumValues converts the contract's []string enum into the []any the
// annotation descriptor carries (its Enum field predates the typed contract).
func enumValues(vals []string) []any {
	if len(vals) == 0 {
		return nil
	}
	out := make([]any, len(vals))
	for i, v := range vals {
		out[i] = v
	}
	return out
}

func isOutputOnly(behaviors []string) *bool {
	if !slices.Contains(behaviors, "OUTPUT_ONLY") {
		return nil
	}
	res := true
	return &res
}

// Use the spec to load descriptions for the given type.
func (p *annotationParser) extractAnnotations(typ reflect.Type) (annotation.File, error) {
	annotations := annotation.File{}

	_, err := jsonschema.FromType(typ, []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		func(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
			ref, ok := p.findRef(typ)
			if !ok {
				return s
			}

			basePath := getPath(typ)
			// The contract carries no schema-level launch stage, so a type is
			// never itself marked private-preview — only its fields are (below).
			// Enum schemas do carry per-value launch stages and descriptions.
			enumLaunchStages := notableEnumLaunchStages(ref.EnumLaunchStages)
			enumDescriptions := nonEmptyEnumDescriptions(ref.EnumDescriptions)
			if ref.Description != "" || ref.Enum != nil || enumLaunchStages != nil || enumDescriptions != nil {
				annotations.SetSelf(basePath, annotation.Descriptor{
					Description:      ref.Description,
					Enum:             enumValues(ref.Enum),
					EnumLaunchStages: enumLaunchStages,
					EnumDescriptions: enumDescriptions,
				})
			}

			for k := range s.Properties {
				if refProp, ok := ref.Fields[k]; ok {
					preview := previewFromLaunchStage(refProp.LaunchStage)
					launchStage := normalizeLaunchStage(refProp.LaunchStage)

					description := refProp.Description

					// If the field doesn't have a description, try to find the referenced type
					// and use its description. This handles cases where the field references
					// a type that has a description but the field itself doesn't.
					if description == "" && refProp.Ref != "" {
						if refType, ok := p.ref[refProp.Ref]; ok {
							description = refType.Description
						}
					}

					annotations.SetField(basePath, k, annotation.Descriptor{
						Description:        description,
						Preview:            preview,
						LaunchStage:        launchStage,
						DeprecationMessage: deprecationMessageFor(refProp.Deprecated),
						OutputOnly:         isOutputOnly(refProp.Behaviors),
					})
				}
			}
			return s
		},
	})
	if err != nil {
		return nil, err
	}
	return annotations, nil
}

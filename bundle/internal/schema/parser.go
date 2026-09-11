package main

import (
	"errors"
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
// 2. Has a Databricks Go SDK type embedded in it, at any depth.
//
// If the above conditions are met, the function returns the schema
// corresponding to the Databricks Go SDK type from the spec.
//
// Embedded structs are traversed breadth first, mirroring how
// [jsonschema.FromType] flattens them, so the shallowest SDK type present in
// the spec wins. Traversing past the first level matters because some resources
// wrap the SDK spec in an intermediate struct rather than embedding it directly
// (e.g. PostgresProject -> PostgresProjectConfig -> postgres.ProjectSpec).
func (p *annotationParser) findRef(typ reflect.Type) (*clijson.SchemaJSON, bool) {
	bfsQueue := []reflect.Type{typ}

	for len(bfsQueue) > 0 {
		ctyp := bfsQueue[0]
		bfsQueue = bfsQueue[1:]

		if ref, ok := p.lookupSDKType(ctyp); ok {
			return ref, true
		}

		if ctyp.Kind() != reflect.Struct {
			continue
		}

		for field := range ctyp.Fields() {
			if !field.Anonymous {
				continue
			}

			// Deference current type if it's a pointer.
			ftyp := field.Type
			for ftyp.Kind() == reflect.Pointer {
				ftyp = ftyp.Elem()
			}

			bfsQueue = append(bfsQueue, ftyp)
		}
	}

	return nil, false
}

// lookupSDKType returns the spec schema for a Databricks Go SDK type, if the
// spec defines one for it.
func (p *annotationParser) lookupSDKType(typ reflect.Type) (*clijson.SchemaJSON, bool) {
	if !strings.HasPrefix(typ.PkgPath(), "github.com/databricks/databricks-sdk-go") {
		return nil, false
	}

	k := fmt.Sprintf("%s.%s", path.Base(typ.PkgPath()), typ.Name())
	ref, ok := p.ref[k]
	return ref, ok
}

// normalizeLaunchStage validates the contract's launch stage and drops GA so it
// isn't persisted in the annotation file. GA is the implicit default for any
// field that isn't in a preview, so storing it would add a stage to thousands
// of entries for no benefit. It errors on any stage the CLI doesn't recognize
// so a stage introduced upstream fails codegen instead of silently rendering as
// GA (see clijson.ParseLaunchStage).
func normalizeLaunchStage(launchStage string) (clijson.LaunchStage, error) {
	stage, err := clijson.ParseLaunchStage(launchStage)
	if err != nil {
		return "", err
	}
	if stage == clijson.LaunchStageGA {
		return "", nil
	}
	return stage, nil
}

// notableEnumLaunchStages keeps only the enum values whose launch stage is
// worth surfacing (i.e. not GA), so the annotation file isn't polluted with a
// stage for every value of a GA enum. Returns nil when nothing remains.
func notableEnumLaunchStages(stages map[string]string) (map[string]clijson.LaunchStage, error) {
	result := map[string]clijson.LaunchStage{}
	for value, stage := range stages {
		ls, err := normalizeLaunchStage(stage)
		if err != nil {
			return nil, err
		}
		if ls != "" {
			result[value] = ls
		}
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
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

	// Launch-stage validation happens inside the transform callback below, which
	// cannot return an error, so failures accumulate here and are returned after.
	var stageErr error
	_, err := jsonschema.FromType(typ, []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		func(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
			ref, ok := p.findRef(typ)
			if !ok {
				return s
			}

			basePath := getPath(typ)
			// A type carries no launch stage by default, so we set to GA, unless overridden.
			typeLaunchStage := annotation.OverrideLaunchStage(basePath, "")
			enumLaunchStages, enumErr := notableEnumLaunchStages(ref.EnumLaunchStages)
			if enumErr != nil {
				stageErr = errors.Join(stageErr, fmt.Errorf("%s: %w", basePath, enumErr))
			}
			enumDescriptions := nonEmptyEnumDescriptions(ref.EnumDescriptions)
			if ref.Description != "" || ref.Enum != nil || enumLaunchStages != nil || enumDescriptions != nil || typeLaunchStage != "" {
				annotations.SetSelf(basePath, annotation.Descriptor{
					Description:      ref.Description,
					LaunchStage:      typeLaunchStage,
					Enum:             enumValues(ref.Enum),
					EnumLaunchStages: enumLaunchStages,
					EnumDescriptions: enumDescriptions,
				})
			}

			for k := range s.Properties {
				if refProp, ok := ref.Fields[k]; ok {
					// An empty stage means the contract assigns none; keep it
					// unmarked rather than letting ParseLaunchStage default it to GA.
					var launchStage clijson.LaunchStage
					if refProp.LaunchStage != "" {
						stage, fieldErr := clijson.ParseLaunchStage(refProp.LaunchStage)
						if fieldErr != nil {
							stageErr = errors.Join(stageErr, fmt.Errorf("%s.%s: %w", basePath, k, fieldErr))
						}
						launchStage = stage
					}
					// Apply custom launch stage override (e.g. keep resource in Beta despite API being GA)
					launchStage = annotation.OverrideLaunchStage(basePath, launchStage)

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
	if stageErr != nil {
		return nil, stageErr
	}
	return annotations, nil
}

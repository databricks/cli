package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/internal/clijson"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/jsonschema"
)

type annotationParser struct {
	ref map[string]*clijson.SchemaJSON
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

	// Launch-stage validation happens inside the transform callback below, which
	// cannot return an error, so failures accumulate here and are returned after.
	var stageErr error
	_, err = jsonschema.FromType(typ, []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		func(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
			ref, ok := p.findRef(typ)
			if !ok {
				return s
			}

			basePath := getPath(typ)
			pkg := map[string]annotation.Descriptor{}
			annotations[basePath] = pkg
			// The contract carries no schema-level launch stage, so a type is
			// never itself marked private-preview — only its fields are (below).
			// Enum schemas do carry per-value launch stages and descriptions.
			enumLaunchStages, enumErr := notableEnumLaunchStages(ref.EnumLaunchStages)
			if enumErr != nil {
				stageErr = errors.Join(stageErr, fmt.Errorf("%s: %w", basePath, enumErr))
			}
			enumDescriptions := nonEmptyEnumDescriptions(ref.EnumDescriptions)
			if ref.Description != "" || ref.Enum != nil || enumLaunchStages != nil || enumDescriptions != nil {
				pkg[RootTypeKey] = annotation.Descriptor{
					Description:      ref.Description,
					Enum:             enumValues(ref.Enum),
					EnumLaunchStages: enumLaunchStages,
					EnumDescriptions: enumDescriptions,
				}
			}

			for k := range s.Properties {
				if refProp, ok := ref.Fields[k]; ok {
					launchStage, fieldErr := normalizeLaunchStage(refProp.LaunchStage)
					if fieldErr != nil {
						stageErr = errors.Join(stageErr, fmt.Errorf("%s.%s: %w", basePath, k, fieldErr))
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
						LaunchStage:        launchStage,
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
	if stageErr != nil {
		return stageErr
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

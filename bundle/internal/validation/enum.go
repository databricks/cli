package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"text/template"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structdiff/structpath"
	"github.com/databricks/cli/libs/structwalk"
)

type EnumPatternInfo struct {
	// The pattern for which the enum values in Values are applicable.
	// This is a string representation of [dyn.Pattern].
	Pattern string

	// List of valid enum values for the pattern. This field will be a string of the
	// form `{value1, value2, ...}` representing a Go slice literal.
	Values string
}

// isEnumType checks if a type is an enum (string type with a Values() method)
func isEnumType(typ reflect.Type) bool {
	// Must be a string type
	if typ.Kind() != reflect.String {
		return false
	}

	// Check for Values() method on both the type and its pointer type
	var valuesMethod reflect.Method
	var hasValues bool

	// First check on the type itself
	valuesMethod, hasValues = typ.MethodByName("Values")
	if !hasValues {
		// Then check on the pointer type
		ptrType := reflect.PointerTo(typ)
		valuesMethod, hasValues = ptrType.MethodByName("Values")
	}

	if !hasValues {
		return false
	}

	// Check method signature: func (T) Values() []T or func (*T) Values() []T
	methodType := valuesMethod.Type
	if methodType.NumIn() != 1 || methodType.NumOut() != 1 {
		return false
	}

	// Check return type is slice of the enum type
	returnType := methodType.Out(0)
	if returnType.Kind() != reflect.Slice {
		return false
	}

	elementType := returnType.Elem()
	return elementType == typ
}

// getEnumValues extracts enum values by calling the Values() method
func getEnumValues(typ reflect.Type) ([]string, error) {
	if !isEnumType(typ) {
		return nil, fmt.Errorf("type %s is not an enum type", typ.Name())
	}

	// Create a zero value of the type
	enumValue := reflect.Zero(typ)

	// Check if the method is on the type or pointer type
	valuesMethod := enumValue.MethodByName("Values")
	if !valuesMethod.IsValid() {
		// Try on pointer type
		enumPtr := reflect.New(typ)
		enumPtr.Elem().Set(enumValue)
		valuesMethod = enumPtr.MethodByName("Values")
		if !valuesMethod.IsValid() {
			return nil, fmt.Errorf("Values() method not found on type %s", typ.Name())
		}
	}

	results := valuesMethod.Call(nil)
	if len(results) != 1 {
		return nil, errors.New("Values() method should return exactly one value")
	}

	valuesSlice := results[0]
	if valuesSlice.Kind() != reflect.Slice {
		return nil, errors.New("Values() method should return a slice")
	}

	// Extract string values from the slice
	var enumStrings []string
	for i := range valuesSlice.Len() {
		value := valuesSlice.Index(i)
		enumStrings = append(enumStrings, value.String())
	}

	return enumStrings, nil
}

// extractEnumFields walks through a struct type and extracts enum field patterns
func extractEnumFields(typ reflect.Type) ([]EnumPatternInfo, error) {
	fieldsByPattern := make(map[string][]string)

	err := structwalk.WalkType(typ, func(path *structpath.PathNode, fieldType reflect.Type) bool {
		if path == nil {
			return true
		}

		// Do not generate enum validation code for fields that are internal or readonly.
		bundleTag := path.BundleTag()
		if bundleTag.Internal() || bundleTag.ReadOnly() {
			return false
		}

		// Check if this field type is an enum
		if isEnumType(fieldType) {
			// Get enum values
			enumValues, err := getEnumValues(fieldType)
			if err != nil {
				// Skip this field if we can't get enum values
				return true
			}

			fieldPath := path.DynPath()
			fieldsByPattern[fieldPath] = enumValues
		}
		return true
	})

	return buildEnumPatternInfos(fieldsByPattern), err
}

// buildEnumPatternInfos converts the field map to EnumPatternInfo slice
func buildEnumPatternInfos(fieldsByPattern map[string][]string) []EnumPatternInfo {
	patterns := make([]EnumPatternInfo, 0, len(fieldsByPattern))

	for pattern, values := range fieldsByPattern {
		patterns = append(patterns, EnumPatternInfo{
			Pattern: pattern,
			Values:  formatSliceToString(values),
		})
	}

	return patterns
}

// groupPatternsByKey groups patterns by their logical grouping key
func groupPatternsByKeyEnum(patterns []EnumPatternInfo) map[string][]EnumPatternInfo {
	groupedPatterns := make(map[string][]EnumPatternInfo)

	for _, pattern := range patterns {
		key := getGroupingKey(pattern.Pattern)
		groupedPatterns[key] = append(groupedPatterns[key], pattern)
	}

	return groupedPatterns
}

func filterTargetsAndEnvironmentsEnum(patterns map[string][]EnumPatternInfo) map[string][]EnumPatternInfo {
	filtered := make(map[string][]EnumPatternInfo)
	for key, patterns := range patterns {
		if key == "targets" || key == "environments" {
			continue
		}
		filtered[key] = patterns
	}
	return filtered
}

// sortGroupedPatterns sorts patterns within each group and returns them as a sorted slice
func sortGroupedPatternsEnum(groupedPatterns map[string][]EnumPatternInfo) [][]EnumPatternInfo {
	// Get sorted group keys
	groupKeys := make([]string, 0, len(groupedPatterns))
	for key := range groupedPatterns {
		groupKeys = append(groupKeys, key)
	}
	sort.Strings(groupKeys)

	// Build sorted result
	result := make([][]EnumPatternInfo, 0, len(groupKeys))
	for _, key := range groupKeys {
		patterns := groupedPatterns[key]

		// Sort patterns within each group by pattern
		sort.Slice(patterns, func(i, j int) bool {
			return patterns[i].Pattern < patterns[j].Pattern
		})

		result = append(result, patterns)
	}

	return result
}

// enumFields returns grouped enum field patterns for validation
func enumFields() ([][]EnumPatternInfo, error) {
	patterns, err := extractEnumFields(reflect.TypeOf(config.Root{}))
	if err != nil {
		return nil, err
	}
	groupedPatterns := groupPatternsByKeyEnum(patterns)
	filteredPatterns := filterTargetsAndEnvironmentsEnum(groupedPatterns)
	return sortGroupedPatternsEnum(filteredPatterns), nil
}

// Generate creates a Go source file with enum field validation rules
func generateEnumFields(outPath string) error {
	enumFields, err := enumFields()
	if err != nil {
		return fmt.Errorf("failed to generate enum fields: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outPath, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Parse and execute template
	tmpl, err := template.New("enum_validation").Parse(enumValidationTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var generatedCode bytes.Buffer
	if err := tmpl.Execute(&generatedCode, enumFields); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Write generated code to file
	filePath := filepath.Join(outPath, "enum_fields.go")
	if err := os.WriteFile(filePath, generatedCode.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write generated code: %w", err)
	}

	return nil
}

// enumValidationTemplate is the Go text template for generating the enum validation map
const enumValidationTemplate = `package generated

// THIS FILE IS AUTOGENERATED.
// DO NOT EDIT THIS FILE DIRECTLY.

import (
	_ "github.com/databricks/cli/libs/dyn"
)

// EnumFields maps [dyn.Pattern] to valid enum values they should have.
var EnumFields = map[string][]string{
{{- range . }}
{{- range . }}
	"{{ .Pattern }}": {{ .Values }},
{{- end }}
{{ end -}}
}
`

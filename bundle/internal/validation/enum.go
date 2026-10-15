package main

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"go/format"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"text/template"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/sdkmeta"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/structs/structwalk"
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
func extractEnumFields(typ reflect.Type, meta *sdkmeta.Meta) ([]EnumPatternInfo, error) {
	fieldsByPattern := make(map[string][]string)
	// Types of the visited nodes, to find the struct declaring a field.
	types := map[string]reflect.Type{}
	// cli.json schema of the API type at each visited node, if any.
	schemas := map[string]string{}

	err := structwalk.WalkType(typ, func(path *structpath.PatternNode, fieldType reflect.Type, field *reflect.StructField) bool {
		if path == nil {
			types[""] = typ
			return true
		}
		// Embedded structs are visited with the path of the struct embedding
		// them; keep the outermost type.
		if _, ok := types[path.String()]; !ok {
			types[path.String()] = fieldType
		}
		// Skip fields of API types that cli.json does not define, as the JSON
		// schema does. Fields whose type sdk-go unified with another one are
		// checked against the schema of the type they had before.
		key, parentKey := path.String(), path.Parent().String()
		schema, _ := meta.Name(fieldType)
		if name, ok := path.StringKey(); ok {
			if !meta.SchemaHasField(schemas[parentKey], name) {
				return false
			}
			if ref, ok := sdkmeta.FieldRef(schemas[parentKey], name); ok {
				schema = ref
			}
		} else {
			schema = schemas[parentKey]
		}
		if _, ok := schemas[key]; !ok {
			schemas[key] = schema
		}

		// Do not generate enum validation code for fields that are internal or readonly.
		if field != nil {
			bundleTag := structtag.BundleTag(field.Tag.Get("bundle"))
			if bundleTag.Internal() || bundleTag.ReadOnly() {
				return false
			}
		}

		// Check if this field type is an enum
		if isEnumType(fieldType) {
			// Get enum values
			enumValues, err := getEnumValues(fieldType)
			if err != nil {
				// Skip this field if we can't get enum values
				return true
			}

			fieldsByPattern[path.String()] = enumValues
		} else if enumValues, ok := apiEnumValues(meta, types, path, fieldType); ok {
			fieldsByPattern[path.String()] = enumValues
		}
		return true
	})

	return buildEnumPatternInfos(fieldsByPattern), err
}

// buildEnumPatternInfos converts the field map to EnumPatternInfo slice
// apiEnumValues returns the values of an API enum field from cli.json: either
// the field's type is an sdk-go enum, or the field is a plain string where the
// previous SDK had an enum (see sdkmeta.FieldRef).
func apiEnumValues(meta *sdkmeta.Meta, types map[string]reflect.Type, path *structpath.PatternNode, fieldType reflect.Type) ([]string, bool) {
	for fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}
	if fieldType.Kind() != reflect.String {
		return nil, false
	}
	// The previous SDK listed enum values alphabetically.
	if values, ok := meta.Enum(fieldType); ok {
		return slices.Sorted(slices.Values(values)), true
	}
	// Elements of a list of enums are matched through the list field.
	fieldPath := path
	if _, ok := path.StringKey(); !ok {
		fieldPath = path.Parent()
	}
	name, ok := fieldPath.StringKey()
	if !ok {
		return nil, false
	}
	parent, ok := meta.Name(types[fieldPath.Parent().String()])
	if !ok {
		return nil, false
	}
	ref, ok := sdkmeta.FieldRef(parent, name)
	if !ok {
		return nil, false
	}
	values, ok := meta.EnumOf(ref)
	return slices.Sorted(slices.Values(values)), ok
}

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
	groupKeys := slices.Sorted(maps.Keys(groupedPatterns))

	// Build sorted result
	result := make([][]EnumPatternInfo, 0, len(groupKeys))
	for _, key := range groupKeys {
		patterns := groupedPatterns[key]

		// Sort patterns within each group by pattern
		slices.SortFunc(patterns, func(a, b EnumPatternInfo) int {
			return cmp.Compare(a.Pattern, b.Pattern)
		})

		result = append(result, patterns)
	}

	return result
}

// enumFields returns grouped enum field patterns for validation
func enumFields(meta *sdkmeta.Meta) ([][]EnumPatternInfo, error) {
	patterns, err := extractEnumFields(reflect.TypeFor[config.Root](), meta)
	if err != nil {
		return nil, err
	}
	groupedPatterns := groupPatternsByKeyEnum(patterns)
	filteredPatterns := filterTargetsAndEnvironmentsEnum(groupedPatterns)
	return sortGroupedPatternsEnum(filteredPatterns), nil
}

// Generate creates a Go source file with enum field validation rules
func generateEnumFields(outPath string, meta *sdkmeta.Meta) error {
	enumFields, err := enumFields(meta)
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

	// Format the generated source with gofmt-equivalent rules so minor template
	// whitespace changes don't create noisy diffs in the checked-in artifacts.
	formatted, err := format.Source(generatedCode.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	// Write generated code to file
	filePath := filepath.Join(outPath, "enum_fields.go")
	if err := os.WriteFile(filePath, formatted, 0o644); err != nil {
		return fmt.Errorf("failed to write generated code: %w", err)
	}

	return nil
}

// enumValidationTemplate is the Go text template for generating the enum validation map
const enumValidationTemplate = `package generated

// THIS FILE IS AUTOGENERATED.
// DO NOT EDIT THIS FILE DIRECTLY.

// EnumFields maps [dyn.Pattern] to valid enum values they should have.
var EnumFields = map[string][]string{
{{- range . }}
{{- range . }}
	"{{ .Pattern }}": {{ .Values }},
{{- end }}
{{ end -}}
}
`

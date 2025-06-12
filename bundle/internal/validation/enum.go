package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structdiff/structpath"
	"github.com/databricks/cli/libs/structwalk"
)

type EnumPatternInfo struct {
	// The full pattern for which the enum values are applicable.
	// This is a string representation of [dyn.Pattern].
	Pattern string

	// List of valid enum values for this pattern. This field will be a string of the
	// form `{value1, value2, ...}`.
	EnumValues string
}

// hasValuesMethod checks if the pointer to a type has a Values() method
func hasValuesMethod(typ reflect.Type) bool {
	// Check if the pointer to the type has a Values() method
	ptrType := reflect.PointerTo(typ)
	return checkValuesMethodSignature(ptrType)
}

// checkValuesMethodSignature verifies that a type has a Values() method with correct signature
func checkValuesMethodSignature(typ reflect.Type) bool {
	method, exists := typ.MethodByName("Values")
	if !exists {
		return false
	}

	// Verify the method signature: func() []string
	methodType := method.Type
	if methodType.NumIn() != 1 || methodType.NumOut() != 1 {
		return false
	}

	// Check return type is []string
	returnType := methodType.Out(0)
	if returnType.Kind() != reflect.Slice || returnType.Elem().Kind() != reflect.String {
		return false
	}

	return true
}

// getEnumValues calls the Values() method on a pointer to the type to get valid enum values
func getEnumValues(typ reflect.Type) ([]string, error) {
	// Create a pointer to zero value of the type and call Values() on it
	zeroValue := reflect.Zero(typ)
	ptrValue := reflect.New(typ)
	ptrValue.Elem().Set(zeroValue)
	method := ptrValue.MethodByName("Values")

	if !method.IsValid() {
		return nil, fmt.Errorf("Values method not found on pointer to type %s", typ.Name())
	}

	result := method.Call(nil)
	if len(result) != 1 {
		return nil, fmt.Errorf("Values method should return exactly one value")
	}

	enumSlice := result[0]
	if enumSlice.Kind() != reflect.Slice {
		return nil, fmt.Errorf("Values method should return a slice")
	}

	values := make([]string, enumSlice.Len())
	for i := 0; i < enumSlice.Len(); i++ {
		values[i] = enumSlice.Index(i).String()
	}

	return values, nil
}

// extractEnumFields walks through a struct type and extracts enum field patterns
func extractEnumFields(typ reflect.Type) ([]EnumPatternInfo, error) {
	var patterns []EnumPatternInfo

	err := structwalk.WalkType(typ, func(path *structpath.PathNode, fieldType reflect.Type) bool {
		if path == nil {
			return true
		}

		// Do not generate enum validation code for fields that are internal or readonly.
		bundleTag := path.BundleTag()
		if bundleTag.Internal() || bundleTag.ReadOnly() {
			return false
		}

		// Check if this type has a Values() method on its pointer
		if !hasValuesMethod(fieldType) {
			return true
		}

		// Get the enum values
		enumValues, err := getEnumValues(fieldType)
		if err != nil {
			// Skip if we can't get enum values
			return true
		}

		// Store the full pattern path (not parent path)
		fullPattern := path.DynPath()
		patterns = append(patterns, EnumPatternInfo{
			Pattern:    fullPattern,
			EnumValues: formatValues(enumValues),
		})

		return true
	})
	if err != nil {
		return nil, err
	}

	return patterns, nil
}

// getEnumGroupingKey determines the grouping key for organizing patterns
// TODO: Combine with the required function.
func getEnumGroupingKey(pattern string) string {
	parts := strings.Split(pattern, ".")

	// Group resources by their resource type (e.g., "resources.jobs")
	if parts[0] == "resources" && len(parts) > 1 {
		return parts[0] + "." + parts[1]
	}

	// Use the top level key for other fields
	return parts[0]
}

// groupEnumPatternsByKey groups patterns by their logical grouping key
func groupEnumPatternsByKey(patterns []EnumPatternInfo) map[string][]EnumPatternInfo {
	groupedPatterns := make(map[string][]EnumPatternInfo)

	for _, pattern := range patterns {
		key := getEnumGroupingKey(pattern.Pattern)
		groupedPatterns[key] = append(groupedPatterns[key], pattern)
	}

	return groupedPatterns
}

func filterEnumTargetsAndEnvironments(patterns map[string][]EnumPatternInfo) map[string][]EnumPatternInfo {
	filtered := make(map[string][]EnumPatternInfo)
	for key, patterns := range patterns {
		if key == "targets" || key == "environments" {
			continue
		}
		filtered[key] = patterns
	}
	return filtered
}

// sortGroupedEnumPatterns sorts patterns within each group and returns them as a sorted slice
func sortGroupedEnumPatterns(groupedPatterns map[string][]EnumPatternInfo) [][]EnumPatternInfo {
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

		// Sort patterns within each group by pattern path
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
	groupedPatterns := groupEnumPatternsByKey(patterns)
	filteredPatterns := filterEnumTargetsAndEnvironments(groupedPatterns)
	return sortGroupedEnumPatterns(filteredPatterns), nil
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
	"{{ .Pattern }}": {{ .EnumValues }},
{{- end }}
{{ end -}}
}
`

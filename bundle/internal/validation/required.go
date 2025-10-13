package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/structs/structwalk"
)

type RequiredPatternInfo struct {
	// The pattern for which the fields in Required are applicable.
	// This is a string representation of [dyn.Pattern].
	Parent string

	// List of required fields that should be set for every path in the
	// config tree that matches the pattern. This field will be a string of the
	// form `{field1, field2, ...}`.
	RequiredFields string
}

// formatSliceToString formats a list of values into string of the form `{value1, value2, ...}`
// representing a Go slice literal.
func formatSliceToString(values []string) string {
	if len(values) == 0 {
		return "{}"
	}

	var quoted []string
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}

	return "{" + strings.Join(quoted, ", ") + "}"
}

// extractRequiredFields walks through a struct type and extracts required field patterns
func extractRequiredFields(typ reflect.Type) ([]RequiredPatternInfo, error) {
	fieldsByPattern := make(map[string][]string)

	err := structwalk.WalkType(typ, func(path *structpath.PathNode, _ reflect.Type, field *reflect.StructField) bool {
		if path == nil || field == nil {
			return true
		}

		// Do not generate required validation code for fields that are internal or readonly.
		bundleTag := structtag.BundleTag(field.Tag.Get("bundle"))
		if bundleTag.Internal() || bundleTag.ReadOnly() {
			return false
		}

		// The "omitempty" tag indicates the field is optional in bundle config.
		jsonTag := structtag.JSONTag(field.Tag.Get("json"))
		if jsonTag.OmitEmpty() {
			return true
		}

		fieldName, ok := path.StringKey()
		if !ok {
			return true
		}

		parentPath := path.Parent().String()
		fieldsByPattern[parentPath] = append(fieldsByPattern[parentPath], fieldName)
		return true
	})

	return buildPatternInfos(fieldsByPattern), err
}

// buildPatternInfos converts the field map to PatternInfo slice
func buildPatternInfos(fieldsByPattern map[string][]string) []RequiredPatternInfo {
	patterns := make([]RequiredPatternInfo, 0, len(fieldsByPattern))

	for parentPath, fields := range fieldsByPattern {
		patterns = append(patterns, RequiredPatternInfo{
			Parent:         parentPath,
			RequiredFields: formatSliceToString(fields),
		})
	}

	return patterns
}

// getGroupingKey determines the grouping key for organizing patterns
func getGroupingKey(pattern string) string {
	parts := strings.Split(pattern, ".")

	// Group resources by their resource type (e.g., "resources.jobs")
	if parts[0] == "resources" && len(parts) > 1 {
		return parts[0] + "." + parts[1]
	}

	// Use the top level key for other fields
	return parts[0]
}

// groupPatternsByKey groups patterns by their logical grouping key
func groupPatternsByKey(patterns []RequiredPatternInfo) map[string][]RequiredPatternInfo {
	groupedPatterns := make(map[string][]RequiredPatternInfo)

	for _, pattern := range patterns {
		key := getGroupingKey(pattern.Parent)
		groupedPatterns[key] = append(groupedPatterns[key], pattern)
	}

	return groupedPatterns
}

func filterTargetsAndEnvironments(patterns map[string][]RequiredPatternInfo) map[string][]RequiredPatternInfo {
	filtered := make(map[string][]RequiredPatternInfo)
	for key, patterns := range patterns {
		if key == "targets" || key == "environments" {
			continue
		}
		filtered[key] = patterns
	}
	return filtered
}

// sortGroupedPatterns sorts patterns within each group and returns them as a sorted slice
func sortGroupedPatterns(groupedPatterns map[string][]RequiredPatternInfo) [][]RequiredPatternInfo {
	// Get sorted group keys
	groupKeys := make([]string, 0, len(groupedPatterns))
	for key := range groupedPatterns {
		groupKeys = append(groupKeys, key)
	}
	sort.Strings(groupKeys)

	// Build sorted result
	result := make([][]RequiredPatternInfo, 0, len(groupKeys))
	for _, key := range groupKeys {
		patterns := groupedPatterns[key]

		// Sort patterns within each group by parent path
		sort.Slice(patterns, func(i, j int) bool {
			return patterns[i].Parent < patterns[j].Parent
		})

		result = append(result, patterns)
	}

	return result
}

// RequiredFields returns grouped required field patterns for validation
func requiredFields() ([][]RequiredPatternInfo, error) {
	patterns, err := extractRequiredFields(reflect.TypeOf(config.Root{}))
	if err != nil {
		return nil, err
	}
	groupedPatterns := groupPatternsByKey(patterns)
	filteredPatterns := filterTargetsAndEnvironments(groupedPatterns)
	return sortGroupedPatterns(filteredPatterns), nil
}

// Generate creates a Go source file with required field validation rules
func generateRequiredFields(outPath string) error {
	requiredFields, err := requiredFields()
	if err != nil {
		return fmt.Errorf("failed to generate required fields: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outPath, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Parse and execute template
	tmpl, err := template.New("validation").Parse(validationTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var generatedCode bytes.Buffer
	if err := tmpl.Execute(&generatedCode, requiredFields); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	formatted, err := format.Source(generatedCode.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	// Write generated code to file
	filePath := filepath.Join(outPath, "required_fields.go")
	if err := os.WriteFile(filePath, formatted, 0o644); err != nil {
		return fmt.Errorf("failed to write generated code: %w", err)
	}

	return nil
}

// validationTemplate is the Go text template for generating the validation map
const validationTemplate = `package generated

// THIS FILE IS AUTOGENERATED.
// DO NOT EDIT THIS FILE DIRECTLY.

// RequiredFields maps [dyn.Pattern] to required fields they should have.
var RequiredFields = map[string][]string{
{{- range . }}
{{- range . }}
	"{{ .Parent }}": {{ .RequiredFields }},
{{- end }}
{{ end -}}
}
`

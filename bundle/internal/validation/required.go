package main

import (
	"bytes"
	"cmp"
	"fmt"
	"go/format"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"text/template"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/sdkmeta"
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
func extractRequiredFields(typ reflect.Type, meta *sdkmeta.Meta) ([]RequiredPatternInfo, error) {
	fieldsByPattern := make(map[string][]string)
	// Required fields of the struct at each visited node.
	required := map[string][]string{}
	// cli.json schema of the API type at each visited node, if any.
	schemas := map[string]string{}
	types := map[string]reflect.Type{}

	err := structwalk.WalkType(typ, func(path *structpath.PatternNode, fieldType reflect.Type, field *reflect.StructField) bool {
		if path == nil {
			required[""] = meta.Required(typ)
			return true
		}
		key, parentKey := path.String(), path.Parent().String()
		// Embedded structs are visited with the path of the struct embedding
		// them; keep the outermost type.
		if _, ok := types[key]; !ok {
			types[key] = fieldType
		}
		// Skip fields of API types that cli.json does not define, as the JSON
		// schema does.
		if name, ok := path.StringKey(); ok && !meta.SchemaHasField(schemas[parentKey], name) {
			return false
		}
		// Fields whose type sdk-go unified with another one keep the required
		// fields of the type they had before (see sdkmeta.FieldRef).
		schema, _ := meta.Name(fieldType)
		if name, ok := path.StringKey(); ok {
			if ref, ok := sdkmeta.FieldRef(schemas[parentKey], name); ok {
				schema = ref
			}
		} else {
			schema = schemas[parentKey]
		}
		schemas[key] = schema
		if ref, _ := meta.Name(fieldType); schema != "" && schema != ref {
			required[key] = sdkmeta.RequiredFields(schema)
		} else {
			required[key] = meta.Required(fieldType)
		}
		if field == nil {
			return true
		}

		// Fields without a json tag (e.g. anonymous embeds) have no independent
		// JSON key and cannot be required. Continue walking their children.
		rawTag := field.Tag.Get("json")
		if rawTag == "" {
			return true
		}

		// Do not generate required validation code for fields that are internal or readonly.
		bundleTag := structtag.BundleTag(field.Tag.Get("bundle"))
		if bundleTag.Internal() || bundleTag.ReadOnly() {
			return false
		}

		// The "omitempty" tag indicates the field is optional in bundle config.
		fieldName, ok := path.StringKey()
		if !ok {
			return true
		}
		parentPath := path.Parent().String()

		// API types mark every field omitempty; their required fields come
		// from sdkmeta.
		jsonTag := structtag.JSONTag(rawTag)
		if jsonTag.OmitEmpty() && !slices.Contains(required[parentPath], fieldName) {
			return true
		}

		fieldsByPattern[parentPath] = append(fieldsByPattern[parentPath], fieldName)
		return true
	})

	for parentPath, fields := range fieldsByPattern {
		order := legacyFieldOrder(meta, types[parentPath])
		if parentPath == "" {
			order = legacyFieldOrder(meta, typ)
		}
		slices.SortStableFunc(fields, func(a, b string) int { return order[a] - order[b] })
	}
	return buildPatternInfos(fieldsByPattern), err
}

// legacyFieldOrder returns the position of each field of t in the order the
// walk visited them with the previous SDK types: embedded structs are expanded
// in place, and the fields of API types, which the previous SDK declared
// alphabetically, are sorted by name.
func legacyFieldOrder(meta *sdkmeta.Meta, t reflect.Type) map[string]int {
	order := map[string]int{}
	var visit func(t reflect.Type)
	visit = func(t reflect.Type) {
		for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Map {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return
		}
		fields := slices.Collect(t.Fields())
		if _, ok := meta.Name(t); ok {
			slices.SortStableFunc(fields, func(a, b reflect.StructField) int { return strings.Compare(a.Name, b.Name) })
		}
		for _, f := range fields {
			if f.Anonymous {
				visit(f.Type)
				continue
			}
			name := structtag.JSONTag(f.Tag.Get("json")).Name()
			if _, ok := order[name]; !ok {
				order[name] = len(order)
			}
		}
	}
	visit(t)
	return order
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
	groupKeys := slices.Sorted(maps.Keys(groupedPatterns))

	// Build sorted result
	result := make([][]RequiredPatternInfo, 0, len(groupKeys))
	for _, key := range groupKeys {
		patterns := groupedPatterns[key]

		// Sort patterns within each group by parent path
		slices.SortFunc(patterns, func(a, b RequiredPatternInfo) int {
			return cmp.Compare(a.Parent, b.Parent)
		})

		result = append(result, patterns)
	}

	return result
}

// RequiredFields returns grouped required field patterns for validation
func requiredFields(meta *sdkmeta.Meta) ([][]RequiredPatternInfo, error) {
	patterns, err := extractRequiredFields(reflect.TypeFor[config.Root](), meta)
	if err != nil {
		return nil, err
	}
	groupedPatterns := groupPatternsByKey(patterns)
	filteredPatterns := filterTargetsAndEnvironments(groupedPatterns)
	return sortGroupedPatterns(filteredPatterns), nil
}

// Generate creates a Go source file with required field validation rules
func generateRequiredFields(outPath string, meta *sdkmeta.Meta) error {
	requiredFields, err := requiredFields(meta)
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

	// Format the generated source with gofmt-equivalent rules so minor template
	// whitespace changes don't create noisy diffs in the checked-in artifacts.
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

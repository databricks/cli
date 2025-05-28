package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	_ "github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/jsonschema"
)

func Generate(rootSchema *jsonschema.Schema, outputPath string) error {
	nodes, err := gatherRequiredFields(rootSchema)
	if err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	// Ensure output directory exists
	err = os.MkdirAll(outputPath, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Parse the template
	tmpl, err := template.New("validation").Parse(validationTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute the template
	var generatedCode bytes.Buffer
	err = tmpl.Execute(&generatedCode, nodes)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Write the generated code to a file
	filePath := filepath.Join(outputPath, "required_fields.go")
	err = os.WriteFile(filePath, generatedCode.Bytes(), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write generated code: %w", err)
	}

	return nil
}

func anyToSchema(a any) (*jsonschema.Schema, error) {
	b, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal any to schema: %w", err)
	}

	res := &jsonschema.Schema{}
	err = json.Unmarshal(b, res)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal any to schema: %w", err)
	}

	return res, nil
}

// PatternInfo holds the path and required status for a schema node.
type PatternInfo struct {
	// Pattern for which the fields in Required are applicable.
	// This is a string presentation of patterns of type [dyn.Pattern]
	// and can be converted to so by using [dyn.NewPatternFromString]
	Pattern string

	// List of required fields that should be set for every path in the
	// config tree that matches the pattern. This field be a string of the
	// form `{field1, field2, ...}`.
	RequiredFields string
}

type walker struct {
	// JSON schema references ($ref) resolved so far while walking the JSON schema.
	// We keep track to avoid getting trapped in a circular loop.
	seenRefs map[string]bool

	// List of patterns gathered while walking the JSON schema.
	patterns []PatternInfo

	originalSchema *jsonschema.Schema
}

func (w *walker) walk(p string, curr *jsonschema.Schema) error {
	if curr == nil {
		return nil
	}

	// "environments" has been deprecated for a while. No need
	// to accumulate required fields for it.
	if p == "environments" {
		return nil
	}

	if len(curr.Required) > 0 {
		requiredList := strings.Builder{}
		for i, r := range curr.Required {
			if i == 0 {
				requiredList.WriteString("{")
			}
			requiredList.WriteString(fmt.Sprintf("%q", r))
			if i < len(curr.Required)-1 {
				requiredList.WriteString(", ")
			}
			if i == len(curr.Required)-1 {
				requiredList.WriteString("}")
			}
		}
		r := requiredList.String()

		w.patterns = append(w.patterns, PatternInfo{
			Pattern:        p,
			RequiredFields: r,
		})
	}

	// Case 1: Resolve references.
	if curr.Reference != nil && *curr.Reference != "" {
		refString := *curr.Reference
		if w.seenRefs[refString] {
			// We've already seen and visited this $ref, so we can skip it.
			return nil
		}

		w.seenRefs[refString] = true

		refSchema, err := w.originalSchema.GetDefinition(refString)
		if err != nil {
			return fmt.Errorf("failed to resolve $ref %q at path %s: %w", refString, p, err)
		}

		err = w.walk(p, refSchema)
		if err != nil {
			return err
		}

		// Stop tracking this $ref now that we've processed it. This allows
		// it to be visited again if reached via a different path/context.
		w.seenRefs[refString] = false
	}

	// Case 2: Process oneOf branches.
	if len(curr.OneOf) > 0 {
		for i, s := range curr.OneOf {
			if s.Type != "object" && s.Type != "array" {
				// No need to walk primitive oneOf branches. These are normally added in the
				// JSON schema to allow for variable interpolation.
				continue
			}

			err := w.walk(p, &s)
			if err != nil {
				return fmt.Errorf("error processing oneOf branch %d at path %s: %w", i, p, err)
			}

		}
		return nil
	}

	// Case 3: Process additionalProperties. This represents a map object.
	if curr.AdditionalProperties != nil {
		v, ok := curr.AdditionalProperties.(map[string]any)
		if ok && len(v) > 0 {
			s, err := anyToSchema(curr.AdditionalProperties)
			if err != nil {
				return fmt.Errorf("failed to convert additionalProperties to schema: %w", err)
			}

			p = p + ".*"
			err = w.walk(p, s)
			if err != nil {
				return err
			}
		}
	}

	// Case 4: Process properties. This represents a struct.
	for name, s := range curr.Properties {
		if s == nil {
			panic(fmt.Sprintf("property %q is nil at path %s", name, p))
		}

		var np string
		if p == "" {
			np = name
		} else {
			np = p + "." + name
		}

		err := w.walk(np, s)
		if err != nil {
			return err
		}
	}

	// Case 5: Process items. This represents an array.
	if curr.Items != nil {
		p = p + "[*]"

		return w.walk(p, curr.Items)
	}

	return nil
}

// gatherRequiredFields recursively traverses the schema and extracts required fields.
func gatherRequiredFields(currentRootSchema *jsonschema.Schema) ([][]PatternInfo, error) {
	walker := &walker{
		seenRefs:       make(map[string]bool),
		originalSchema: currentRootSchema,
	}

	err := walker.walk("", currentRootSchema)
	if err != nil {
		return nil, err
	}

	// Group the required fields by the name of their top level field. This allow for better rendering.
	patterns := walker.patterns
	patternMap := map[string][]PatternInfo{}
	for _, node := range patterns {
		parts := strings.Split(node.Pattern, ".")

		var k string
		if parts[0] == "resources" {
			// Group resources by their type.
			k = parts[0] + "." + parts[1]
		} else if parts[0] == "targets" {
			// group target overrides by their first 3 keys
			k = parts[0] + "." + parts[1] + "." + parts[2]
		} else {
			// Just use the top level key for other fields.
			k = parts[0]
		}

		if _, ok := patternMap[k]; ok {
			patternMap[k] = append(patternMap[k], node)
		} else {
			patternMap[k] = []PatternInfo{node}
		}
	}

	// Convert map to an array to make it easier to render.
	sortedNodeMap := make([][]PatternInfo, 0, len(patternMap))
	sortedKeys := []string{}
	for k := range patternMap {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)
	for _, k := range sortedKeys {
		sort.Slice(patternMap[k], func(i, j int) bool {
			return patternMap[k][i].Pattern < patternMap[k][j].Pattern
		})
		sortedNodeMap = append(sortedNodeMap, patternMap[k])
	}

	return sortedNodeMap, nil
}

// validationTemplate is the Go text template for generating the map.
const validationTemplate = `package generated

// THIS FILE IS AUTOGENERATED.
// DO NOT EDIT THIS FILE DIRECTLY.

import (
	_ "github.com/databricks/cli/libs/dyn"
)

// RequiredFields maps [dyn.Pattern] to required fields they should have.
var RequiredFields = map[string][]string{
{{- range .}}
{{range .}}	"{{.Pattern}}": {{.RequiredFields}},
{{end}}{{end -}}
}
`

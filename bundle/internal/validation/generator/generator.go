package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/databricks/cli/libs/jsonschema"
)

// Generate creates the Go map from the schema.
func Generate(ctx context.Context, rootSchema *jsonschema.Schema, outputPath string) error {
	if rootSchema == nil {
		return fmt.Errorf("root schema provided to generator is nil")
	}
	nodes, err := parseSchema(rootSchema)
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

// NodeInfo holds the path and required status for a schema node.
type NodeInfo struct {
	Path      string
	Required  string
	GroupName string
}

type walker struct {
	// References resolved so far while walking the JSON schema.
	// We keep track to avoid getting trapped in a circular loop.
	refsPath map[string]bool

	nodes []NodeInfo

	originalSchema *jsonschema.Schema
}

func (w *walker) walk(p string, curr *jsonschema.Schema) error {
	if curr == nil {
		return nil
	}

	// "environments" has been deprecated for a while. No need to perform validation for it.
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

		w.nodes = append(w.nodes, NodeInfo{
			Path:     p,
			Required: r,
		})
	}

	// Case 1: Resolve references.
	if curr.Reference != nil && *curr.Reference != "" {
		refString := *curr.Reference
		if w.refsPath[refString] {
			// We've already seen and visited this $ref, so we can skip it.
			return nil
		}

		w.refsPath[refString] = true

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
		w.refsPath[refString] = false
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

// parseSchema recursively traverses the schema and extracts required fields.
func parseSchema(currentRootSchema *jsonschema.Schema) ([][]NodeInfo, error) {
	walker := &walker{
		refsPath:       make(map[string]bool),
		originalSchema: currentRootSchema,
	}

	err := walker.walk("", currentRootSchema)
	if err != nil {
		return nil, err
	}

	// Group the required fields by the name of their top level field. This allow for better rendering.
	nodes := walker.nodes
	nodeMap := map[string][]NodeInfo{}
	for _, node := range nodes {
		parts := strings.Split(node.Path, ".")

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

		if _, ok := nodeMap[k]; ok {
			nodeMap[k] = append(nodeMap[k], node)
		} else {
			nodeMap[k] = []NodeInfo{node}
		}
	}

	// Convert map to an array to make it easier to render.
	sortedNodeMap := make([][]NodeInfo, 0, len(nodeMap))
	sortedKeys := []string{}
	for k := range nodeMap {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)
	for _, k := range sortedKeys {
		sortedNodeMap = append(sortedNodeMap, nodeMap[k])
	}

	return sortedNodeMap, nil
}

// validationTemplate is the Go text template for generating the map.
const validationTemplate = `package generated

// THIS FILE IS AUTOGENERATED.
// DO NOT EDIT THIS FILE DIRECTLY.

// RequiredFields maps schema paths to their required status.
var RequiredFields = map[string][]string{
{{- range .}}
{{range .}}	"{{.Path}}": {{.Required}},
{{end}}{{end -}}
}
`

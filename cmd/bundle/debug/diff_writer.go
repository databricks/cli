package debug

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/r3labs/diff/v3"
	"gopkg.in/yaml.v3"
)

// DiffWriter handles writing diff changes back to YAML files
type DiffWriter struct {
	bundle *bundle.Bundle
}

// NewDiffWriter creates a new DiffWriter
func NewDiffWriter(b *bundle.Bundle) *DiffWriter {
	return &DiffWriter{bundle: b}
}

// WriteJobDiff writes job diff changes back to the YAML file
func (w *DiffWriter) WriteJobDiff(ctx context.Context, jobKey string, currentState any, changelog diff.Changelog) error {
	return w.writeResourceDiff(ctx, "jobs", jobKey, currentState, changelog, extractJobSettings)
}

// WritePipelineDiff writes pipeline diff changes back to the YAML file
func (w *DiffWriter) WritePipelineDiff(ctx context.Context, pipelineKey string, currentState any, changelog diff.Changelog) error {
	return w.writeResourceDiff(ctx, "pipelines", pipelineKey, currentState, changelog, extractPipelineSpec)
}

// extractorFunc extracts the relevant settings from the full API response
type extractorFunc func(any) (any, error)

// extractJobSettings extracts JobSettings from a Job response
func extractJobSettings(state any) (any, error) {
	job, ok := state.(*jobs.Job)
	if !ok {
		return nil, fmt.Errorf("expected *jobs.Job, got %T", state)
	}
	if job.Settings == nil {
		return nil, errors.New("job settings is nil")
	}
	return job.Settings, nil
}

// extractPipelineSpec extracts PipelineSpec from a Pipeline response
func extractPipelineSpec(state any) (any, error) {
	pipeline, ok := state.(*pipelines.GetPipelineResponse)
	if !ok {
		return nil, fmt.Errorf("expected *pipelines.GetPipelineResponse, got %T", state)
	}
	if pipeline.Spec == nil {
		return nil, errors.New("pipeline spec is nil")
	}
	return pipeline.Spec, nil
}

// unwrapSettingsPath removes the "Settings" or "Spec" wrapper from the path
// SDK responses have job.Settings.Field, but YAML has jobs.my_job.field
func unwrapSettingsPath(path []string) []string {
	if len(path) > 0 && (path[0] == "Settings" || path[0] == "Spec") {
		return path[1:]
	}
	return path
}

// convertChangePathToDynPath converts a changelog path to a dyn.Path
// It handles converting SDK struct field names to JSON tag names using reflection
func convertChangePathToDynPath(path []string, structType reflect.Type) (dyn.Path, error) {
	var dynPath dyn.Path
	currentType := structType

	for _, segment := range path {
		if currentType.Kind() == reflect.Ptr {
			currentType = currentType.Elem()
		}

		if currentType.Kind() != reflect.Struct {
			// For non-struct types (maps, slices, etc.), use the segment as-is
			dynPath = dynPath.Append(dyn.Key(segment))
			continue
		}

		// Find the field in the struct
		field, found := currentType.FieldByName(segment)
		if !found {
			return nil, fmt.Errorf("field %s not found in type %s", segment, currentType.Name())
		}

		// Get the JSON tag name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			// No JSON tag, use lowercase field name
			jsonTag = strings.ToLower(segment)
		} else {
			// Parse the JSON tag (it may have options like "omitempty")
			parts := strings.Split(jsonTag, ",")
			jsonTag = parts[0]
		}

		dynPath = dynPath.Append(dyn.Key(jsonTag))
		currentType = field.Type
	}

	return dynPath, nil
}

// ensurePathExists ensures all intermediate path segments exist before setting a value
// If an intermediate path doesn't exist, it creates an empty mapping at that location
func ensurePathExists(ctx context.Context, v dyn.Value, path dyn.Path) (dyn.Value, error) {
	// Build the path incrementally and ensure each level exists
	for i := range path {
		intermediatePath := path[:i+1]

		// Check if this path exists
		item, err := dyn.GetByPath(v, intermediatePath)
		if err != nil || !item.IsValid() {
			// Path doesn't exist, create an empty mapping
			log.Debugf(ctx, "Creating intermediate path: %s", intermediatePath)
			v, err = dyn.SetByPath(v, intermediatePath, dyn.V(dyn.NewMapping()))
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("failed to create intermediate path %s: %w", intermediatePath, err)
			}
		}
	}

	return v, nil
}

// writeResourceDiff writes resource diff changes back to the YAML file
func (w *DiffWriter) writeResourceDiff(ctx context.Context, resourceType, resourceKey string, currentState any, changelog diff.Changelog, extractor extractorFunc) error {
	// Build the path to the resource in the bundle config
	resourcePath := dyn.MustPathFromString(fmt.Sprintf("resources.%s.%s", resourceType, resourceKey))

	// Get the current config value for this resource
	resourceValue, err := dyn.GetByPath(w.bundle.Config.Value(), resourcePath)
	if err != nil {
		return fmt.Errorf("failed to get resource at path %s: %w", resourcePath, err)
	}

	// Get the file location for this resource
	location := resourceValue.Location()
	if location.File == "" {
		return fmt.Errorf("resource %s.%s has no file location", resourceType, resourceKey)
	}

	log.Infof(ctx, "Updating %s.%s in %s with %d changes", resourceType, resourceKey, location.File, len(changelog))

	// Extract the relevant settings from the API response
	// (e.g., JobSettings from Job, PipelineSpec from Pipeline)
	settings, err := extractor(currentState)
	if err != nil {
		return fmt.Errorf("failed to extract settings from current state: %w", err)
	}

	// Convert the entire remote settings to dyn.Value so we can extract specific field values
	remoteValue, err := convert.FromTyped(settings, resourceValue)
	if err != nil {
		return fmt.Errorf("failed to convert settings to dyn.Value: %w", err)
	}

	log.Debugf(ctx, "Converted remote state to dyn.Value")

	// Read the YAML file
	content, err := os.ReadFile(location.File)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", location.File, err)
	}

	// Parse the YAML file to dyn.Value
	fileValue, err := yamlloader.LoadYAML(location.File, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("failed to parse YAML file %s: %w", location.File, err)
	}

	// Get the struct type for path conversion
	settingsType := reflect.TypeOf(settings)
	if settingsType.Kind() == reflect.Ptr {
		settingsType = settingsType.Elem()
	}

	// Apply each change from the changelog
	updatedFileValue := fileValue
	for _, change := range changelog {
		// Unwrap Settings/Spec wrapper from the path
		unwrappedPath := unwrapSettingsPath(change.Path)
		if len(unwrappedPath) == 0 {
			log.Debugf(ctx, "Skipping empty path after unwrapping")
			continue
		}

		// Convert to dyn.Path with JSON tag names
		dynPath, err := convertChangePathToDynPath(unwrappedPath, settingsType)
		if err != nil {
			log.Warnf(ctx, "Failed to convert path %v: %v", change.Path, err)
			continue
		}

		// Prepend the resource path
		fullPath := resourcePath.Append(dynPath...)

		log.Debugf(ctx, "Applying change %s at path %s", change.Type, fullPath)

		// Apply the change based on type
		switch change.Type {
		case "create", "update":
			// Extract the value from the remote state
			fieldValue, err := dyn.GetByPath(remoteValue, dynPath)
			if err != nil {
				log.Warnf(ctx, "Failed to get value at path %s: %v", dynPath, err)
				continue
			}

			// Ensure all intermediate paths exist before setting the value
			updatedFileValue, err = ensurePathExists(ctx, updatedFileValue, fullPath)
			if err != nil {
				log.Warnf(ctx, "Failed to ensure path exists %s: %v", fullPath, err)
				continue
			}

			// Update the file value
			updatedFileValue, err = dyn.SetByPath(updatedFileValue, fullPath, fieldValue)
			if err != nil {
				log.Warnf(ctx, "Failed to set value at path %s: %v", fullPath, err)
				continue
			}

		case "delete":
			// For delete operations, we need to manually manipulate the mapping
			// since dyn doesn't have a DeleteByPath function
			log.Debugf(ctx, "Skipping delete operation for path %s (not yet implemented)", fullPath)
			// TODO: Implement deletion by reconstructing the parent mapping without the key
		}
	}

	// Write the updated file
	return w.writeYAMLFile(ctx, location.File, updatedFileValue)
}

// writeYAMLFile writes a dyn.Value to a YAML file
func (w *DiffWriter) writeYAMLFile(ctx context.Context, filePath string, fileValue dyn.Value) error {
	// Create directory if needed
	err := os.MkdirAll(filepath.Dir(filePath), 0o755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()

	// Convert to yaml.Node directly from dyn.Value
	yamlNode, err := dynValueToYamlNode(fileValue)
	if err != nil {
		return fmt.Errorf("failed to convert to YAML node: %w", err)
	}

	// Write the YAML
	enc := yaml.NewEncoder(file)
	enc.SetIndent(2)
	err = enc.Encode(yamlNode)
	if err != nil {
		return fmt.Errorf("failed to write YAML: %w", err)
	}

	log.Infof(ctx, "Successfully updated %s", filePath)
	return nil
}

// dynValueToYamlNode converts a dyn.Value to a yaml.Node
// This is similar to yamlsaver.toYamlNode but handles our use case
func dynValueToYamlNode(v dyn.Value) (*yaml.Node, error) {
	return dynValueToYamlNodeWithStyle(v, yaml.Style(0), nil)
}

func dynValueToYamlNodeWithStyle(v dyn.Value, style yaml.Style, stylesMap map[string]yaml.Style) (*yaml.Node, error) {
	switch v.Kind() {
	case dyn.KindMap:
		m, _ := v.AsMap()
		var content []*yaml.Node

		// Sort by location line number to preserve order
		pairs := m.Pairs()
		for _, pair := range pairs {
			pk := pair.Key
			pv := pair.Value
			keyNode := yaml.Node{Kind: yaml.ScalarNode, Value: pk.MustString(), Style: style}

			// Check if this key has a custom style
			var nestedStyle yaml.Style
			if stylesMap != nil {
				if customStyle, ok := stylesMap[pk.MustString()]; ok {
					nestedStyle = customStyle
				} else {
					nestedStyle = style
				}
			} else {
				nestedStyle = style
			}

			valueNode, err := dynValueToYamlNodeWithStyle(pv, nestedStyle, stylesMap)
			if err != nil {
				return nil, err
			}
			content = append(content, &keyNode)
			content = append(content, valueNode)
		}
		return &yaml.Node{Kind: yaml.MappingNode, Content: content, Style: style}, nil

	case dyn.KindSequence:
		seq, _ := v.AsSequence()
		var content []*yaml.Node
		for _, item := range seq {
			node, err := dynValueToYamlNodeWithStyle(item, style, stylesMap)
			if err != nil {
				return nil, err
			}
			content = append(content, node)
		}
		return &yaml.Node{Kind: yaml.SequenceNode, Content: content, Style: style}, nil

	case dyn.KindNil:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: "null", Style: style}, nil

	case dyn.KindString:
		s := v.MustString()
		// Quote strings that look like scalars
		if isScalarLikeString(s) {
			return &yaml.Node{Kind: yaml.ScalarNode, Value: s, Style: yaml.DoubleQuotedStyle}, nil
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Value: s, Style: style}, nil

	case dyn.KindBool:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: strconv.FormatBool(v.MustBool()), Style: style}, nil

	case dyn.KindInt:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: strconv.FormatInt(v.MustInt(), 10), Style: style}, nil

	case dyn.KindFloat:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprint(v.MustFloat()), Style: style}, nil

	case dyn.KindTime:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: v.MustTime().String(), Style: style}, nil

	default:
		return nil, fmt.Errorf("unsupported kind: %s", v.Kind())
	}
}

func isScalarLikeString(s string) bool {
	if s == "" || s == "true" || s == "false" {
		return true
	}
	// Check if it's a number
	if _, err := strconv.ParseInt(s, 0, 64); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}

package debug

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
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
func (w *DiffWriter) WriteJobDiff(ctx context.Context, jobKey string, currentState any) error {
	return w.writeResourceDiff(ctx, "jobs", jobKey, currentState, extractJobSettings)
}

// WritePipelineDiff writes pipeline diff changes back to the YAML file
func (w *DiffWriter) WritePipelineDiff(ctx context.Context, pipelineKey string, currentState any) error {
	return w.writeResourceDiff(ctx, "pipelines", pipelineKey, currentState, extractPipelineSpec)
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

// writeResourceDiff writes resource diff changes back to the YAML file
func (w *DiffWriter) writeResourceDiff(ctx context.Context, resourceType, resourceKey string, currentState any, extractor extractorFunc) error {
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

	log.Infof(ctx, "Updating %s.%s in %s", resourceType, resourceKey, location.File)

	// Extract the relevant settings from the API response
	// (e.g., JobSettings from Job, PipelineSpec from Pipeline)
	settings, err := extractor(currentState)
	if err != nil {
		return fmt.Errorf("failed to extract settings from current state: %w", err)
	}

	// Convert the settings to dyn.Value, using the current resource value as reference
	// to preserve locations and structure
	updatedValue, err := convert.FromTyped(settings, resourceValue)
	if err != nil {
		return fmt.Errorf("failed to convert settings to dyn.Value: %w", err)
	}

	log.Debugf(ctx, "Converted remote state to dyn.Value")

	// Update the YAML file with the new resource value
	return w.updateYAMLFile(ctx, location.File, resourcePath, updatedValue)
}

// updateYAMLFile updates a specific resource in a YAML file
func (w *DiffWriter) updateYAMLFile(ctx context.Context, filePath string, resourcePath dyn.Path, newValue dyn.Value) error {
	// Read the existing file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Parse the YAML file to dyn.Value
	fileValue, err := yamlloader.LoadYAML(filePath, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("failed to parse YAML file %s: %w", filePath, err)
	}

	// Update the resource in the file value
	updatedFileValue, err := dyn.SetByPath(fileValue, resourcePath, newValue)
	if err != nil {
		return fmt.Errorf("failed to update resource at path %s: %w", resourcePath, err)
	}

	// Write back to the file using the internal encode method
	// We need to write the file manually since SaveAsYAML expects .AsAny()
	// but our value may contain types that dyn.V() can't handle
	err = os.MkdirAll(filepath.Dir(filePath), 0o755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()

	// Convert to yaml.Node directly from dyn.Value
	yamlNode, err := dynValueToYamlNode(updatedFileValue)
	if err != nil {
		return fmt.Errorf("failed to convert to YAML node: %w", err)
	}

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

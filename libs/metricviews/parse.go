package metricviews

import (
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"
)

const (
	version01            = "0.1"
	version10            = "1.0"
	version11            = "1.1"
	viewTypeSingleSource = "SINGLE_SOURCE"
	viewTypeMultiSource  = "MULTI_SOURCE"
)

// MetricView is a parsed metric view. Exactly one variant pointer is non-nil.
type MetricView struct {
	V10          *MetricViewV10
	SingleSource *SingleSourceMetricView
	MultiSource  *MultiSourceMetricView
}

// Version returns the declared YAML version.
func (m *MetricView) Version() string {
	switch {
	case m.V10 != nil:
		return m.V10.Version
	case m.SingleSource != nil:
		return m.SingleSource.Version
	case m.MultiSource != nil:
		return m.MultiSource.Version
	default:
		return ""
	}
}

// MarshalYAML returns the active variant for the YAML encoder.
func (m *MetricView) MarshalYAML() (any, error) {
	switch {
	case m.V10 != nil:
		return m.V10, nil
	case m.SingleSource != nil:
		return m.SingleSource, nil
	case m.MultiSource != nil:
		return m.MultiSource, nil
	default:
		return nil, errors.New("metricviews: empty MetricView")
	}
}

// peek reads the top-level "version" and "view_type" scalars, plus the
// presence of "sources", without type-coercing scalars (version may be an
// unquoted float).
func peek(root *yaml.Node) (version, viewType string, hasSources bool) {
	doc := root
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		doc = doc.Content[0]
	}
	if doc.Kind != yaml.MappingNode {
		return "", "", false
	}
	for i := 0; i+1 < len(doc.Content); i += 2 {
		switch doc.Content[i].Value {
		case "version":
			version = yamlScalar(doc.Content[i+1])
		case "view_type":
			viewType = yamlScalar(doc.Content[i+1])
		case "sources":
			hasSources = true
		}
	}
	return version, viewType, hasSources
}

// Parse decodes YAML definitions without checking structural rules. Use
// ParseAndValidate for user input that must satisfy those rules.
func Parse(data []byte) (*MetricView, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("metricviews: parsing YAML: %w", err)
	}
	return decodeMetricView(&root)
}

func decodeMetricView(root *yaml.Node) (*MetricView, error) {
	version, viewType, hasSources := peek(root)

	switch version {
	case version01, version10:
		var v MetricViewV10
		if err := root.Decode(&v); err != nil {
			return nil, fmt.Errorf("metricviews: decoding v1.0: %w", err)
		}
		return &MetricView{V10: &v}, nil
	default:
		if viewType == viewTypeMultiSource || hasSources {
			var v MultiSourceMetricView
			if err := root.Decode(&v); err != nil {
				return nil, fmt.Errorf("metricviews: decoding v1.1 multi-source: %w", err)
			}
			return &MetricView{MultiSource: &v}, nil
		}
		var v SingleSourceMetricView
		if err := root.Decode(&v); err != nil {
			return nil, fmt.Errorf("metricviews: decoding v1.1 single-source: %w", err)
		}
		return &MetricView{SingleSource: &v}, nil
	}
}

package metricviews

import (
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"
)

// Validate checks the structural rules in a parsed YAML document or mapping.
// It uses YAML nodes so missing and duplicate keys remain observable.
func Validate(node *yaml.Node) error {
	root := node
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return errors.New("metricviews: missing YAML document")
		}
		root = node.Content[0]
	}
	if err := requireYAMLFields(root, "version"); err != nil {
		return fmt.Errorf("metricviews: %w", err)
	}
	version, viewType, hasSources := peek(root)
	switch version {
	case version01, version10:
		return validateV10(root)
	case version11:
		if yamlField(root, "view_type") != nil && viewType != viewTypeSingleSource && viewType != viewTypeMultiSource {
			return fmt.Errorf("metricviews: unsupported v1.1 view_type %q", viewType)
		}
		if viewType == viewTypeMultiSource {
			return validateMultiSource(root)
		}
		if hasSources {
			return errors.New("metricviews: v1.1 sources requires view_type: MULTI_SOURCE")
		}
		return validateSingleSource(root)
	default:
		return fmt.Errorf("metricviews: invalid YAML version: %q", version)
	}
}

// ParseAndValidate decodes a user-supplied definition after checking its shape.
func ParseAndValidate(data []byte) (*MetricView, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("metricviews: parsing YAML: %w", err)
	}
	if err := Validate(&root); err != nil {
		return nil, err
	}
	return decodeMetricView(&root)
}

func validateV10(node *yaml.Node) error {
	if err := requireYAMLFields(node, "version", "source"); err != nil {
		return err
	}
	if err := validateDimensionColumns(node, validateColumnV10); err != nil {
		return err
	}
	if err := visitYAMLSequence(node, "measures", validateColumnV10); err != nil {
		return err
	}
	return validateSingleSourceDetails(node, validateParameterV10)
}

func validateSingleSource(node *yaml.Node) error {
	if err := requireYAMLFields(node, "version", "source"); err != nil {
		return err
	}
	if err := validateDimensionColumns(node, validateColumnV11); err != nil {
		return err
	}
	if err := visitYAMLSequence(node, "measures", validateColumnV11); err != nil {
		return err
	}
	return validateSingleSourceDetails(node, validateParameterV11)
}

func validateSingleSourceDetails(node *yaml.Node, parameter func(*yaml.Node) error) error {
	if err := visitYAMLSequence(node, "joins", validateJoin); err != nil {
		return err
	}
	if err := visitYAMLSequence(node, "parameters", parameter); err != nil {
		return err
	}
	return visitYAMLField(node, "materialization", validateMaterialization)
}

func validateMultiSource(node *yaml.Node) error {
	if err := requireYAMLFields(node, "version", "sources"); err != nil {
		return err
	}
	if err := visitYAMLSequence(node, "sources", validateSourceNode); err != nil {
		return err
	}
	if err := validateDimensionColumns(node, validateColumnV11); err != nil {
		return err
	}
	if err := visitYAMLSequence(node, "measures", validateColumnV11); err != nil {
		return err
	}
	return visitYAMLSequence(node, "parameters", validateParameterV11)
}

func validateDimensionColumns(node *yaml.Node, column func(*yaml.Node) error) error {
	if err := rejectDimensionFieldConflict(node); err != nil {
		return err
	}
	field := "dimensions"
	if yamlField(node, "fields") != nil {
		field = "fields"
	}
	return visitYAMLSequence(node, field, column)
}

func validateColumnV10(node *yaml.Node) error {
	if err := requireYAMLFields(node, "name", "expr"); err != nil {
		return err
	}
	return validateColumnDetails(node)
}

func validateColumnV11(node *yaml.Node) error {
	if err := requireYAMLFields(node, "expr"); err != nil {
		return err
	}
	return validateColumnDetails(node)
}

func validateColumnDetails(node *yaml.Node) error {
	if err := visitYAMLSequence(node, "window", validateWindow); err != nil {
		return err
	}
	return visitYAMLField(node, "format", validateFormat)
}

func validateWindow(node *yaml.Node) error {
	return requireYAMLFields(node, "order", "semiadditive", "range")
}

func validateJoin(node *yaml.Node) error {
	if err := requireYAMLFields(node, "name", "source"); err != nil {
		return err
	}
	return visitYAMLSequence(node, "joins", validateJoin)
}

func validateMaterialization(node *yaml.Node) error {
	if err := requireYAMLFields(node, "schedule", "mode", "materialized_views"); err != nil {
		return err
	}
	return visitYAMLSequence(node, "materialized_views", validateMaterializedView)
}

func validateMaterializedView(node *yaml.Node) error {
	if err := requireYAMLFields(node, "name", "type"); err != nil {
		return err
	}
	return rejectDimensionFieldConflict(node)
}

func validateParameterV10(node *yaml.Node) error {
	return requireYAMLFields(node, "name", "data_type")
}

func validateParameterV11(node *yaml.Node) error {
	if err := requireYAMLFields(node, "name", "data_type"); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		switch key {
		case "name", "data_type", "default":
			if seen[key] {
				return fmt.Errorf("duplicate parameter field %q", key)
			}
			seen[key] = true
		default:
			return fmt.Errorf("unknown parameter field %q", key)
		}
	}
	return nil
}

func validateSourceNode(node *yaml.Node) error {
	if err := requireYAMLFields(node, "name", "from"); err != nil {
		return err
	}
	return visitYAMLSequence(node, "relationships", validateRelationship)
}

func validateRelationship(node *yaml.Node) error {
	return requireYAMLFields(node, "ref")
}

func validateFormat(node *yaml.Node) error {
	if err := requireYAMLFields(node, "type"); err != nil {
		return err
	}
	typ := yamlScalar(yamlField(node, "type"))
	allowed := map[string]bool{"type": true}
	var required []string
	switch typ {
	case "number":
		allowed["decimal_places"], allowed["hide_group_separator"], allowed["abbreviation"] = true, true, true
	case "currency":
		allowed["decimal_places"], allowed["hide_group_separator"], allowed["abbreviation"], allowed["currency_code"] = true, true, true, true
		required = []string{"currency_code"}
	case "percentage", "byte":
		allowed["decimal_places"], allowed["hide_group_separator"] = true, true
	case "date":
		allowed["date_format"], allowed["leading_zeros"] = true, true
		required = []string{"date_format"}
	case "date_time":
		allowed["date_format"], allowed["time_format"], allowed["leading_zeros"] = true, true, true
		required = []string{"date_format", "time_format"}
	default:
		return fmt.Errorf("unsupported format type %q", typ)
	}
	if err := requireYAMLFields(node, required...); err != nil {
		return err
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		field := node.Content[i].Value
		if !allowed[field] {
			return fmt.Errorf("field %q is not valid for format type %q", field, typ)
		}
	}
	return visitYAMLField(node, "decimal_places", validateDecimalPlaces)
}

func validateDecimalPlaces(node *yaml.Node) error {
	if err := requireYAMLFields(node, "type"); err != nil {
		return err
	}
	typ := yamlScalar(yamlField(node, "type"))
	switch typ {
	case "max", "exact", "all":
		return nil
	default:
		return fmt.Errorf("unsupported decimal_places.type %q", typ)
	}
}

func visitYAMLField(node *yaml.Node, field string, visit func(*yaml.Node) error) error {
	value := yamlField(node, field)
	if value == nil {
		return nil
	}
	value = resolveYAMLAlias(value)
	if value.Tag == "!!null" {
		return nil
	}
	if err := visit(value); err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	return nil
}

func visitYAMLSequence(node *yaml.Node, field string, visit func(*yaml.Node) error) error {
	return visitYAMLField(node, field, func(value *yaml.Node) error {
		if value.Kind != yaml.SequenceNode {
			return fmt.Errorf("expected a sequence, got kind %d", value.Kind)
		}
		for i, item := range value.Content {
			if err := visit(resolveYAMLAlias(item)); err != nil {
				return fmt.Errorf("item %d: %w", i, err)
			}
		}
		return nil
	})
}

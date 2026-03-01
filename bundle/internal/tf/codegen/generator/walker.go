package generator

import (
	"fmt"
	"slices"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/iancoleman/strcase"
	"github.com/zclconf/go-cty/cty"
)

type field struct {
	Name string
	Type string
	Tag  string
}

type structType struct {
	Name   string
	Fields []field
}

// walker represents the set of types to declare to
// represent a [tfjson.SchemaBlock] as Go structs.
// See the [walk] function for usage.
type walker struct {
	StructTypes  []structType
	resourceName string // The Terraform resource name (e.g., "databricks_postgres_endpoint")
}

// floatAttributePaths is an allowlist of attribute paths (resource_name.field_path)
// that should be mapped to float64 instead of int for cty.Number types.
// Generated from Terraform provider schema based on databricks-sdk-go types.
var floatAttributePaths = map[string]bool{
	// Alert thresholds - double_value fields hold fractional values (e.g., 1.3)
	"databricks_alert_v2.evaluation.threshold.value.double_value": true,

	// Postgres Service - autoscaling compute units support fractional values (e.g., 0.5 CU)
	"databricks_postgres_endpoint.spec.autoscaling_limit_max_cu":                                      true,
	"databricks_postgres_endpoint.spec.autoscaling_limit_min_cu":                                      true,
	"databricks_postgres_endpoint.status.autoscaling_limit_max_cu":                                    true,
	"databricks_postgres_endpoint.status.autoscaling_limit_min_cu":                                    true,
	"databricks_postgres_endpoints.endpoints.spec.autoscaling_limit_max_cu":                           true,
	"databricks_postgres_endpoints.endpoints.spec.autoscaling_limit_min_cu":                           true,
	"databricks_postgres_endpoints.endpoints.status.autoscaling_limit_max_cu":                         true,
	"databricks_postgres_endpoints.endpoints.status.autoscaling_limit_min_cu":                         true,
	"databricks_postgres_project.spec.default_endpoint_settings.autoscaling_limit_max_cu":             true,
	"databricks_postgres_project.spec.default_endpoint_settings.autoscaling_limit_min_cu":             true,
	"databricks_postgres_project.status.default_endpoint_settings.autoscaling_limit_max_cu":           true,
	"databricks_postgres_project.status.default_endpoint_settings.autoscaling_limit_min_cu":           true,
	"databricks_postgres_projects.projects.spec.default_endpoint_settings.autoscaling_limit_max_cu":   true,
	"databricks_postgres_projects.projects.spec.default_endpoint_settings.autoscaling_limit_min_cu":   true,
	"databricks_postgres_projects.projects.status.default_endpoint_settings.autoscaling_limit_max_cu": true,
	"databricks_postgres_projects.projects.status.default_endpoint_settings.autoscaling_limit_min_cu": true,
}

// buildAttributePath constructs the attribute path from the type name path.
// For example: ["Resource", "PostgresEndpoint", "Spec"] + "autoscaling_limit_min_cu" -> "spec.autoscaling_limit_min_cu"
func buildAttributePath(name []string, fieldName string) string {
	// Skip the type prefix (Resource/DataSource) and resource name (first 2 elements)
	// and convert remaining CamelCase to snake_case
	var parts []string
	for i := 2; i < len(name); i++ {
		parts = append(parts, toSnakeCase(name[i]))
	}
	parts = append(parts, fieldName)
	return strings.Join(parts, ".")
}

// toSnakeCase converts CamelCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func processAttributeType(typ cty.Type, resourceName, attributePath string) string {
	var out string

	switch {
	case typ.IsPrimitiveType():
		switch {
		case typ.Equals(cty.Bool):
			out = "bool"
		case typ.Equals(cty.Number):
			// Check if this resource + attribute path should be float64
			fullPath := resourceName + "." + attributePath
			if floatAttributePaths[fullPath] {
				out = "float64"
			} else {
				out = "int"
			}
		case typ.Equals(cty.String):
			out = "string"
		default:
			panic("No idea what to do for: " + typ.FriendlyName())
		}
	case typ.IsMapType():
		out = "map[string]" + processAttributeType(*typ.MapElementType(), resourceName, attributePath)
	case typ.IsSetType():
		out = "[]" + processAttributeType(*typ.SetElementType(), resourceName, attributePath)
	case typ.IsListType():
		out = "[]" + processAttributeType(*typ.ListElementType(), resourceName, attributePath)
	case typ.IsObjectType():
		out = "any"
	default:
		panic("No idea what to do for: " + typ.FriendlyName())
	}

	return out
}

func nestedBlockKeys(block *tfjson.SchemaBlock) []string {
	keys := sortKeys(block.NestedBlocks)

	// Remove TF specific "timeouts" block.
	if i := slices.Index(keys, "timeouts"); i != -1 {
		keys = slices.Delete(keys, i, i+1)
	}

	return keys
}

func nestedField(name []string, k string, isRef bool) field {
	// Collect field properties.
	fieldName := strcase.ToCamel(k)
	fieldTypePrefix := ""
	if isRef {
		fieldTypePrefix = "*"
	} else {
		fieldTypePrefix = "[]"
	}
	fieldType := fmt.Sprintf("%s%s", fieldTypePrefix, strings.Join(append(name, strcase.ToCamel(k)), ""))
	fieldTag := fmt.Sprintf("%s,omitempty", k)

	return field{
		Name: fieldName,
		Type: fieldType,
		Tag:  fieldTag,
	}
}

func (w *walker) walk(block *tfjson.SchemaBlock, name []string) error {
	// Produce nested types before this block itself.
	// This ensures types are defined before they are referenced.
	for _, k := range nestedBlockKeys(block) {
		v := block.NestedBlocks[k]
		err := w.walk(v.Block, append(name, strcase.ToCamel(k)))
		if err != nil {
			return err
		}
	}

	// Declare type.
	typ := structType{
		Name: strings.Join(name, ""),
	}

	// Declare attributes.
	for _, k := range sortKeys(block.Attributes) {
		v := block.Attributes[k]

		// Assert the attribute type is always set.
		if v.AttributeType == cty.NilType && v.AttributeNestedType == nil {
			return fmt.Errorf("unexpected nil type for attribute %s", k)
		}

		// If there is a nested type, walk it and continue to next attribute.
		if v.AttributeNestedType != nil {
			nestedBlock := &tfjson.SchemaBlock{
				Attributes: v.AttributeNestedType.Attributes,
			}
			err := w.walk(nestedBlock, append(name, strcase.ToCamel(k)))
			if err != nil {
				return err
			}
			// Append to list of fields for type.
			typ.Fields = append(typ.Fields, nestedField(name, k, v.AttributeNestedType.NestingMode == tfjson.SchemaNestingModeSingle))
			continue
		}

		// Collect field properties.
		fieldName := strcase.ToCamel(k)
		attributePath := buildAttributePath(name, k)
		fieldType := processAttributeType(v.AttributeType, w.resourceName, attributePath)
		fieldTag := k
		if v.Required && v.Optional {
			return fmt.Errorf("both required and optional are set for attribute %s", k)
		}
		if !v.Required {
			fieldTag = fmt.Sprintf("%s,omitempty", fieldTag)
		}

		// Append to list of fields for type.
		typ.Fields = append(typ.Fields, field{
			Name: fieldName,
			Type: fieldType,
			Tag:  fieldTag,
		})
	}

	// Declare nested blocks.
	for _, k := range nestedBlockKeys(block) {
		v := block.NestedBlocks[k]
		// Append to list of fields for type.
		typ.Fields = append(typ.Fields, nestedField(name, k, v.MaxItems == 1))
	}

	// Append type to list of structs.
	w.StructTypes = append(w.StructTypes, typ)
	return nil
}

// walk recursively traverses [tfjson.SchemaBlock] and returns the
// set of types to declare to represents it as Go structs.
// The resourceName parameter is the Terraform resource name (e.g., "databricks_postgres_endpoint").
func walk(block *tfjson.SchemaBlock, name []string, resourceName string) (*walker, error) {
	w := &walker{resourceName: resourceName}
	err := w.walk(block, name)
	return w, err
}

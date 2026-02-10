package generator

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/apps/manifest"
)

// Config holds configuration values collected from user prompts.
type Config struct {
	ProjectName   string
	WorkspaceHost string
	Profile       string
	// ResourceValues maps resource value keys to values.
	// Keys use "resource_key.field_name" format (e.g., "sql-warehouse.id" -> "abc123").
	ResourceValues map[string]string
}

// hasResourceValues returns true if any value exists in cfg for the given resource.
func hasResourceValues(r manifest.Resource, cfg Config) bool {
	for _, v := range variableNamesForResource(r) {
		if cfg.ResourceValues[v.valueKey] != "" {
			return true
		}
	}
	return false
}

// GenerateBundleVariables generates the variables section for databricks.yml.
// Output is indented with 2 spaces for insertion under "variables:".
// Includes both required resources and optional resources that have values.
func GenerateBundleVariables(plugins []manifest.Plugin, cfg Config) string {
	var lines []string

	for _, p := range plugins {
		for _, r := range p.Resources.Required {
			lines = append(lines, generateVariableLines(r)...)
		}
		for _, r := range p.Resources.Optional {
			if hasResourceValues(r, cfg) {
				lines = append(lines, generateVariableLines(r)...)
			}
		}
	}

	return strings.Join(lines, "\n")
}

// generateVariableLines returns the variable definition lines for a resource.
// Multi-field resources (database, secret, genie_space) produce multiple variables.
func generateVariableLines(r manifest.Resource) []string {
	var lines []string
	for _, v := range variableNamesForResource(r) {
		lines = append(lines, fmt.Sprintf("  %s:", v.name))
		if v.description != "" {
			lines = append(lines, "    description: "+v.description)
		}
	}
	return lines
}

// varInfo holds a variable name, its description, and the key used to look up its value in ResourceValues.
type varInfo struct {
	name        string // variable name in databricks.yml (e.g., "cache_instance_name")
	description string
	valueKey    string // key in Config.ResourceValues (e.g., "cache.instance_name", "warehouse.id")
}

// GenerateBundleResources generates the resources section for databricks.yml (app resources).
// Output is indented with 8 spaces for insertion under "resources: [...app resources...]".
// Includes both required resources and optional resources that have values.
func GenerateBundleResources(plugins []manifest.Plugin, cfg Config) string {
	var blocks []string

	for _, p := range plugins {
		// Required resources
		for _, r := range p.Resources.Required {
			resource := generateResourceYAML(r, 8)
			if resource != "" {
				blocks = append(blocks, resource)
			}
		}
		// Optional resources (only if value provided)
		for _, r := range p.Resources.Optional {
			if hasResourceValues(r, cfg) {
				resource := generateResourceYAML(r, 8)
				if resource != "" {
					blocks = append(blocks, resource)
				}
			}
		}
	}

	return strings.Join(blocks, "\n")
}

// GenerateTargetVariables generates the dev target variables section for databricks.yml.
// Output is indented with 6 spaces for insertion under "targets: default: variables:".
// Includes both required resources and optional resources that have values.
func GenerateTargetVariables(plugins []manifest.Plugin, cfg Config) string {
	var lines []string

	for _, p := range plugins {
		for _, r := range p.Resources.Required {
			lines = append(lines, generateTargetVarLines(r, cfg)...)
		}
		for _, r := range p.Resources.Optional {
			if hasResourceValues(r, cfg) {
				lines = append(lines, generateTargetVarLines(r, cfg)...)
			}
		}
	}

	return strings.Join(lines, "\n")
}

// generateTargetVarLines returns the target variable assignment lines for a resource.
func generateTargetVarLines(r manifest.Resource, cfg Config) []string {
	var lines []string
	for _, v := range variableNamesForResource(r) {
		value := cfg.ResourceValues[v.valueKey]
		if value != "" {
			lines = append(lines, fmt.Sprintf("      %s: %s", v.name, value))
		}
	}
	return lines
}

// dotEnvActualLines returns .env lines with actual values from cfg.
func dotEnvActualLines(r manifest.Resource, cfg Config) []string {
	var lines []string
	for _, fieldName := range r.FieldNames() {
		field := r.Fields[fieldName]
		if field.Env == "" {
			continue
		}
		value := cfg.ResourceValues[r.Key()+"."+fieldName]
		lines = append(lines, fmt.Sprintf("%s=%s", field.Env, value))
	}
	return lines
}

// dotEnvExampleLines returns .env.example lines with placeholders.
func dotEnvExampleLines(r manifest.Resource, commented bool) []string {
	var lines []string
	for _, fieldName := range r.FieldNames() {
		field := r.Fields[fieldName]
		if field.Env == "" {
			continue
		}
		placeholder := "your_" + r.VarPrefix() + "_" + fieldName
		if commented {
			lines = append(lines, fmt.Sprintf("# %s=%s", field.Env, placeholder))
		} else {
			lines = append(lines, fmt.Sprintf("%s=%s", field.Env, placeholder))
		}
	}
	return lines
}

// GenerateDotEnv generates the .env file content with actual values.
// Includes both required resources and optional resources that have values.
func GenerateDotEnv(plugins []manifest.Plugin, cfg Config) string {
	var lines []string

	for _, p := range plugins {
		for _, r := range p.Resources.Required {
			lines = append(lines, dotEnvActualLines(r, cfg)...)
		}
		for _, r := range p.Resources.Optional {
			if hasResourceValues(r, cfg) {
				lines = append(lines, dotEnvActualLines(r, cfg)...)
			}
		}
	}

	return strings.Join(lines, "\n")
}

// GenerateDotEnvExample generates the .env.example file content with placeholders.
// Includes both required and optional resources (optional ones are commented out).
func GenerateDotEnvExample(plugins []manifest.Plugin) string {
	var lines []string

	for _, p := range plugins {
		for _, r := range p.Resources.Required {
			lines = append(lines, dotEnvExampleLines(r, false)...)
		}
		for _, r := range p.Resources.Optional {
			lines = append(lines, dotEnvExampleLines(r, true)...)
		}
	}

	return strings.Join(lines, "\n")
}

// defaultPermissions maps resource type to its default permission when none is specified.
var defaultPermissions = map[string]string{
	"sql_warehouse":       "CAN_USE",
	"job":                 "CAN_MANAGE_RUN",
	"serving_endpoint":    "CAN_QUERY",
	"secret":              "READ",
	"experiment":          "CAN_READ",
	"database":            "CAN_CONNECT_AND_CREATE",
	"genie_space":         "CAN_VIEW",
	"volume":              "READ_VOLUME",
	"uc_function":         "EXECUTE",
	"uc_connection":       "USE_CONNECTION",
	"vector_search_index": "CAN_USE",
	// TODO: uncomment when bundles support app as an app resource type.
	// "app": "CAN_USE",
}

// varNameForField returns the bundle variable name for a specific field of a resource.
// Uses VarPrefix (resource_key with hyphens replaced by underscores).
func varNameForField(r manifest.Resource, fieldName string) string {
	return r.VarPrefix() + "_" + fieldName
}

// singleVarName returns the variable name for a single-field resource.
// Uses the first field from Fields, or falls back to varPrefix_id.
func singleVarName(r manifest.Resource) string {
	names := r.FieldNames()
	if len(names) > 0 {
		return varNameForField(r, names[0])
	}
	return aliasToVarName(r.VarPrefix())
}

// variableNamesForResource returns the variable names that a resource type needs.
// Variable names are derived from VarPrefix (resource_key with hyphens as underscores).
// Value keys use Key() (resource_key) with field names.
func variableNamesForResource(r manifest.Resource) []varInfo {
	var vars []varInfo
	for _, fieldName := range r.FieldNames() {
		field := r.Fields[fieldName]
		desc := field.Description
		if desc == "" {
			desc = r.Description
		}
		vars = append(vars, varInfo{
			name:        varNameForField(r, fieldName),
			description: desc,
			valueKey:    r.Key() + "." + fieldName,
		})
	}
	if len(vars) > 0 {
		return vars
	}
	// Fallback for resources without explicit Fields.
	return []varInfo{
		{name: aliasToVarName(r.VarPrefix()), description: r.Description, valueKey: r.Key()},
	}
}

// generateResourceYAML generates YAML for a single app resource based on its type.
// Each resource type has its own field structure per the Databricks Apps schema.
// Variable references are derived from the resource's Fields via singleVarName/varNameForField.
func generateResourceYAML(r manifest.Resource, indent int) string {
	if r.Type == "" {
		return ""
	}

	permission := r.Permission
	if permission == "" {
		if def, ok := defaultPermissions[r.Type]; ok {
			permission = def
		} else {
			permission = "CAN_USE"
		}
	}

	pad := strings.Repeat(" ", indent)

	key := r.Key()

	switch r.Type {
	case "sql_warehouse":
		return fmt.Sprintf(`%s- name: %s
%s  sql_warehouse:
%s    id: ${var.%s}
%s    permission: %s`, pad, key, pad, pad, singleVarName(r), pad, permission)

	case "job":
		return fmt.Sprintf(`%s- name: %s
%s  job:
%s    id: ${var.%s}
%s    permission: %s`, pad, key, pad, pad, singleVarName(r), pad, permission)

	case "serving_endpoint":
		return fmt.Sprintf(`%s- name: %s
%s  serving_endpoint:
%s    name: ${var.%s}
%s    permission: %s`, pad, key, pad, pad, singleVarName(r), pad, permission)

	case "experiment":
		return fmt.Sprintf(`%s- name: %s
%s  experiment:
%s    experiment_id: ${var.%s}
%s    permission: %s`, pad, key, pad, pad, singleVarName(r), pad, permission)

	case "secret":
		return fmt.Sprintf(`%s- name: %s
%s  secret:
%s    scope: ${var.%s}
%s    key: ${var.%s}
%s    permission: %s`, pad, key, pad, pad, varNameForField(r, "scope"), pad, varNameForField(r, "key"), pad, permission)

	case "database":
		return fmt.Sprintf(`%s- name: %s
%s  database:
%s    instance_name: ${var.%s}
%s    database_name: ${var.%s}
%s    permission: %s`, pad, key, pad, pad, varNameForField(r, "instance_name"), pad, varNameForField(r, "database_name"), pad, permission)

	case "genie_space":
		return fmt.Sprintf(`%s- name: %s
%s  genie_space:
%s    name: %s
%s    space_id: ${var.%s}
%s    permission: %s`, pad, key, pad, pad, r.Alias, pad, varNameForField(r, "space_id"), pad, permission)

	case "volume", "uc_function", "uc_connection":
		securableType := ucSecurableType(r.Type)
		return fmt.Sprintf(`%s- name: %s
%s  uc_securable:
%s    securable_full_name: ${var.%s}
%s    securable_type: %s
%s    permission: %s`, pad, key, pad, pad, singleVarName(r), pad, securableType, pad, permission)

	case "vector_search_index":
		return fmt.Sprintf(`%s- name: %s
%s  vector_search_index:
%s    id: ${var.%s}
%s    permission: %s`, pad, key, pad, pad, singleVarName(r), pad, permission)

	// TODO: uncomment when bundles support app as an app resource type.
	// case "app":
	// 	return fmt.Sprintf(`%s- name: %s
	// %s  app:
	// %s    name: ${var.%s}
	// %s    permission: %s`, pad, key, pad, pad, singleVarName(r), pad, permission)

	default:
		return ""
	}
}

// ucSecurableType maps a manifest resource type to the uc_securable securable_type value.
func ucSecurableType(resourceType string) string {
	switch resourceType {
	case "volume":
		return "VOLUME"
	case "uc_function":
		return "FUNCTION"
	case "uc_connection":
		return "CONNECTION"
	default:
		return ""
	}
}

// aliasToVarName converts a variable prefix to a variable name by appending "_id".
// e.g., "sql_warehouse" -> "sql_warehouse_id"
func aliasToVarName(prefix string) string {
	if strings.HasSuffix(prefix, "_id") {
		return prefix
	}
	return prefix + "_id"
}

// GetSelectedPlugins returns plugins that match the given names.
func GetSelectedPlugins(m *manifest.Manifest, names []string) []manifest.Plugin {
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	var selected []manifest.Plugin
	for _, p := range m.GetPlugins() {
		if nameSet[p.Name] {
			selected = append(selected, p)
		}
	}
	return selected
}

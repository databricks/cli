package generator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/databricks/cli/libs/apps/manifest"
)

// validEnvVar matches safe environment variable names (letters, digits, underscores, starting with a letter or underscore).
var validEnvVar = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// yamlNeedsQuoting is true when a value contains characters that can break YAML parsing.
var yamlNeedsQuoting = regexp.MustCompile(`[:#\[\]{}&*!|>'"%@` + "`" + `\n\r\\]|^\s|\s$|^$`)

// quoteYAMLValue wraps a value in double quotes if it contains YAML-special characters.
func quoteYAMLValue(v string) string {
	if yamlNeedsQuoting.MatchString(v) {
		escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(v)
		return `"` + escaped + `"`
	}
	return v
}

// sanitizeEnvValue removes newlines and carriage returns from a .env value
// to prevent injection of additional environment variables.
func sanitizeEnvValue(v string) string {
	v = strings.ReplaceAll(v, "\n", "")
	v = strings.ReplaceAll(v, "\r", "")
	return v
}

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
			lines = append(lines, fmt.Sprintf("      %s: %s", v.name, quoteYAMLValue(value)))
		}
	}
	return lines
}

// dotEnvActualLines returns .env lines with actual values from cfg.
// Fields with invalid env var names are skipped. Values are sanitized to prevent injection.
func dotEnvActualLines(r manifest.Resource, cfg Config) []string {
	var lines []string
	for _, fieldName := range r.FieldNames() {
		field := r.Fields[fieldName]
		if field.Env == "" || !validEnvVar.MatchString(field.Env) {
			continue
		}
		value := sanitizeEnvValue(cfg.ResourceValues[r.Key()+"."+fieldName])
		lines = append(lines, fmt.Sprintf("%s=%s", field.Env, value))
	}
	return lines
}

// dotEnvExampleLines returns .env.example lines with placeholders.
// Fields with invalid env var names are skipped.
func dotEnvExampleLines(r manifest.Resource, commented bool) []string {
	var lines []string
	for _, fieldName := range r.FieldNames() {
		field := r.Fields[fieldName]
		if field.Env == "" || !validEnvVar.MatchString(field.Env) {
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

// GenerateAppEnv generates the env entries for app.yaml.
// Each resource field with an Env mapping produces a YAML list entry with
// name (the env var) and valueFrom (the resource key in databricks.yml resources).
// Includes both required resources and optional resources that have values.
func GenerateAppEnv(plugins []manifest.Plugin, cfg Config) string {
	var lines []string

	for _, p := range plugins {
		for _, r := range p.Resources.Required {
			lines = append(lines, appEnvLines(r)...)
		}
		for _, r := range p.Resources.Optional {
			if hasResourceValues(r, cfg) {
				lines = append(lines, appEnvLines(r)...)
			}
		}
	}

	return strings.Join(lines, "\n")
}

// appEnvLines returns app.yaml env entries for a resource.
// Each field with an Env produces "- name: <ENV>\n  valueFrom: <resourceKey>".
func appEnvLines(r manifest.Resource) []string {
	var lines []string
	for _, fieldName := range r.FieldNames() {
		field := r.Fields[fieldName]
		if field.Env == "" || !validEnvVar.MatchString(field.Env) {
			continue
		}
		lines = append(lines,
			"  - name: "+field.Env,
			"    valueFrom: "+r.Key(),
		)
	}
	return lines
}

// appResourceSpec defines how a manifest resource type maps to DABs AppResource YAML.
type appResourceSpec struct {
	yamlKey      string      // DABs YAML key under the resource entry (e.g., "sql_warehouse", "uc_securable")
	varFields    [][2]string // {manifestFieldName, yamlFieldName} pairs that generate ${var.xxx} references
	staticFields [][2]string // {yamlFieldName, literalValue} pairs for constants
	permission   string      // default permission when the manifest doesn't specify one
}

// appResourceSpecs maps manifest resource types to their DABs AppResource YAML specification.
var appResourceSpecs = map[string]appResourceSpec{
	"sql_warehouse": {
		yamlKey:    "sql_warehouse",
		varFields:  [][2]string{{"id", "id"}},
		permission: "CAN_USE",
	},
	"job": {
		yamlKey:    "job",
		varFields:  [][2]string{{"id", "id"}},
		permission: "CAN_MANAGE_RUN",
	},
	"serving_endpoint": {
		yamlKey:    "serving_endpoint",
		varFields:  [][2]string{{"id", "name"}},
		permission: "CAN_QUERY",
	},
	"experiment": {
		yamlKey:    "experiment",
		varFields:  [][2]string{{"id", "experiment_id"}},
		permission: "CAN_READ",
	},
	"secret": {
		yamlKey:    "secret",
		varFields:  [][2]string{{"scope", "scope"}, {"key", "key"}},
		permission: "READ",
	},
	"database": {
		yamlKey:    "database",
		varFields:  [][2]string{{"instance_name", "instance_name"}, {"database_name", "database_name"}},
		permission: "CAN_CONNECT_AND_CREATE",
	},
	"genie_space": {
		yamlKey:    "genie_space",
		varFields:  [][2]string{{"name", "name"}, {"id", "space_id"}},
		permission: "CAN_VIEW",
	},
	"volume": {
		yamlKey:      "uc_securable",
		varFields:    [][2]string{{"id", "securable_full_name"}},
		staticFields: [][2]string{{"securable_type", "VOLUME"}},
		permission:   "READ_VOLUME",
	},
	"uc_function": {
		yamlKey:      "uc_securable",
		varFields:    [][2]string{{"id", "securable_full_name"}},
		staticFields: [][2]string{{"securable_type", "FUNCTION"}},
		permission:   "EXECUTE",
	},
	"uc_connection": {
		yamlKey:      "uc_securable",
		varFields:    [][2]string{{"id", "securable_full_name"}},
		staticFields: [][2]string{{"securable_type", "CONNECTION"}},
		permission:   "USE_CONNECTION",
	},
	"vector_search_index": {
		yamlKey:      "uc_securable",
		varFields:    [][2]string{{"id", "securable_full_name"}},
		staticFields: [][2]string{{"securable_type", "TABLE"}},
		permission:   "SELECT",
	},
	// TODO: uncomment when bundles support app as an app resource type.
	// "app": {
	// 	yamlKey:    "app",
	// 	varFields:  [][2]string{{"id", "name"}},
	// 	permission: "CAN_USE",
	// },
}

// varNameForField returns the bundle variable name for a specific field of a resource.
// Uses VarPrefix (resource_key with hyphens replaced by underscores).
func varNameForField(r manifest.Resource, fieldName string) string {
	return r.VarPrefix() + "_" + fieldName
}

// variableNamesForResource returns the variable names that a resource type needs.
// It merges manifest Fields with spec varFields so that fields required by the
// DABs YAML (e.g., genie_space name) are included even when the manifest doesn't
// declare them. Manifest Fields take precedence for descriptions.
func variableNamesForResource(r manifest.Resource) []varInfo {
	var vars []varInfo
	covered := make(map[string]bool)

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
		covered[fieldName] = true
	}

	// Include spec varFields not already covered by manifest Fields.
	if spec, ok := appResourceSpecs[r.Type]; ok {
		for _, f := range spec.varFields {
			if !covered[f[0]] {
				vars = append(vars, varInfo{
					name:        varNameForField(r, f[0]),
					description: r.Description,
					valueKey:    r.Key() + "." + f[0],
				})
			}
		}
	}

	if len(vars) > 0 {
		return vars
	}
	// Fallback for resources without explicit Fields and no spec.
	// Uses "key.id" to stay consistent with the composite key convention.
	return []varInfo{
		{name: aliasToVarName(r.VarPrefix()), description: r.Description, valueKey: r.Key() + ".id"},
	}
}

// generateResourceYAML generates YAML for a single app resource based on its type.
// Uses the appResourceSpecs mapping to produce the correct DABs AppResource structure.
func generateResourceYAML(r manifest.Resource, indent int) string {
	spec, ok := appResourceSpecs[r.Type]
	if !ok {
		return ""
	}

	permission := r.Permission
	if permission == "" {
		permission = spec.permission
	}

	pad := strings.Repeat(" ", indent)

	var lines []string
	lines = append(lines, fmt.Sprintf("%s- name: %s", pad, r.Key()))
	lines = append(lines, fmt.Sprintf("%s  %s:", pad, spec.yamlKey))

	for _, f := range spec.varFields {
		manifestField, yamlField := f[0], f[1]
		lines = append(lines, fmt.Sprintf("%s    %s: ${var.%s}", pad, yamlField, varNameForField(r, manifestField)))
	}
	for _, sf := range spec.staticFields {
		yamlField, value := sf[0], sf[1]
		lines = append(lines, fmt.Sprintf("%s    %s: %s", pad, yamlField, value))
	}

	lines = append(lines, fmt.Sprintf("%s    permission: %s", pad, permission))
	return strings.Join(lines, "\n")
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

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
	// ResourceValues maps resource alias to its value (e.g., "warehouse" -> "abc123")
	ResourceValues map[string]string
}

// GenerateBundleVariables generates the variables section for databricks.yml.
// Output is indented with 2 spaces for insertion under "variables:".
// Includes both required resources and optional resources that have values.
func GenerateBundleVariables(plugins []manifest.Plugin, cfg Config) string {
	var lines []string

	for _, p := range plugins {
		// Required resources
		for _, r := range p.Resources.Required {
			varName := aliasToVarName(r.Alias)
			lines = append(lines, fmt.Sprintf("  %s:", varName))
			if r.Description != "" {
				lines = append(lines, "    description: "+r.Description)
			}
		}
		// Optional resources (only if value provided)
		for _, r := range p.Resources.Optional {
			if _, hasValue := cfg.ResourceValues[r.Alias]; hasValue {
				varName := aliasToVarName(r.Alias)
				lines = append(lines, fmt.Sprintf("  %s:", varName))
				if r.Description != "" {
					lines = append(lines, "    description: "+r.Description)
				}
			}
		}
	}

	return strings.Join(lines, "\n")
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
			if _, hasValue := cfg.ResourceValues[r.Alias]; hasValue {
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
		// Required resources
		for _, r := range p.Resources.Required {
			varName := aliasToVarName(r.Alias)
			value := cfg.ResourceValues[r.Alias]
			if value != "" {
				lines = append(lines, fmt.Sprintf("      %s: %s", varName, value))
			}
		}
		// Optional resources (only if value provided)
		for _, r := range p.Resources.Optional {
			value := cfg.ResourceValues[r.Alias]
			if value != "" {
				varName := aliasToVarName(r.Alias)
				lines = append(lines, fmt.Sprintf("      %s: %s", varName, value))
			}
		}
	}

	return strings.Join(lines, "\n")
}

// GenerateDotEnv generates the .env file content with actual values.
// Includes both required resources and optional resources that have values.
func GenerateDotEnv(plugins []manifest.Plugin, cfg Config) string {
	var lines []string

	for _, p := range plugins {
		// Required resources
		for _, r := range p.Resources.Required {
			if r.Env != "" {
				value := cfg.ResourceValues[r.Alias]
				lines = append(lines, fmt.Sprintf("%s=%s", r.Env, value))
			}
		}
		// Optional resources (only if value provided)
		for _, r := range p.Resources.Optional {
			value := cfg.ResourceValues[r.Alias]
			if r.Env != "" && value != "" {
				lines = append(lines, fmt.Sprintf("%s=%s", r.Env, value))
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
		// Required resources
		for _, r := range p.Resources.Required {
			if r.Env != "" {
				placeholder := "your_" + strings.ToLower(r.Alias)
				lines = append(lines, fmt.Sprintf("%s=%s", r.Env, placeholder))
			}
		}
		// Optional resources (commented out)
		for _, r := range p.Resources.Optional {
			if r.Env != "" {
				placeholder := "your_" + strings.ToLower(r.Alias)
				lines = append(lines, fmt.Sprintf("# %s=%s", r.Env, placeholder))
			}
		}
	}

	return strings.Join(lines, "\n")
}

// generateResourceYAML generates YAML for a single resource based on its type.
// indent specifies the number of spaces to indent each line.
func generateResourceYAML(r manifest.Resource, indent int) string {
	switch r.Type {
	case "sql_warehouse":
		return generateSQLWarehouseResource(r, indent)
	default:
		// Unknown resource type - skip
		return ""
	}
}

// generateSQLWarehouseResource generates the app resource for a SQL warehouse.
func generateSQLWarehouseResource(r manifest.Resource, indent int) string {
	varName := aliasToVarName(r.Alias)
	permission := r.Permission
	if permission == "" {
		permission = "CAN_USE"
	}

	pad := strings.Repeat(" ", indent)
	return fmt.Sprintf(`%s- name: %s
%s  sql_warehouse:
%s    id: ${var.%s}
%s    permission: %s`, pad, r.Alias, pad, pad, varName, pad, permission)
}

// aliasToVarName converts a resource alias to a variable name.
// e.g., "warehouse" -> "warehouse_id"
func aliasToVarName(alias string) string {
	if strings.HasSuffix(alias, "_id") {
		return alias
	}
	return alias + "_id"
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

package legacytemplates

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"text/template"

	"github.com/databricks/cli/cmd/apps/internal"
	"gopkg.in/yaml.v3"
)

//go:embed env.tmpl
var envFileTemplate string

// EnvVar represents a single environment variable in app.yml.
type EnvVar struct {
	Name      string `yaml:"name"`
	Value     string `yaml:"value"`
	ValueFrom string `yaml:"value_from"`
}

// AppYml represents the structure of app.yml/app.yaml.
type AppYml struct {
	Command []string `yaml:"command"`
	Env     []EnvVar `yaml:"env"`
}

// EnvFileBuilder builds .env file content from app.yml and resource values.
type EnvFileBuilder struct {
	host      string
	profile   string
	appName   string
	env       []EnvVar
	resources map[string]string
}

// NewEnvFileBuilder creates a new EnvFileBuilder.
// host: Databricks workspace host
// profile: Databricks CLI profile name (empty or "DEFAULT" for default profile)
// appName: Application name
// appYmlPath: Path to app.yml or app.yaml file
// resources: Map of resource names from databricks.yml to their values
//
//	(e.g., "sql-warehouse" -> "abc123", "experiment" -> "exp-456")
//	These names match the resource.name field in databricks.yml
func NewEnvFileBuilder(host, profile, appName, appYmlPath string, resources map[string]string) (*EnvFileBuilder, error) {
	// Read app.yml
	data, err := os.ReadFile(appYmlPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No app.yml, return empty builder
			return &EnvFileBuilder{
				host:      host,
				profile:   profile,
				appName:   appName,
				env:       []EnvVar{},
				resources: resources,
			}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", appYmlPath, err)
	}

	// Parse into yaml.Node to handle camelCase conversion
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", appYmlPath, err)
	}

	// Convert camelCase keys to snake_case (legacy templates use camelCase)
	ConvertKeysToSnakeCase(&node)

	// Marshal back to YAML with snake_case keys
	convertedData, err := yaml.Marshal(&node)
	if err != nil {
		return nil, fmt.Errorf("failed to convert %s to snake_case: %w", appYmlPath, err)
	}

	// Parse app.yml with snake_case keys
	var appYml AppYml
	if err := yaml.Unmarshal(convertedData, &appYml); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", appYmlPath, err)
	}

	return &EnvFileBuilder{
		host:      host,
		profile:   profile,
		appName:   appName,
		env:       appYml.Env,
		resources: resources,
	}, nil
}

// envTemplateData holds data for rendering the .env file template.
type envTemplateData struct {
	DatabricksHost    string
	AppName           string
	MlflowTrackingURI string
	AppYmlVars        []envVarPair
}

// envVarPair represents a single environment variable name-value pair.
type envVarPair struct {
	Name  string
	Value string
}

// ConvertKeysToSnakeCase recursively converts all mapping keys in a yaml.Node from camelCase to snake_case.
func ConvertKeysToSnakeCase(node *yaml.Node) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			ConvertKeysToSnakeCase(child)
		}
	case yaml.MappingNode:
		// Process key-value pairs
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Convert key to snake_case
			if keyNode.Kind == yaml.ScalarNode {
				keyNode.Value = internal.CamelToSnake(keyNode.Value)
			}

			// Recursively process value
			ConvertKeysToSnakeCase(valueNode)
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			ConvertKeysToSnakeCase(child)
		}
	case yaml.ScalarNode:
		// Leaf node, nothing to recurse into
	case yaml.AliasNode:
		// Alias nodes point to other nodes, no conversion needed
	}
}

// Build generates the .env file content.
func (b *EnvFileBuilder) Build() (string, error) {
	if len(b.env) == 0 && b.host == "" && b.profile == "" {
		return "", nil
	}

	// Prepare template data
	data := envTemplateData{
		AppName:    b.appName,
		AppYmlVars: make([]envVarPair, 0, len(b.env)),
	}

	// Check if DATABRICKS_HOST is already present in env vars
	hasHost := false
	for _, envVar := range b.env {
		if envVar.Name == "DATABRICKS_HOST" {
			hasHost = true
			break
		}
	}

	// Add DATABRICKS_HOST to template data if not in app.yml and we have a host
	if !hasHost && b.host != "" {
		data.DatabricksHost = b.host
	}

	// Always set MLFLOW_TRACKING_URI (override if present in app.yml)
	// Format: "databricks" for default profile, "databricks://<profile>" for named profiles
	data.MlflowTrackingURI = "databricks"
	if b.profile != "" && b.profile != "DEFAULT" {
		data.MlflowTrackingURI = "databricks://" + b.profile
	}

	// Process env vars from app.yml (skip MLFLOW_TRACKING_URI as we already added it)
	for _, envVar := range b.env {
		if envVar.Name == "" || envVar.Name == "MLFLOW_TRACKING_URI" {
			continue
		}

		var value string
		if envVar.Value != "" {
			// Direct value
			value = envVar.Value
		} else if envVar.ValueFrom != "" {
			// Lookup from resources (should match resource.name from databricks.yml)
			resourceValue, ok := b.resources[envVar.ValueFrom]
			if !ok {
				return "", fmt.Errorf("resource reference %q not found for environment variable %q (should match a resource.name from databricks.yml)", envVar.ValueFrom, envVar.Name)
			}
			value = resourceValue
		} else {
			// Empty value
			value = ""
		}

		data.AppYmlVars = append(data.AppYmlVars, envVarPair{
			Name:  envVar.Name,
			Value: value,
		})
	}

	// Execute template
	tmpl, err := template.New("env").Parse(envFileTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse .env template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute .env template: %w", err)
	}

	return buf.String(), nil
}

// WriteEnvFile writes the .env file to the specified directory.
func (b *EnvFileBuilder) WriteEnvFile(destDir string) error {
	content, err := b.Build()
	if err != nil {
		return err
	}

	if content == "" {
		// No env vars to write
		return nil
	}

	envPath := destDir + "/.env"
	return os.WriteFile(envPath, []byte(content), 0o644)
}

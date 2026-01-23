package apps

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	env       []EnvVar
	resources map[string]string
}

// NewEnvFileBuilder creates a new EnvFileBuilder.
// host: Databricks workspace host
// appYmlPath: Path to app.yml or app.yaml file
// resources: Map of resource references (e.g., "WAREHOUSE_ID" -> "abc123")
func NewEnvFileBuilder(host string, appYmlPath string, resources map[string]string) (*EnvFileBuilder, error) {
	// Read app.yml
	data, err := os.ReadFile(appYmlPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No app.yml, return empty builder
			return &EnvFileBuilder{
				host:      host,
				env:       []EnvVar{},
				resources: resources,
			}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", appYmlPath, err)
	}

	// Parse app.yml
	var appYml AppYml
	if err := yaml.Unmarshal(data, &appYml); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", appYmlPath, err)
	}

	return &EnvFileBuilder{
		host:      host,
		env:       appYml.Env,
		resources: resources,
	}, nil
}

// Build generates the .env file content.
func (b *EnvFileBuilder) Build() (string, error) {
	if len(b.env) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("# Environment variables from app.yml\n")
	sb.WriteString("# Generated automatically - modify app.yml to change\n\n")

	// Add DATABRICKS_HOST if not already present in env vars
	hasHost := false
	for _, envVar := range b.env {
		if envVar.Name == "DATABRICKS_HOST" {
			hasHost = true
			break
		}
	}
	if !hasHost && b.host != "" {
		sb.WriteString(fmt.Sprintf("DATABRICKS_HOST=%s\n", b.host))
	}

	// Process env vars from app.yml
	for _, envVar := range b.env {
		if envVar.Name == "" {
			continue
		}

		var value string
		if envVar.Value != "" {
			// Direct value
			value = envVar.Value
		} else if envVar.ValueFrom != "" {
			// Lookup from resources
			resourceValue, ok := b.resources[envVar.ValueFrom]
			if !ok {
				return "", fmt.Errorf("resource reference %q not found for environment variable %q", envVar.ValueFrom, envVar.Name)
			}
			value = resourceValue
		} else {
			// Empty value
			value = ""
		}

		sb.WriteString(fmt.Sprintf("%s=%s\n", envVar.Name, value))
	}

	return sb.String(), nil
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

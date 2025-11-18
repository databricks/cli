// Package mcp provides Model Context Protocol (MCP) server functionality
// integrated into the Databricks CLI.
package mcp

// Config holds MCP server configuration.
// Configuration is populated from CLI flags and Databricks client context.
type Config struct {
	AllowDeployment    bool
	WithWorkspaceTools bool
	WarehouseID        string
	DatabricksHost     string
	IoConfig           *IoConfig
}

// IoConfig configures the IO provider for project scaffolding and validation.
type IoConfig struct {
	Template   *TemplateConfig
	Validation *ValidationConfig
	Dagger     *DaggerConfig
}

// TemplateConfig specifies which template to use for scaffolding new projects.
type TemplateConfig struct {
	Name string
	Path string
}

// ValidationConfig defines custom validation commands and docker images for project validation.
type ValidationConfig struct {
	Command     string
	DockerImage string
	UseDagger   bool
	Timeout     int
}

// SetDefaults applies default values to ValidationConfig if not explicitly set.
func (v *ValidationConfig) SetDefaults() {
	if v.DockerImage == "" {
		v.DockerImage = "node:20-alpine"
	}
	if v.Timeout == 0 {
		v.Timeout = 600
	}
}

// DaggerConfig configures the Dagger sandbox when use_dagger is enabled.
type DaggerConfig struct {
	Image          string
	ExecuteTimeout int
}

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() *Config {
	validationCfg := &ValidationConfig{}
	validationCfg.SetDefaults()

	return &Config{
		AllowDeployment:    false,
		WithWorkspaceTools: false,
		IoConfig: &IoConfig{
			Template: &TemplateConfig{
				Name: "default",
				Path: "",
			},
			Validation: validationCfg,
			Dagger: &DaggerConfig{
				Image:          "node:20-alpine",
				ExecuteTimeout: 600,
			},
		},
		WarehouseID: "",
	}
}

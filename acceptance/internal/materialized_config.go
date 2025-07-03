package internal

import (
	"encoding/json"
)

const MaterializedConfigFile = "out.config.json"

type MaterializedConfig struct {
	GOOS                 map[string]bool     `json:"GOOS,omitempty"`
	CloudEnvs            map[string]bool     `json:"CloudEnvs,omitempty"`
	Local                *bool               `json:"Local,omitempty"`
	Cloud                *bool               `json:"Cloud,omitempty"`
	CloudSlow            *bool               `json:"CloudSlow,omitempty"`
	RequiresUnityCatalog *bool               `json:"RequiresUnityCatalog,omitempty"`
	RequiresCluster      *bool               `json:"RequiresCluster,omitempty"`
	RequiresWarehouse    *bool               `json:"RequiresWarehouse,omitempty"`
	EnvMatrix            map[string][]string `json:"EnvMatrix,omitempty"`
}

// GenerateMaterializedConfig creates a JSON representation of the configuration fields
// that determine where and how a test is executed
func GenerateMaterializedConfig(config TestConfig) (string, error) {
	materialized := MaterializedConfig{
		GOOS:                 config.GOOS,
		CloudEnvs:            config.CloudEnvs,
		Local:                config.Local,
		Cloud:                config.Cloud,
		CloudSlow:            config.CloudSlow,
		RequiresUnityCatalog: config.RequiresUnityCatalog,
		RequiresCluster:      config.RequiresCluster,
		RequiresWarehouse:    config.RequiresWarehouse,
		EnvMatrix:            config.EnvMatrix,
	}

	configBytes, err := json.MarshalIndent(materialized, "", "  ")
	if err != nil {
		return "", err
	}

	// Add newline at the end of the JSON
	return string(configBytes) + "\n", nil
}

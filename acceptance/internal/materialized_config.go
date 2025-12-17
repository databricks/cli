package internal

import (
	"bytes"

	"github.com/BurntSushi/toml"
)

const MaterializedConfigFile = "out.test.toml"

type MaterializedConfig struct {
	GOOS                 map[string]bool     `toml:"GOOS,omitempty"`
	CloudEnvs            map[string]bool     `toml:"CloudEnvs,omitempty"`
	Local                *bool               `toml:"Local,omitempty"`
	Cloud                *bool               `toml:"Cloud,omitempty"`
	CloudSlow            *bool               `toml:"CloudSlow,omitempty"`
	RequiresUnityCatalog *bool               `toml:"RequiresUnityCatalog,omitempty"`
	RequiresCluster      *bool               `toml:"RequiresCluster,omitempty"`
	RequiresWarehouse    *bool               `toml:"RequiresWarehouse,omitempty"`
	EnvMatrix            map[string][]string `toml:"EnvMatrix,omitempty"`
}

// GenerateMaterializedConfig creates a TOML representation of the configuration fields
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

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	err := encoder.Encode(materialized)
	if err != nil {
		return "", err
	}

	// Add newline at the end of the TOML
	return buf.String(), nil
}

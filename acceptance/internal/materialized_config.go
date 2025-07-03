package internal

import (
	"encoding/json"
	"time"
)

const MaterializedConfigFile = "out.config.json"

type MaterializedConfig struct {
	GOOS                 map[string]bool     `json:"goos,omitempty"`
	CloudEnvs            map[string]bool     `json:"cloud_envs,omitempty"`
	Local                *bool               `json:"local,omitempty"`
	Cloud                *bool               `json:"cloud,omitempty"`
	CloudSlow            *bool               `json:"cloud_slow,omitempty"`
	RequiresUnityCatalog *bool               `json:"requires_unity_catalog,omitempty"`
	RequiresCluster      *bool               `json:"requires_cluster,omitempty"`
	RequiresWarehouse    *bool               `json:"requires_warehouse,omitempty"`
	EnvMatrix            map[string][]string `json:"env_matrix,omitempty"`
	Timeout              float64             `json:"timeout_seconds,omitempty"`
	TimeoutWindows       float64             `json:"timeout_windows_seconds,omitempty"`
	TimeoutCloud         float64             `json:"timeout_cloud_seconds,omitempty"`
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
		Timeout:              durationSeconds(config.Timeout),
		TimeoutWindows:       durationSeconds(config.TimeoutWindows),
		TimeoutCloud:         durationSeconds(config.TimeoutCloud),
	}

	configBytes, err := json.MarshalIndent(materialized, "", "  ")
	if err != nil {
		return "", err
	}

	// Add newline at the end of the JSON
	return string(configBytes) + "\n", nil
}

func durationSeconds(d time.Duration) float64 {
	if d == 0 {
		return 0
	}
	return d.Seconds()
}

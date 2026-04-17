package debug

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

func TestCollectTargetsSortsByName(t *testing.T) {
	targets := map[string]*config.Target{
		"prod":    {Default: false},
		"dev":     {Default: true, Mode: config.Development},
		"staging": {},
	}

	result := collectTargets(targets)

	assert.Len(t, result, 3)
	assert.Equal(t, "dev", result[0].Name)
	assert.Equal(t, "prod", result[1].Name)
	assert.Equal(t, "staging", result[2].Name)
}

func TestCollectTargetsIncludesAllFields(t *testing.T) {
	targets := map[string]*config.Target{
		"dev": {
			Default: true,
			Mode:    config.Development,
			Workspace: &config.Workspace{
				Host: "https://dev.example.com",
			},
		},
		"prod": {
			Mode: config.Production,
			Workspace: &config.Workspace{
				Host: "https://prod.example.com",
			},
		},
	}

	result := collectTargets(targets)

	assert.Equal(t, "dev", result[0].Name)
	assert.True(t, result[0].Default)
	assert.Equal(t, config.Development, result[0].Mode)
	assert.Equal(t, "https://dev.example.com", result[0].Host)

	assert.Equal(t, "prod", result[1].Name)
	assert.False(t, result[1].Default)
	assert.Equal(t, config.Production, result[1].Mode)
	assert.Equal(t, "https://prod.example.com", result[1].Host)
}

func TestCollectTargetsHandlesNilWorkspace(t *testing.T) {
	targets := map[string]*config.Target{
		"dev": {Default: true},
	}

	result := collectTargets(targets)

	assert.Equal(t, "dev", result[0].Name)
	assert.True(t, result[0].Default)
	assert.Empty(t, result[0].Host)
}

func TestCollectTargetsEmpty(t *testing.T) {
	result := collectTargets(map[string]*config.Target{})
	assert.Empty(t, result)
}

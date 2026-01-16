package dresources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Resources)

	// Verify some known resource configurations
	volumes := cfg.Resources["volumes"]
	assert.Len(t, volumes.RecreateOnChanges, 4)
	assert.Len(t, volumes.UpdateIDOnChanges, 1)
	assert.Equal(t, "name", volumes.UpdateIDOnChanges[0].String())

	schemas := cfg.Resources["schemas"]
	assert.Len(t, schemas.RecreateOnChanges, 3)

	// Verify nested paths work
	endpoints := cfg.Resources["model_serving_endpoints"]
	found := false
	for _, p := range endpoints.RecreateOnChanges {
		if p.String() == "config.auto_capture_config.catalog_name" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find nested path config.auto_capture_config.catalog_name")
}

func TestGetResourceConfig(t *testing.T) {
	// Existing resource
	cfg := GetResourceConfig("volumes")
	require.NotNil(t, cfg)
	assert.Len(t, cfg.RecreateOnChanges, 4)

	// Non-existing resource returns nil
	cfg = GetResourceConfig("nonexistent")
	assert.Nil(t, cfg)

	// Jobs have no config in resources.yml
	cfg = GetResourceConfig("jobs")
	assert.Nil(t, cfg)
}

func TestConfigIgnoreRemoteChanges(t *testing.T) {
	cfg := GetResourceConfig("experiments")
	require.NotNil(t, cfg)
	require.Len(t, cfg.IgnoreRemoteChanges, 1)
	assert.Equal(t, "tags", cfg.IgnoreRemoteChanges[0].String())
}

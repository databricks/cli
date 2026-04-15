package generate

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

func TestGenerateCoverageMatchesSupportedResources(t *testing.T) {
	covered := map[string]bool{
		"jobs":       true,
		"pipelines":  true,
		"dashboards": true,
		"alerts":     true,
		"apps":       true,
	}

	for _, spec := range genericGenerateSpecs() {
		covered[spec.resourceGroup] = true
	}

	assert.Equal(t, len(config.SupportedResources()), len(covered))
	for group := range config.SupportedResources() {
		assert.Contains(t, covered, group)
	}
}

package generate

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

func TestGeneratorsAreRegisteredUnderResourceKeys(t *testing.T) {
	supported := config.SupportedResources()
	for resourceKey := range generatorCommands {
		assert.Contains(t, supported, resourceKey, "generator registered under unknown resource key %q", resourceKey)
	}
}

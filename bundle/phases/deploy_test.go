package phases

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

func TestResourceDestroyMessageIsComplete(t *testing.T) {
	supported := config.SupportedResources()

	for resourceType := range supported {
		_, ok := resourceDestroyMessage[resourceType]
		assert.True(t, ok, "resourceDestroyMessage is missing entry for %q", resourceType)
	}

	for resourceType := range resourceDestroyMessage {
		_, ok := supported[resourceType]
		assert.True(t, ok, "resourceDestroyMessage has entry for %q which is not a supported resource type", resourceType)
	}
}

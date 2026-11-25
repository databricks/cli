package configsync

import (
	"maps"
	"slices"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

func TestRegistrationsUseResourceTypeKeys(t *testing.T) {
	keys := slices.Concat(
		slices.Collect(maps.Keys(changeHooks)),
		slices.Collect(maps.Keys(clusterPolicyRemotes)),
		slices.Collect(maps.Keys(resourceIDFields)),
	)
	for _, key := range keys {
		_, ok := config.ResourceTypeAs[config.ConfigResource](key)
		assert.True(t, ok, "%q is registered but is not a resource type", key)
	}
}

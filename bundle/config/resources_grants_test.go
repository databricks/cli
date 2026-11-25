package config

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

// Shared code finds a type's securable type only through its declaration, so a type
// with a grants field that declares nothing would fail to deploy its grants.
func TestGrantsDeclaredIffGrantsField(t *testing.T) {
	for _, rt := range AllResourceTypes() {
		_, hasField := ResourcesTypes()[rt.Key+".grants"]
		d, declares := ResourceTypeAs[resources.GrantsDeclarer](rt.Key)
		if !hasField {
			assert.False(t, declares, "resource type %q implements resources.GrantsDeclarer but has no grants field", rt.Key)
			continue
		}
		if assert.True(t, declares, "resource type %q has a grants field but does not implement resources.GrantsDeclarer", rt.Key) {
			assert.NotEmpty(t, d.GrantsSupport().SecurableType, "resource type %q declares an empty securable type", rt.Key)
		}
	}
}

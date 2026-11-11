package dresources

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourcesWithoutCreate(t *testing.T) {
	for resourceType, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, resourceType, nil)
		require.NoError(t, err)
		noCreate := strings.HasSuffix(resourceType, ".grants") || strings.HasSuffix(resourceType, ".permissions")
		assert.Equal(t, !noCreate, adapter.HasDoCreate(), resourceType)
		if noCreate {
			assert.True(t, adapter.HasIDFromState(), resourceType)
			assert.True(t, adapter.HasDoUpdate(), resourceType)
		}
	}
}

func TestResourcesWithoutCreateRequireIdentityAndUpdate(t *testing.T) {
	for _, missing := range []string{"IDFromState", "DoUpdate"} {
		t.Run(missing, func(t *testing.T) {
			adapter, err := NewAdapter((*ResourceGrants)(nil), "schemas.grants", nil)
			require.NoError(t, err)
			switch missing {
			case "IDFromState":
				adapter.idFromState = nil
			case "DoUpdate":
				adapter.doUpdate = nil
			}
			assert.EqualError(t, adapter.validate(), "resources without DoCreate require IDFromState and DoUpdate")
		})
	}
}

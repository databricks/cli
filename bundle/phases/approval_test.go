package phases

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

// A type that declares nothing would get no data-loss warning, and nothing else would notice.
func TestEveryResourceTypeDeclaresApproval(t *testing.T) {
	config.AssertResourceTypesImplement[resources.ApprovalDeclarer](t)
}

func TestApprovalDeclarations(t *testing.T) {
	orders := map[int]string{}
	for _, a := range approvals() {
		_, hasHook := destroyApprovalHooks[a.key]
		assert.Equal(t, a.DestroyFromState, hasHook, "%s: DestroyFromState must be set if and only if a destroy approval hook is registered", a.key)
		if a.DestroyFromState {
			assert.Empty(t, a.Destroy, "%s: Destroy is unused when DestroyFromState is set", a.key)
		}
		if a.DeploySkipChildren {
			assert.NotEmpty(t, a.Deploy, "%s: DeploySkipChildren is set without a Deploy message", a.key)
		}

		if a.Deploy == "" && a.Destroy == "" && !a.DestroyFromState {
			assert.Zero(t, a.Order, "%s: Order is set without a message", a.key)
			continue
		}
		// The order of the messages must not depend on how ties are broken.
		if assert.NotZero(t, a.Order, "%s: a type with a message needs an Order", a.key) {
			other, dup := orders[a.Order]
			assert.False(t, dup, "%s and %s have the same Order %d", a.key, other, a.Order)
			orders[a.Order] = a.key
		}
	}

	for key := range destroyApprovalHooks {
		_, ok := config.ResourceTypeAs[resources.ApprovalDeclarer](key)
		assert.True(t, ok, "destroy approval hook registered for %q, which is not a resource type", key)
	}
}

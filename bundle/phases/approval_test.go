package phases

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

// Every group in approvalTable must be a resource type and needs a deploy message. Groups
// without a destroy message are listed explicitly, so adding a group to only one command
// is a visible choice here.
func TestApprovalTableEntries(t *testing.T) {
	supported := config.SupportedResources()
	var noDestroyMessage []string
	for _, e := range approvalTable {
		// A group that is not a resource type matches no action, so its messages never print.
		_, ok := supported[e.group]
		assert.True(t, ok, "group %s is not a resource type", e.group)
		assert.NotEmpty(t, e.deployMessage, "group %s has no deploy message", e.group)
		if e.destroyMessage == "" {
			noDestroyMessage = append(noDestroyMessage, e.group)
		}
	}

	// approvalTable says why each of these has none.
	assert.ElementsMatch(t, []string{"pipelines", "dashboards", "materialization_sets"}, noDestroyMessage)
}

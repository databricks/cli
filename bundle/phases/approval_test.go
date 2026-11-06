package phases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every group in approvalTable needs a deploy prompt. Groups without a destroy prompt
// are listed explicitly, so adding a group to only one command is a visible choice here.
func TestApprovalTableEntries(t *testing.T) {
	var noDestroyPrompt []string
	for _, e := range approvalTable {
		require.NotEmpty(t, e.deployMessage, "group %s has no deploy message", e.group)
		if e.destroyMessage == nil {
			noDestroyPrompt = append(noDestroyPrompt, e.group)
			continue
		}
		assert.NotEmpty(t, *e.destroyMessage, "group %s has an empty destroy message", e.group)
	}

	assert.ElementsMatch(t, []string{
		// Deploy only.
		"dashboards",
		"materialization_sets",
		// Destroy prompts for pipelines in logPipelineDeleteApproval instead.
		"pipelines",
	}, noDestroyPrompt)
}

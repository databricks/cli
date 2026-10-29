package agent

import (
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestAgentNote(t *testing.T) {
	testutil.SetAgent(t, "test-agent")
	assert.Equal(t, "Note for AI agents (test-agent): ask the user.", AgentNote("ask the user."))
}

func TestAgentNoteWithoutAgent(t *testing.T) {
	testutil.SetAgent(t, "")
	assert.Empty(t, AgentNote("ask the user."))
}

func TestAgentNotice(t *testing.T) {
	testutil.SetAgent(t, "test-agent")
	assert.Equal(t, "\n\nNote for AI agents (test-agent): do not retry with the flag suggested above\n"+
		"unless the user has explicitly approved it. The flag bypasses a safety check\n"+
		"and the operation may be irreversible.", AgentNotice())
}

func TestAgentNoticeWithoutAgent(t *testing.T) {
	testutil.SetAgent(t, "")
	assert.Empty(t, AgentNotice())
}

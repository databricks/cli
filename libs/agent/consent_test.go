package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentNoticeNoAgent(t *testing.T) {
	ctx := Mock(t.Context(), "")
	assert.Empty(t, AgentNotice(ctx))
}

func TestAgentNoticeWithAgent(t *testing.T) {
	ctx := Mock(t.Context(), ClaudeCode)
	notice := AgentNotice(ctx)
	assert.Contains(t, notice, "claude-code")
	assert.Contains(t, notice, "do not retry")
	assert.Contains(t, notice, "irreversible data loss")
}

func TestAgentNoticeBeforeDetect(t *testing.T) {
	ctx := t.Context()
	// Should not panic, just return empty.
	assert.Empty(t, AgentNotice(ctx))
}

package root

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
)

func TestAgentClaudeCode(t *testing.T) {
	ctx := context.Background()
	ctx = agent.Mock(ctx, agent.ClaudeCode)

	ctx = withAgentInUserAgent(ctx)
	assert.Contains(t, useragent.FromContext(ctx), "agent/claude-code")
}

func TestAgentGeminiCLI(t *testing.T) {
	ctx := context.Background()
	ctx = agent.Mock(ctx, agent.GeminiCLI)

	ctx = withAgentInUserAgent(ctx)
	assert.Contains(t, useragent.FromContext(ctx), "agent/gemini-cli")
}

func TestAgentCursor(t *testing.T) {
	ctx := context.Background()
	ctx = agent.Mock(ctx, agent.Cursor)

	ctx = withAgentInUserAgent(ctx)
	assert.Contains(t, useragent.FromContext(ctx), "agent/cursor")
}

func TestAgentNotSet(t *testing.T) {
	ctx := context.Background()
	ctx = agent.Mock(ctx, "")

	ctx = withAgentInUserAgent(ctx)
	assert.NotContains(t, useragent.FromContext(ctx), "agent/")
}

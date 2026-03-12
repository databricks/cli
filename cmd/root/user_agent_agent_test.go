package root

import (
	"testing"

	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
)

func TestAgentInUserAgent(t *testing.T) {
	for _, product := range []string{
		agent.Antigravity,
		agent.ClaudeCode,
		agent.Cline,
		agent.Codex,
		agent.Cursor,
		agent.GeminiCLI,
		agent.OpenCode,
	} {
		t.Run(product, func(t *testing.T) {
			ctx := t.Context()
			ctx = agent.Mock(ctx, product)

			ctx = withAgentInUserAgent(ctx)
			assert.Contains(t, useragent.FromContext(ctx), "agent/"+product)
		})
	}
}

func TestAgentNotSet(t *testing.T) {
	ctx := t.Context()
	ctx = agent.Mock(ctx, "")

	ctx = withAgentInUserAgent(ctx)
	assert.NotContains(t, useragent.FromContext(ctx), "agent/")
}

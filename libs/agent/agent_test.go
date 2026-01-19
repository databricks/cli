package agent

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect(t *testing.T) {
	ctx := context.Background()
	// Clear other agent env vars to ensure clean test environment
	ctx = env.Set(ctx, geminiCliEnvVar, "")
	ctx = env.Set(ctx, cursorAgentEnvVar, "")
	ctx = env.Set(ctx, claudeCodeEnvVar, "1")

	ctx = Detect(ctx)

	assert.Equal(t, ClaudeCode, Product(ctx))
}

func TestProductCalledBeforeDetect(t *testing.T) {
	ctx := context.Background()

	require.Panics(t, func() {
		Product(ctx)
	})
}

func TestMock(t *testing.T) {
	ctx := context.Background()
	ctx = Mock(ctx, "test-agent")

	assert.Equal(t, "test-agent", Product(ctx))
}

func TestDetectNoAgent(t *testing.T) {
	ctx := context.Background()
	ctx = env.Set(ctx, claudeCodeEnvVar, "")
	ctx = env.Set(ctx, geminiCliEnvVar, "")
	ctx = env.Set(ctx, cursorAgentEnvVar, "")

	ctx = Detect(ctx)

	assert.Equal(t, "", Product(ctx))
}

func TestDetectClaudeCode(t *testing.T) {
	ctx := context.Background()
	// Clear other agent env vars to ensure clean test environment
	ctx = env.Set(ctx, geminiCliEnvVar, "")
	ctx = env.Set(ctx, cursorAgentEnvVar, "")
	ctx = env.Set(ctx, claudeCodeEnvVar, "1")

	result := detect(ctx)
	assert.Equal(t, ClaudeCode, result)
}

func TestDetectGeminiCLI(t *testing.T) {
	ctx := context.Background()
	// Clear other agent env vars to ensure clean test environment
	ctx = env.Set(ctx, claudeCodeEnvVar, "")
	ctx = env.Set(ctx, cursorAgentEnvVar, "")
	ctx = env.Set(ctx, geminiCliEnvVar, "1")

	result := detect(ctx)
	assert.Equal(t, GeminiCLI, result)
}

func TestDetectCursor(t *testing.T) {
	ctx := context.Background()
	// Clear other agent env vars to ensure clean test environment
	ctx = env.Set(ctx, claudeCodeEnvVar, "")
	ctx = env.Set(ctx, geminiCliEnvVar, "")
	ctx = env.Set(ctx, cursorAgentEnvVar, "1")

	result := detect(ctx)
	assert.Equal(t, Cursor, result)
}

func TestDetectMultipleAgents(t *testing.T) {
	ctx := context.Background()
	// Clear all agent env vars first
	ctx = env.Set(ctx, cursorAgentEnvVar, "")
	// If multiple agents are detected, return empty string
	ctx = env.Set(ctx, claudeCodeEnvVar, "1")
	ctx = env.Set(ctx, geminiCliEnvVar, "1")

	result := detect(ctx)
	assert.Equal(t, "", result)
}

func TestDetectMultipleAgentsAllThree(t *testing.T) {
	ctx := context.Background()
	// If all three agents are detected, return empty string
	ctx = env.Set(ctx, claudeCodeEnvVar, "1")
	ctx = env.Set(ctx, geminiCliEnvVar, "1")
	ctx = env.Set(ctx, cursorAgentEnvVar, "1")

	result := detect(ctx)
	assert.Equal(t, "", result)
}

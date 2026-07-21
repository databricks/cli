package genieclicmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSystemPromptWithProfile(t *testing.T) {
	got := buildSystemPrompt("https://myworkspace.databricks.com", "prod")
	assert.Contains(t, got, "https://myworkspace.databricks.com")
	assert.Contains(t, got, `--profile prod`)
	assert.Contains(t, got, "Databricks CLI assistant")
	// Data questions should route to hosted Genie.
	assert.Contains(t, got, "databricks experimental genie ask")
}

func TestBuildSystemPromptWithoutProfile(t *testing.T) {
	got := buildSystemPrompt("https://myworkspace.databricks.com", "")
	assert.Contains(t, got, "https://myworkspace.databricks.com")
	// With no named profile the agent is told auth comes from the environment
	// and must not be told to pass a specific `--profile <name>`.
	assert.Contains(t, got, "resolved from the environment")
	assert.NotContains(t, got, `--profile "`)
}

func TestTomlQuoteEscapes(t *testing.T) {
	got := tomlQuote("a\"b\nc\\d")
	assert.Equal(t, `"a\"b\nc\\d"`, got)
}

package genieclicmd

import (
	"strings"
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

func TestDeveloperInstructionsOverrideIsQuotedTOML(t *testing.T) {
	got := developerInstructionsOverride("https://myworkspace.databricks.com", "prod")
	assert.True(t, strings.HasPrefix(got, `developer_instructions="`))
	assert.True(t, strings.HasSuffix(got, `"`))
	// The multi-line prompt must be escaped to a single TOML basic string.
	value := strings.TrimPrefix(got, "developer_instructions=")
	assert.NotContains(t, value[1:len(value)-1], "\n")
	assert.Contains(t, value, `\n`)
}

func TestTomlQuoteEscapes(t *testing.T) {
	got := tomlQuote("a\"b\nc\\d")
	assert.Equal(t, `"a\"b\nc\\d"`, got)
}

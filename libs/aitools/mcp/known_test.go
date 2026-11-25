package mcp

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnownSystemAITable(t *testing.T) {
	tests := []struct {
		leaf        string
		label       string
		summary     string
		beta        bool
		recommended bool
	}{
		{leaf: "genie_one_mcp", label: "Genie One", summary: "Ask questions about your data", recommended: true},
		{leaf: "dbsql", label: "Databricks SQL", summary: "Run SQL on a SQL warehouse", beta: true},
		{leaf: "web_search", label: "Web search", summary: "Search the public web", beta: true},
		{leaf: "sandbox", label: "Sandbox", summary: "Run Python or shell in a sandbox", beta: true},
		{leaf: "github", label: "GitHub", summary: "Repositories, issues, pull requests"},
		{leaf: "slack", label: "Slack", summary: "Channels and messages"},
		{leaf: "atlassian", label: "Jira and Confluence", summary: "Issues and pages"},
		{leaf: "google_drive", label: "Google Drive", summary: "Drive, Docs, Sheets, Slides"},
		{leaf: "google_calendar", label: "Google Calendar", summary: "Calendar events"},
		{leaf: "gmail", label: "Gmail", summary: "Email messages"},
		{leaf: "microsoft_365", label: "Microsoft 365", summary: "SharePoint, Outlook, Teams"},
	}
	require.Len(t, knownSystemAI, len(tests))
	for _, tt := range tests {
		t.Run(tt.leaf, func(t *testing.T) {
			known, ok := knownSystemAI[tt.leaf]
			require.True(t, ok)
			assert.Equal(t, tt.label, known.Label)
			assert.Equal(t, tt.summary, known.Summary)
			assert.Equal(t, tt.beta, known.Beta)
			assert.Equal(t, tt.recommended, known.Recommended)
		})
	}
}

// TestKnownLeavesAreTheSystemAITable checks that KnownLeaves lists the leaves of the system.ai table, sorted, with each
// leaf known as a system.ai service.
func TestKnownLeavesAreTheSystemAITable(t *testing.T) {
	leaves := KnownLeaves()
	assert.Contains(t, leaves, "genie_one_mcp")
	assert.True(t, slices.IsSorted(leaves))
	assert.Len(t, leaves, len(knownSystemAI))
	for _, leaf := range leaves {
		assert.NotNil(t, KnownOf(SystemAILocation+"."+leaf), leaf)
	}
}

func TestProviderLabel(t *testing.T) {
	assert.Equal(t, "GitHub", ProviderLabel("system.ai.github"))
	assert.Equal(t, "Genie One", ProviderLabel("SYSTEM.AI.genie_one_mcp"))
	assert.Equal(t, "system.ai.unknown_service", ProviderLabel("system.ai.unknown_service"))
	assert.Equal(t, "main.mcp.github", ProviderLabel("main.mcp.github"))
	assert.Equal(t, "GitHub", ProviderLabel("SYSTEM.AI.GITHUB"), "the leaf is compared without case")
}

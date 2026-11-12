package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnownSystemAITable(t *testing.T) {
	tests := []struct {
		leaf        string
		label       string
		beta        bool
		recommended bool
	}{
		{leaf: "genie_one_mcp", label: "Genie One", recommended: true},
		{leaf: "dbsql", label: "Databricks SQL", beta: true},
		{leaf: "web_search", label: "Web search", beta: true},
		{leaf: "sandbox", label: "Sandbox", beta: true},
		{leaf: "github", label: "GitHub"},
		{leaf: "slack", label: "Slack"},
		{leaf: "atlassian", label: "Jira and Confluence"},
		{leaf: "google_drive", label: "Google Drive"},
		{leaf: "google_calendar", label: "Google Calendar"},
		{leaf: "gmail", label: "Gmail"},
		{leaf: "microsoft_365", label: "Microsoft 365"},
	}
	require.Len(t, knownSystemAI, len(tests))
	for _, tt := range tests {
		t.Run(tt.leaf, func(t *testing.T) {
			known, ok := knownSystemAI[tt.leaf]
			require.True(t, ok)
			assert.Equal(t, tt.leaf, known.Leaf)
			assert.Equal(t, tt.label, known.Label)
			assert.Equal(t, tt.beta, known.Beta)
			assert.Equal(t, tt.recommended, known.Recommended)
			assert.Equal(t, builtinServicesDocs, known.Docs)
		})
	}
}

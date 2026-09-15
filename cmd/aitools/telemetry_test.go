package aitools

import (
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
)

func TestAgentsField(t *testing.T) {
	cases := []struct {
		name string
		plan []agentPlanItem
		want []protos.AitoolsAgentType
	}{
		{
			name: "empty plan",
			plan: nil,
			want: nil,
		},
		{
			name: "single agent",
			plan: []agentPlanItem{{agent: &agents.Agent{Name: agents.NameClaudeCode}}},
			want: []protos.AitoolsAgentType{protos.AitoolsAgentTypeClaudeCode},
		},
		{
			name: "multiple agents are sorted",
			plan: []agentPlanItem{
				{agent: &agents.Agent{Name: agents.NameCursor}},
				{agent: &agents.Agent{Name: agents.NameClaudeCode}},
				{agent: &agents.Agent{Name: agents.NameCodex}},
			},
			want: []protos.AitoolsAgentType{
				protos.AitoolsAgentTypeClaudeCode,
				protos.AitoolsAgentTypeCodex,
				protos.AitoolsAgentTypeCursor,
			},
		},
		{
			name: "duplicates are deduplicated",
			plan: []agentPlanItem{
				{agent: &agents.Agent{Name: agents.NameClaudeCode}},
				{agent: &agents.Agent{Name: agents.NameClaudeCode}},
			},
			want: []protos.AitoolsAgentType{protos.AitoolsAgentTypeClaudeCode},
		},
		{
			name: "nil agent entries are skipped",
			plan: []agentPlanItem{
				{agent: nil},
				{agent: &agents.Agent{Name: agents.NameCursor}},
			},
			want: []protos.AitoolsAgentType{protos.AitoolsAgentTypeCursor},
		},
		{
			name: "unknown agent maps to unspecified",
			plan: []agentPlanItem{{agent: &agents.Agent{Name: "future-agent"}}},
			want: []protos.AitoolsAgentType{protos.AitoolsAgentTypeUnspecified},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := agentsField(tc.plan)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestAgentTypeCoversRegistry fails when an agent is added to agents.Registry
// without a matching telemetry enum in agentType. This is the guard against
// silently logging a new agent as UNSPECIFIED.
func TestAgentTypeCoversRegistry(t *testing.T) {
	for _, a := range agents.Registry {
		assert.NotEqualf(t, protos.AitoolsAgentTypeUnspecified, agentType(a.Name),
			"agent %q has no telemetry enum: add a case to agentType and a value to "+
				"AitoolsAgentType in enum.proto (Universe) + aitools_install.go (CLI)", a.Name)
	}
}

func TestScopeType(t *testing.T) {
	assert.Equal(t, protos.AitoolsInstallScopeGlobal, scopeType(installer.ScopeGlobal))
	assert.Equal(t, protos.AitoolsInstallScopeProject, scopeType(installer.ScopeProject))
	assert.Equal(t, protos.AitoolsInstallScopeUnspecified, scopeType(""))
}

func TestAgentResultsField(t *testing.T) {
	claude := &agents.Agent{Name: agents.NameClaudeCode}
	codex := &agents.Agent{Name: agents.NameCodex}
	cursor := &agents.Agent{Name: agents.NameCursor}

	outcomes := []agentOutcome{
		// Successful agents produce no entry; production leaves errorCategory unset.
		{agent: cursor, status: outcomeInstalled},
		{agent: codex, status: outcomeFailed, errorCategory: protos.AitoolsErrorCategoryPluginInstallFailed},
		{agent: claude, status: outcomeSkipped, errorCategory: protos.AitoolsErrorCategoryUnsupportedScope},
	}

	// Sorted by agent enum, only non-successful agents included.
	want := []protos.AitoolsAgentResult{
		{Agent: protos.AitoolsAgentTypeClaudeCode, ErrorCategory: protos.AitoolsErrorCategoryUnsupportedScope},
		{Agent: protos.AitoolsAgentTypeCodex, ErrorCategory: protos.AitoolsErrorCategoryPluginInstallFailed},
	}
	assert.Equal(t, want, agentResultsField(outcomes))

	assert.Nil(t, agentResultsField(nil))
}

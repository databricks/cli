package aitools

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/agents/agenttest"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uploadInstallEvent runs logInstallEvent, uploads the buffered log to a local
// /telemetry-ext and returns the install event as the server received it.
func uploadInstallEvent(t *testing.T) map[string]any {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	var protoLogs []string
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		var body telemetry.RequestBody
		require.NoError(t, json.Unmarshal(req.Body, &body))
		protoLogs = append(protoLogs, body.ProtoLogs...)
		return telemetry.ResponseBody{NumProtoSuccess: int64(len(body.ProtoLogs))}
	})

	// Upload honours these from the host env; an empty value restores the
	// defaults (enabled, 3s timeout) so the test does not depend on the runner.
	ctx := env.Set(t.Context(), "DATABRICKS_CLI_DISABLE_TELEMETRY", "")
	ctx = env.Set(ctx, "DATABRICKS_CLI_TELEMETRY_TIMEOUT", "")
	ctx = telemetry.WithNewLogger(ctx)
	plan := []agentPlanItem{{agent: &agents.Agent{Name: agents.NameCursor}}}
	logInstallEvent(ctx, plan, installOpts{Scope: installer.ScopeGlobal}, protos.AitoolsErrorCategoryUnspecified, nil)

	// AuthType pins PAT so a host DATABRICKS_AUTH_TYPE cannot pick another method.
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{Host: server.URL, Token: "token", AuthType: "pat"})
	require.NoError(t, telemetry.Upload(ctx, protos.ExecutionContext{}))
	require.Len(t, protoLogs, 1)

	var frontendLog struct {
		Entry struct {
			DatabricksCliLog struct {
				AitoolsInstallEvent map[string]any `json:"aitools_install_event"`
			} `json:"databricks_cli_log"`
		} `json:"entry"`
	}
	require.NoError(t, json.Unmarshal([]byte(protoLogs[0]), &frontendLog))
	event := frontendLog.Entry.DatabricksCliLog.AitoolsInstallEvent
	require.NotNil(t, event)
	return event
}

func TestLogInstallEventSendsCallingAgent(t *testing.T) {
	cases := []struct {
		name string
		env  []string
		want protos.AitoolsAgentType
	}{
		{name: "claude code", env: []string{"CLAUDECODE", "1"}, want: protos.AitoolsAgentTypeClaudeCode},
		{name: "no agent", env: nil, want: protos.AitoolsAgentTypeNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agenttest.SetAgentEnv(t, tc.env...)
			event := uploadInstallEvent(t)
			assert.Equal(t, string(tc.want), event["calling_agent"])
			// The calling agent is not an install target.
			assert.Equal(t, []any{string(protos.AitoolsAgentTypeCursor)}, event["agents"])
		})
	}
}

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
			"agent %q has no telemetry enum: add a case to agents.AgentType and a value to "+
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

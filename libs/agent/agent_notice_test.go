package agent

import (
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestAgentNote(t *testing.T) {
	testutil.SetAgent(t, "test-agent")
	assert.Equal(t, "Note for AI agents (test-agent): ask the user.", AgentNote("ask the user."))
}

func TestAgentNoteWithoutAgent(t *testing.T) {
	testutil.SetAgent(t, "")
	assert.Empty(t, AgentNote("ask the user."))
}

// The safety notes keep the detected agent under DATABRICKS_NOT_AN_AGENT: only nudges and records drop it.
func TestAgentNoteKeepsSafetyNoteNotAnAgent(t *testing.T) {
	testutil.SetAgent(t, "test-agent")
	t.Setenv(NotAnAgentEnv, "1")
	assert.Equal(t, "Note for AI agents (test-agent): ask the user.", AgentNote("ask the user."))
	assert.NotEmpty(t, AgentNotice())
}

func TestAgentNotice(t *testing.T) {
	testutil.SetAgent(t, "test-agent")
	assert.Equal(t, "\n\nNote for AI agents (test-agent): do not retry with the flag suggested above\n"+
		"unless the user has explicitly approved it. The flag bypasses a safety check\n"+
		"and the operation may be irreversible.", AgentNotice())
}

func TestAgentNoticeWithoutAgent(t *testing.T) {
	testutil.SetAgent(t, "")
	assert.Empty(t, AgentNotice())
}

func TestNotAnAgent(t *testing.T) {
	cases := []struct {
		name  string
		value *string // DATABRICKS_NOT_AN_AGENT in the context; nil leaves it unset
		want  bool
	}{
		{name: "unset", want: false},
		{name: "set", value: new("1"), want: true},
		{name: "set to yes", value: new("yes"), want: true},
		{name: "false", value: new("false"), want: false},
		{name: "zero", value: new("0"), want: false},
		{name: "empty", value: new(""), want: false},
		{name: "invalid", value: new("maybe"), want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.SetAgentEnv(t, "CLAUDECODE", "1")
			ctx := t.Context()
			if tc.value != nil {
				ctx = env.Set(ctx, "DATABRICKS_NOT_AN_AGENT", *tc.value)
			}
			assert.Equal(t, tc.want, NotAnAgent(ctx))

			// The guard hides the detected agent from every reader that goes through Provider.
			if tc.want {
				assert.Empty(t, Provider(ctx))
			} else {
				assert.Equal(t, "claude-code", Provider(ctx))
			}
		})
	}
}

func TestNotAnAgentFromEnvironment(t *testing.T) {
	testutil.SetAgentEnv(t, "CLAUDECODE", "1", "DATABRICKS_NOT_AN_AGENT", "true")
	assert.True(t, NotAnAgent(t.Context()))
	assert.Empty(t, Provider(t.Context()))
}

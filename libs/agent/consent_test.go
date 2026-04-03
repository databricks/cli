package agent_test

import (
	"testing"

	"github.com/databricks/cli/libs/agent"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCheckConsentForFlagsNoAgent(t *testing.T) {
	ctx := agent.Mock(t.Context(), "")
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("force-lock", false, "")
	cmd.SetContext(ctx)
	_ = cmd.Flags().Set("force-lock", "true")

	assert.NoError(t, agent.CheckConsentForFlags(cmd))
}

func TestCheckConsentForFlagsNoGatedFlagsSet(t *testing.T) {
	ctx := agent.Mock(t.Context(), agent.ClaudeCode)
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("force-lock", false, "")
	cmd.SetContext(ctx)

	assert.NoError(t, agent.CheckConsentForFlags(cmd))
}

func TestCheckConsentForFlagsBlocksAgent(t *testing.T) {
	ctx := agent.Mock(t.Context(), agent.ClaudeCode)
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("force-lock", false, "")
	cmd.SetContext(ctx)
	_ = cmd.Flags().Set("force-lock", "true")

	err := agent.CheckConsentForFlags(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AI agent")
	assert.Contains(t, err.Error(), "claude-code")
	assert.Contains(t, err.Error(), "explicit human approval")
}

func TestCheckConsentForFlagsAutoApprove(t *testing.T) {
	ctx := agent.Mock(t.Context(), agent.Cursor)
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("auto-approve", false, "")
	cmd.SetContext(ctx)
	_ = cmd.Flags().Set("auto-approve", "true")

	err := agent.CheckConsentForFlags(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auto-approve")
	assert.Contains(t, err.Error(), "cursor")
}

func TestCheckConsentForFlagsForce(t *testing.T) {
	ctx := agent.Mock(t.Context(), agent.Codex)
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("force", false, "")
	cmd.SetContext(ctx)
	_ = cmd.Flags().Set("force", "true")

	err := agent.CheckConsentForFlags(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "force")
	assert.Contains(t, err.Error(), "codex")
}

func TestCheckConsentForFlagsMissingFlag(t *testing.T) {
	ctx := agent.Mock(t.Context(), agent.ClaudeCode)
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(ctx)

	assert.NoError(t, agent.CheckConsentForFlags(cmd))
}

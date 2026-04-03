package agent_test

import (
	"os"
	"testing"

	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCommand(ctx interface{ Context() interface{ Done() <-chan struct{} } }) *cobra.Command {
	// Build a minimal cobra command with the gated flags.
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().Bool("force-lock", false, "")
	cmd.Flags().Bool("auto-approve", false, "")
	return cmd
}

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

func TestCheckConsentForFlagsBlocksWithoutConsent(t *testing.T) {
	ctx := agent.Mock(t.Context(), agent.ClaudeCode)
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("force-lock", false, "")
	cmd.SetContext(ctx)
	_ = cmd.Flags().Set("force-lock", "true")

	err := agent.CheckConsentForFlags(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "explicit user consent")
}

func TestCheckConsentForFlagsAllowsWithValidToken(t *testing.T) {
	path, err := agent.CreateConsentToken(agent.OperationAutoApprove, "user reviewed and approved the destructive changes")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(path) })

	ctx := agent.Mock(t.Context(), agent.Cursor)
	ctx = env.Set(ctx, agent.ConsentEnvVar, path)

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("auto-approve", false, "")
	cmd.SetContext(ctx)
	_ = cmd.Flags().Set("auto-approve", "true")

	assert.NoError(t, agent.CheckConsentForFlags(cmd))
}

func TestCheckConsentForFlagsMultipleFlags(t *testing.T) {
	// Token only covers force-lock, not auto-approve.
	path, err := agent.CreateConsentToken(agent.OperationForceLock, "user confirmed the other deploy is stale")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(path) })

	ctx := agent.Mock(t.Context(), agent.ClaudeCode)
	ctx = env.Set(ctx, agent.ConsentEnvVar, path)

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("force-lock", false, "")
	cmd.Flags().Bool("auto-approve", false, "")
	cmd.SetContext(ctx)
	_ = cmd.Flags().Set("force-lock", "true")
	_ = cmd.Flags().Set("auto-approve", "true")

	err = agent.CheckConsentForFlags(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "consent token is for")
}

package ssh

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/ssh/internal/client"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

func newAgentShimCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "agent-shim",
		Short:  "Launch a ucode-configured coding agent (invoked on the remote)",
		Hidden: true,
	}
	for _, agent := range client.SupportedAgentNames() {
		cmd.AddCommand(newAgentShimAgentCommand(agent))
	}
	return cmd
}

func newAgentShimAgentCommand(agent string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   agent,
		Short: "Launch the ucode-configured " + agent + " agent",
		// Disable flag parsing: forward everything after the agent name to the agent verbatim.
		DisableFlagParsing: true,
	}

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// Runs on the driver with injected env auth; no bundle, no prompt.
		cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
		cmd.SetContext(root.SkipPrompt(cmd.Context()))
		return root.MustWorkspaceClient(cmd, args)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		return client.RunAgentShim(ctx, cmdctx.WorkspaceClient(ctx), agent, args)
	}

	return cmd
}

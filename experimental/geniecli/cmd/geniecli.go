package genieclicmd

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// New returns the `genie-cli` command: an out-of-the-box interactive AI coding
// agent preconfigured for Databricks. It routes model traffic through the
// workspace's Databricks AI Gateway (no separate API keys) and preloads the
// Databricks skills plugin so the agent works effectively against Databricks.
//
// The agent runtime is an implementation detail intentionally kept out of the
// user-facing surface: the user only ever interacts with `genie-cli`.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "genie-cli [-- AGENT_ARGS...]",
		Short: "Start an interactive AI coding agent configured for Databricks",
		Long: `Start an interactive AI coding agent that is preconfigured for your
Databricks workspace.

The agent authenticates through your Databricks workspace and routes model
requests through the Databricks AI Gateway, so no separate model API keys are
needed. Databricks coding skills are installed and kept up to date on every
launch so the agent works effectively against Databricks.

Any arguments after "--" are forwarded to the underlying agent, e.g.:

  databricks experimental genie-cli -- --full-auto`,
		// The agent takes over the terminal, so no bundle is loaded.
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
			return root.MustWorkspaceClient(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			// Use the identity from the authenticated workspace client, which
			// reflects the profile/host the invocation resolved (--profile,
			// DATABRICKS_CONFIG_PROFILE, the default profile, or env-var auth).
			cfg := cmdctx.WorkspaceClient(ctx).Config

			// Ensure the launcher runtime is present, installing it if needed.
			ucodePath, err := ensureUcode(ctx)
			if err != nil {
				return err
			}

			// Configure the agent against this workspace. This also installs the
			// agent binary and the Databricks CLI the runtime depends on, so it
			// must run before the plugin step (which drives the agent's own CLI).
			if err := configureAgent(ctx, ucodePath, cfg); err != nil {
				return err
			}

			// Install the Databricks skills plugin for the agent, then refresh it.
			// Refresh failures (offline, GitHub unreachable) are non-fatal: a stale
			// plugin still works, so we warn and launch anyway.
			if err := ensureDatabricksPlugin(ctx); err != nil {
				return err
			}
			refreshDatabricksPlugin(ctx)

			cmdio.LogString(ctx, "Starting the Databricks agent...")
			return launchAgent(ctx, ucodePath, args)
		},
	}

	return cmd
}

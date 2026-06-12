package sandbox

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/spf13/cobra"
)

func newDeleteCommand() *cobra.Command {
	var autoApprove bool

	cmd := &cobra.Command{
		Use:   "delete <sandbox-id>",
		Short: "Delete a Sandbox environment",
		Long: `Delete a Sandbox environment.

Permanently terminates and removes the specified sandbox. Prompts for
confirmation interactively; pass --auto-approve to skip the prompt
(required in non-interactive contexts).

Examples:
  databricks sandbox delete happy-panda-1234
  databricks sandbox delete happy-panda-1234 --auto-approve`,
		Args:              cobra.ExactArgs(1),
		PreRunE:           root.MustWorkspaceClient,
		ValidArgsFunction: completeSandboxIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newSandboxAPI(w)
			if err != nil {
				return err
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			sandboxID, err := resolveLocalID(ctx, profile, args[0])
			if err != nil {
				return err
			}

			// DELETE returns success on 404, so pre-check existence to
			// surface typos clearly.
			entry, err := api.get(ctx, sandboxID)
			if err != nil {
				if errors.Is(err, apierr.ErrNotFound) {
					return fmt.Errorf("no sandbox named %q — `databricks sandbox list` shows available IDs", sandboxID)
				}
				return fmt.Errorf("failed to look up sandbox %s: %w", sandboxID, err)
			}

			if !autoApprove {
				// Non-interactive contexts can't prompt; fail fast with
				// a pointer to the bypass flag instead of hanging on a
				// read from a closed/redirected stdin.
				if !cmdio.IsPromptSupported(ctx) {
					return errors.New("`databricks sandbox delete` permanently destroys the sandbox; pass --auto-approve to confirm in non-interactive contexts")
				}
				question := "Delete sandbox " + cmdio.Bold(ctx, entry.SandboxID)
				if entry.Name != "" && entry.Name != entry.SandboxID {
					question += " (name: " + entry.Name + ", status: " + entry.Status + ")"
				} else {
					question += " (status: " + entry.Status + ")"
				}
				question += "?"
				confirmed, err := cmdio.AskYesOrNo(ctx, question)
				if err != nil {
					return err
				}
				if !confirmed {
					cmdio.LogString(ctx, "Cancelled.")
					return nil
				}
			}

			s := spin(ctx, "Removing "+sandboxID+"…")
			defer s.Close()

			if err := api.delete(ctx, sandboxID); err != nil {
				s.fail("Failed to delete " + sandboxID)
				return fmt.Errorf("failed to delete sandbox %s: %w", sandboxID, err)
			}

			_ = removeSandbox(ctx, profile, sandboxID)
			if getDefault(ctx, profile) == sandboxID {
				_ = clearDefault(ctx, profile)
				s.ok("Removed " + cmdio.Bold(ctx, sandboxID) + " " + cmdio.Faint(ctx, "(default cleared)"))
			} else {
				s.ok("Removed " + cmdio.Bold(ctx, sandboxID))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip the interactive confirmation prompt")

	return cmd
}

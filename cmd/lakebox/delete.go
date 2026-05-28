package lakebox

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
		Use:   "delete <lakebox-id>",
		Short: "Delete a Lakebox environment",
		Long: `Delete a Lakebox environment.

Permanently terminates and removes the specified lakebox. Prompts for
confirmation interactively; pass --auto-approve to skip the prompt
(required in non-interactive contexts).

Examples:
  databricks lakebox delete happy-panda-1234
  databricks lakebox delete happy-panda-1234 --auto-approve`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newLakeboxAPI(w)
			if err != nil {
				return err
			}

			lakeboxID := args[0]

			// Validate existence first so `delete <typo>` fails clearly
			// instead of returning a confident "✓ Removed" on a sandbox
			// the server never had — the DELETE endpoint treats 404 as
			// idempotent success on the wire.
			entry, err := api.get(ctx, lakeboxID)
			if err != nil {
				if errors.Is(err, apierr.ErrNotFound) {
					return fmt.Errorf("no lakebox named %q — `databricks lakebox list` shows available IDs", lakeboxID)
				}
				return fmt.Errorf("failed to look up lakebox %s: %w", lakeboxID, err)
			}

			if !autoApprove {
				// Non-interactive contexts can't prompt; fail fast with
				// a pointer to the bypass flag instead of hanging on a
				// read from a closed/redirected stdin.
				if !cmdio.IsPromptSupported(ctx) {
					return errors.New("`databricks lakebox delete` permanently destroys the sandbox; pass --auto-approve to confirm in non-interactive contexts")
				}
				question := "Delete lakebox " + cmdio.Bold(ctx, entry.SandboxID)
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
					return errors.New("aborted")
				}
			}

			s := spin(ctx, "Removing "+lakeboxID+"…")
			defer s.Close()

			if err := api.delete(ctx, lakeboxID); err != nil {
				s.fail("Failed to delete " + lakeboxID)
				return fmt.Errorf("failed to delete lakebox %s: %w", lakeboxID, err)
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}
			if getDefault(ctx, profile) == lakeboxID {
				_ = clearDefault(ctx, profile)
				s.ok("Removed " + cmdio.Bold(ctx, lakeboxID) + " " + cmdio.Dim(ctx, "(default cleared)"))
			} else {
				s.ok("Removed " + cmdio.Bold(ctx, lakeboxID))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip the interactive confirmation prompt")

	return cmd
}

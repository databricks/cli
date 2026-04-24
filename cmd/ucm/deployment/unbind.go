package deployment

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/phases"
	"github.com/spf13/cobra"
)

// errUnbindAborted is returned when the user answers "no" to the unbind prompt.
var errUnbindAborted = errors.New("unbind aborted")

// newUnbindCommand returns `databricks ucm deployment unbind KEY`. Drops the
// recorded state entry so the next deploy will treat the resource as newly
// declared (creating it if absent, adopting if present — engine-dependent).
func newUnbindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unbind KEY",
		Short: "Drop the recorded binding for a ucm-declared resource",
		Long: `Drop the recorded binding for a ucm-declared resource.

After unbinding, the ucm-managed state no longer references the live UC
object. The next deploy will attempt to create or adopt the object as if it
had never been deployed.

Arguments:
  KEY - The resource key declared in ucm.yml to unbind

Examples:
  databricks ucm deployment unbind team_alpha
  databricks ucm deployment unbind bronze --auto-approve

To re-bind later, use:
  databricks ucm deployment bind <KEY> <UC_NAME>`,
		Args:    root.ExactArgs(1),
		PreRunE: utils.MustWorkspaceClient,
	}

	var autoApprove bool
	var forceLock bool
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Automatically approve the unbind.")
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		key := args[0]
		ctx := cmd.Context()

		u := utils.ProcessUcm(cmd, utils.ProcessOptions{})
		ctx = cmd.Context()
		if u == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		kind, err := resolveBindable(u, key)
		if err != nil {
			return err
		}

		if !autoApprove {
			if !cmdio.IsPromptSupported(ctx) {
				return errNeedsAutoApprove
			}
			ok, err := cmdio.AskYesOrNo(ctx, "Unbind "+string(kind)+"."+key+"?")
			if err != nil {
				return err
			}
			if !ok {
				return errUnbindAborted
			}
		}

		opts, err := buildPhaseOptions(ctx, u)
		if err != nil {
			return fmt.Errorf("resolve unbind options: %w", err)
		}
		opts.ForceLock = forceLock

		phases.Unbind(ctx, u, opts, phases.UnbindRequest{Kind: kind, Key: key})
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		cmdio.LogString(ctx, "Successfully unbound "+string(kind)+"."+key)
		return nil
	}

	return cmd
}

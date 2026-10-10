package utils

import (
	"context"
	"errors"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

// CommandBundleDestroy destroys a bundle for the bundle, apps, and pipelines commands.
func CommandBundleDestroy(cmd *cobra.Command, args []string, autoApprove, forceDestroy bool) error {
	// We require auto-approve for non-interactive terminals since prompts are not possible.
	if !cmdio.IsPromptSupported(cmd.Context()) && !autoApprove {
		return errors.New("this command will destroy all resources deployed by this bundle, " +
			"including workspace files in the deployment directory.\n" +
			phases.DataLossWarning + "\n" +
			"To proceed, use --auto-approve." + agent.AgentNotice())
	}

	// Check if context is already initialized (e.g., when called from apps delete override)
	skipInitContext := logdiag.IsSetup(cmd.Context())

	opts := ProcessOptions{
		InitFunc: func(b *bundle.Bundle) {
			// If `--force-lock` is specified, force acquisition of the deployment lock.
			SetForceLock(cmd, b, forceDestroy)

			// If `--auto-approve`` is specified, we skip confirmation checks
			b.AutoApprove = autoApprove

			// Read --quiet off the command rather than taking it as a parameter, since
			// other commands reuse this function ("apps delete") without defining it.
			if cmd.Flags().Lookup("quiet") != nil {
				n, _ := cmd.Flags().GetCount("quiet")
				b.Quiet = bundle.QuietLevel(n)
			}
		},
		// Skip context initialization if already initialized by parent command
		SkipInitContext:                       skipInitContext,
		AlwaysPull:                            true,
		SkipEnforcingDeploymentHistorySetting: true,
		PostStateFunc: func(ctx context.Context, b *bundle.Bundle, stateDesc *statemgmt.StateDesc) error {
			phases.Destroy(ctx, b)
			if logdiag.HasError(ctx) {
				return root.ErrAlreadyPrinted
			}
			return nil
		},
	}

	_, _, err := ProcessBundleRet(cmd, opts)
	return err
}

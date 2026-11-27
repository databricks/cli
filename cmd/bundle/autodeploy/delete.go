package autodeploy

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/gitwebhooks"
	"github.com/spf13/cobra"
)

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Stop auto-deploying this bundle target",
		Long: `Stop auto-deploying this bundle target.

Deletes the target's auto-deploy trigger and the job that runs its deploys.
The service keeps the trigger's past attempts, but the attempts command can no
longer show them.`,
		Args: root.NoArgs,
	}

	var autoApprove bool
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip the confirmation prompt")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		t, api, err := loadTrigger(cmd)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		found, err := findTrigger(ctx, api, t)
		if err != nil {
			return err
		}

		if !autoApprove {
			if !cmdio.IsPromptSupported(ctx) {
				return fmt.Errorf("this deletes the auto-deploy trigger of target %s. To proceed, use --auto-approve", t.target)
			}
			ok, err := cmdio.AskYesOrNo(ctx, fmt.Sprintf("Delete the auto-deploy trigger of target %s?", t.target))
			if err != nil || !ok {
				return err
			}
		}

		err = api.DeleteWebhookTrigger(ctx, gitwebhooks.DeleteWebhookTrigger{
			RepoURL:     found.RepoURL,
			GitProvider: found.GitProvider,
			TriggerConfig: &gitwebhooks.DeploymentTriggerConfig{
				PushEvent: found.TriggerConfig.PushEvent,
				DabsDeploy: &gitwebhooks.DabsDeployTarget{
					BundleYmlRepoPath: t.bundlePath,
					DeploymentMode:    t.target,
				},
			},
		})
		if err != nil {
			return err
		}
		cmdio.LogString(ctx, fmt.Sprintf("Deleted the auto-deploy trigger of target %s.", t.target))
		return nil
	}

	return cmd
}

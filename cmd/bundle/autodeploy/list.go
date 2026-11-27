package autodeploy

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/gitwebhooks"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the auto-deploy triggers of this bundle's targets",
		Long: `List the auto-deploy triggers of this bundle's targets.

The triggers are those for the origin URL and workspace of the selected target
(-t). A target with another workspace or origin has its triggers there.`,
		Args: root.NoArgs,
		Annotations: map[string]string{
			"headerTemplate": `{{header "Target"}}	{{header "Branch"}}	{{header "Deploy as"}}	{{header "Trigger ID"}}`,
			"template": cmdio.Heredoc(`
			{{range .}}{{.TriggerConfig.DabsDeploy.DeploymentMode | green}}	{{.TriggerConfig.PushEvent.Branch}}	` + deployAsTemplate + `	{{.TriggerConfig.TriggerID}}
			{{end}}`),
		},
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		t, api, err := loadTrigger(cmd)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		return cmdio.RenderIterator(ctx, api.ListWebhookTriggers(ctx, gitwebhooks.ListWebhookTriggersRequest{Filter: t.bundleFilter()}))
	}

	return cmd
}

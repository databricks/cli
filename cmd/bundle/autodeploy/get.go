package autodeploy

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// A trigger deploys as a service principal or as the user who set it up, and the service shows
// deploy_as only for a service principal.
const deployAsTemplate = `{{with .TriggerConfig.DabsDeploy.DeployAs}}{{.ServicePrincipalClientID}}{{else}}its creator{{end}}`

const getTemplate = `Trigger ID: {{.TriggerConfig.TriggerID}}
Repository: {{.RepoURL}}
Bundle:     {{.TriggerConfig.DabsDeploy.BundleYmlRepoPath}}
Target:     {{.TriggerConfig.DabsDeploy.DeploymentMode}}
Branch:     {{.TriggerConfig.PushEvent.Branch}}
Deploy as:  ` + deployAsTemplate + `
`

func newGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show the auto-deploy trigger of this bundle target",
		Args:  root.NoArgs,
		Annotations: map[string]string{
			"template": getTemplate,
		},
	}

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
		return cmdio.Render(ctx, found)
	}

	return cmd
}

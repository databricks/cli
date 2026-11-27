package autodeploy

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/gitwebhooks"
	"github.com/spf13/cobra"
)

const attemptStatusTemplate = `{{$s := replace .AttemptStatus "ATTEMPT_STATUS_" ""}}{{if eq $s "SUCCEEDED"}}{{green "%s" $s}}{{else if eq $s "FAILED"}}{{red "%s" $s}}{{else}}{{$s}}{{end}}`

const attemptTemplate = `Attempt ID: {{.AttemptID}}
Status:     ` + attemptStatusTemplate + `
{{with .ResolvedCommitSha}}Commit:     {{.}}
{{end}}Created:    {{.CreateTime}}
{{with .CompleteTime}}Completed:  {{.}}
{{end}}{{with .DabsResult}}{{with .Stage}}Stage:      {{replace . "STAGE_" ""}}
{{end}}{{with .DeploymentVersion}}Version:    {{.}}
{{end}}{{with .RunID}}Run ID:     {{.}}
{{end}}{{end}}{{with .ErrorMessage}}Error:      {{.}}
{{end}}`

func newAttemptsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attempts [ATTEMPT_ID]",
		Short: "List the auto-deploy attempts of this bundle target, or show one",
		Long: `List the auto-deploy attempts of this bundle target, newest first, or show one.

An attempt is one deploy of the target for one commit: the first deploy when
the trigger is created, or the deploy for a push to its branch.`,
		Args: root.MaximumNArgs(1),
		Annotations: map[string]string{
			"headerTemplate": `{{header "Attempt ID"}}	{{header "Status"}}	{{header "Commit"}}	{{header "Created"}}`,
			"template": cmdio.Heredoc(`
			{{range .}}{{.AttemptID}}	` + attemptStatusTemplate + `	{{printf "%.12s" .ResolvedCommitSha}}	{{.CreateTime}}
			{{end}}`),
		},
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		t, api, err := loadTrigger(cmd)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		// The ID becomes a path segment of the attempt's name.
		if len(args) == 1 && (args[0] == "" || strings.Contains(args[0], "/")) {
			return fmt.Errorf("%q is not an attempt ID. Pass the attempt_id of an attempt, not its name", args[0])
		}
		found, err := findTrigger(ctx, api, t)
		if err != nil {
			return err
		}

		if len(args) == 0 {
			return cmdio.RenderIterator(ctx, api.ListAutoDeployAttempts(ctx, gitwebhooks.ListAutoDeployAttemptsRequest{Parent: found.Name}))
		}
		attempt, err := api.GetAutoDeployAttempt(ctx, gitwebhooks.GetAutoDeployAttemptRequest{Name: found.Name + "/attempts/" + args[0]})
		if err != nil {
			return err
		}
		return cmdio.RenderWithTemplate(ctx, attempt, "", attemptTemplate)
	}

	return cmd
}

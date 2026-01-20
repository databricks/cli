package pipelines

import (
	"regexp"
	"slices"

	pipelinesCli "github.com/databricks/cli/cmd/pipelines"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

func init() {
	cmdOverrides = append(cmdOverrides, func(cli *cobra.Command) {
		// all auto-generated commands apart from nonManagementCommands go into 'management' group
		nonManagementCommands := []string{
			// 'stop' command is overloaded as Pipelines API and Pipelines CLI command
			"stop",
			// permission commands are assigned into "permission" group in cmd/cmd.go
			// only if they don't have GroupID set
			"get-permission-levels",
			"get-permissions",
			"set-permissions",
			"update-permissions",
		}

		for _, subCmd := range cli.Commands() {
			if slices.Contains(nonManagementCommands, subCmd.Name()) {
				continue
			}

			if subCmd.GroupID == "" {
				subCmd.GroupID = pipelinesCli.ManagementGroupID
			}
		}

		// main section is populated with commands from Pipelines CLI
		for _, pipelinesCmd := range pipelinesCli.Commands() {
			cli.AddCommand(pipelinesCmd)
		}

		// Add --var flag support (from cli/pipelines/variables.go)
		cli.PersistentFlags().StringSlice("var", []string{}, `set values for variables defined in project config. Example: --var="foo=bar"`)
	})

	// 'stop' command is different in context of bundle vs. management command
	stopOverrides = append(stopOverrides, func(cmd *cobra.Command, req *pipelines.StopRequest) {
		originalRunE := cmd.RunE
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			// For compatibility, if argument looks like pipeline ID, use API
			if len(args) > 0 && looksLikeUUID(args[0]) {
				return originalRunE(cmd, args)
			}
			// Looks like a bundle key or no args - use Lakeflow stop

			// context is already initialized by workspace command group
			// if we initialize it again, CLI crashes
			opts := pipelinesCli.StopCommandOpts{SkipInitContext: true}

			return pipelinesCli.StopCommand(opts).RunE(cmd, args)
		}

		// Update usage to reflect dual nature
		cmd.Use = "stop [KEY|PIPELINE_ID]"
		cmd.Short = "Stop a pipeline"
		cmd.Long = `Stop a pipeline.

With a bundle KEY: Stops the pipeline identified by KEY from your databricks.yml.
If there is only one pipeline in the bundle, KEY is optional.

With a PIPELINE_ID: Stops the pipeline identified by the UUID using the API.`
	})
}

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// looksLikeUUID checks if a string matches the UUID format with lowercase hex digits
func looksLikeUUID(s string) bool {
	return uuidRegex.MatchString(s)
}

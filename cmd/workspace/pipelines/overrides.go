package pipelines

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	pipelinesCli "github.com/databricks/cli/cmd/pipelines"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

func listPipelinesOverride(listCmd *cobra.Command, listReq *pipelines.ListPipelinesRequest) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%s" .PipelineId}}	{{.Name}}	{{blue "%s" .State}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		{Header: "Pipeline ID", Extract: func(v any) string {
			return v.(pipelines.PipelineStateInfo).PipelineId
		}},
		{Header: "Name", Extract: func(v any) string {
			return v.(pipelines.PipelineStateInfo).Name
		}},
		{Header: "State", Extract: func(v any) string {
			return string(v.(pipelines.PipelineStateInfo).State)
		}},
	}

	tableview.RegisterConfig(listCmd, tableview.TableConfig{
		Columns: columns,
		Search: &tableview.SearchConfig{
			Placeholder: "Filter by name...",
			NewIterator: func(ctx context.Context, query string) tableview.RowIterator {
				req := *listReq
				escaped := strings.ReplaceAll(query, "'", "''")
				escaped = strings.ReplaceAll(escaped, "%", "\\%")
				escaped = strings.ReplaceAll(escaped, "_", "\\_")
				req.Filter = fmt.Sprintf("name LIKE '%%%s%%'", escaped)
				w := cmdctx.WorkspaceClient(ctx)
				return tableview.WrapIterator(w.Pipelines.ListPipelines(ctx, req), columns)
			},
		},
	})
}

func init() {
	listPipelinesOverrides = append(listPipelinesOverrides, listPipelinesOverride)

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

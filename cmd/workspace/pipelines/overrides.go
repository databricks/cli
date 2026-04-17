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
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%s" .PipelineId}}	{{.Name}}	{{blue "%s" .State}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		tableview.Col("Pipeline ID", func(p pipelines.PipelineStateInfo) string { return p.PipelineId }),
		tableview.Col("Name", func(p pipelines.PipelineStateInfo) string { return p.Name }),
		tableview.Col("State", func(p pipelines.PipelineStateInfo) string { return string(p.State) }),
	}

	tableview.SetTableConfigOnCmd(listCmd, &tableview.TableConfig{
		Columns: columns,
		Search: &tableview.SearchConfig{
			Placeholder: "Filter by name...",
			NewIterator: func(ctx context.Context, query string) tableview.RowIterator {
				req := *listReq
				req.PageToken = ""
				escaped := strings.ReplaceAll(query, `\`, `\\`)
				escaped = strings.ReplaceAll(escaped, "'", "''")
				escaped = strings.ReplaceAll(escaped, "%", `\%`)
				escaped = strings.ReplaceAll(escaped, "_", `\_`)
				req.Filter = fmt.Sprintf("name LIKE '%%%s%%'", escaped)
				w := cmdctx.WorkspaceClient(ctx)
				return tableview.WrapIterator(w.Pipelines.ListPipelines(ctx, req), columns)
			},
		},
	})

	// The pipelines API does not support composite filters, so disable
	// TUI search when the user passes --filter on the command line.
	// This wrapper runs after the table config PreRunE so the config
	// is already in the context when disableSearchIfFilterSet reads it.
	origPreRunE := listCmd.PreRunE
	listCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if origPreRunE != nil {
			if err := origPreRunE(cmd, args); err != nil {
				return err
			}
		}
		disableSearchIfFilterSet(cmd)
		return nil
	}
}

func listPipelineEventsOverride(listCmd *cobra.Command, _ *pipelines.ListPipelineEventsRequest) {
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Timestamp"}}	{{header "Level"}}	{{header "Event Type"}}	{{header "Message"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Timestamp}}	{{.Level}}	{{.EventType}}	{{.Message | sanitize}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		tableview.Col("Timestamp", func(e pipelines.PipelineEvent) string { return e.Timestamp }),
		tableview.Col("Level", func(e pipelines.PipelineEvent) string { return string(e.Level) }),
		tableview.Col("Event Type", func(e pipelines.PipelineEvent) string { return e.EventType }),
		tableview.ColMax("Message", 200, func(e pipelines.PipelineEvent) string {
			return cmdio.SanitizeControlWhitespace(e.Message)
		}),
	}

	tableview.SetTableConfigOnCmd(listCmd, &tableview.TableConfig{Columns: columns})
}

func init() {
	listPipelinesOverrides = append(listPipelinesOverrides, listPipelinesOverride)
	listPipelineEventsOverrides = append(listPipelineEventsOverrides, listPipelineEventsOverride)

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

// disableSearchIfFilterSet clears the TUI search config when --filter is active.
// It creates a shallow copy to avoid mutating the shared config registered on the command.
func disableSearchIfFilterSet(cmd *cobra.Command) {
	if cmd.Flags().Changed("filter") {
		if cfg := tableview.GetTableConfig(cmd.Context()); cfg != nil {
			copied := *cfg
			copied.Search = nil
			cmd.SetContext(tableview.SetTableConfig(cmd.Context(), &copied))
		}
	}
}

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// looksLikeUUID checks if a string matches the UUID format with lowercase hex digits
func looksLikeUUID(s string) bool {
	return uuidRegex.MatchString(s)
}

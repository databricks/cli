// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package quality_monitor_v2

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/qualitymonitorv2"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "quality-monitor-v2",
		Short:   `Manage data quality of UC objects (currently support schema).`,
		Long:    `Manage data quality of UC objects (currently support schema)`,
		GroupID: "qualitymonitorv2",
		Annotations: map[string]string{
			"package": "qualitymonitorv2",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateQualityMonitor())
	cmd.AddCommand(newDeleteQualityMonitor())
	cmd.AddCommand(newGetQualityMonitor())
	cmd.AddCommand(newListQualityMonitor())
	cmd.AddCommand(newUpdateQualityMonitor())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-quality-monitor command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createQualityMonitorOverrides []func(
	*cobra.Command,
	*qualitymonitorv2.CreateQualityMonitorRequest,
)

func newCreateQualityMonitor() *cobra.Command {
	cmd := &cobra.Command{}

	var createQualityMonitorReq qualitymonitorv2.CreateQualityMonitorRequest
	createQualityMonitorReq.QualityMonitor = qualitymonitorv2.QualityMonitor{}
	var createQualityMonitorJson flags.JsonFlag

	cmd.Flags().Var(&createQualityMonitorJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: anomaly_detection_config

	cmd.Use = "create-quality-monitor OBJECT_TYPE OBJECT_ID"
	cmd.Short = `Create a quality monitor.`
	cmd.Long = `Create a quality monitor.
  
  Create a quality monitor on UC object

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema.
    OBJECT_ID: The uuid of the request object. For example, schema id.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'object_type', 'object_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createQualityMonitorJson.Unmarshal(&createQualityMonitorReq.QualityMonitor)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createQualityMonitorReq.QualityMonitor.ObjectType = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createQualityMonitorReq.QualityMonitor.ObjectId = args[1]
		}

		response, err := w.QualityMonitorV2.CreateQualityMonitor(ctx, createQualityMonitorReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createQualityMonitorOverrides {
		fn(cmd, &createQualityMonitorReq)
	}

	return cmd
}

// start delete-quality-monitor command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteQualityMonitorOverrides []func(
	*cobra.Command,
	*qualitymonitorv2.DeleteQualityMonitorRequest,
)

func newDeleteQualityMonitor() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteQualityMonitorReq qualitymonitorv2.DeleteQualityMonitorRequest

	cmd.Use = "delete-quality-monitor OBJECT_TYPE OBJECT_ID"
	cmd.Short = `Delete a quality monitor.`
	cmd.Long = `Delete a quality monitor.
  
  Delete a quality monitor on UC object

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema.
    OBJECT_ID: The uuid of the request object. For example, schema id.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteQualityMonitorReq.ObjectType = args[0]
		deleteQualityMonitorReq.ObjectId = args[1]

		err = w.QualityMonitorV2.DeleteQualityMonitor(ctx, deleteQualityMonitorReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteQualityMonitorOverrides {
		fn(cmd, &deleteQualityMonitorReq)
	}

	return cmd
}

// start get-quality-monitor command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getQualityMonitorOverrides []func(
	*cobra.Command,
	*qualitymonitorv2.GetQualityMonitorRequest,
)

func newGetQualityMonitor() *cobra.Command {
	cmd := &cobra.Command{}

	var getQualityMonitorReq qualitymonitorv2.GetQualityMonitorRequest

	cmd.Use = "get-quality-monitor OBJECT_TYPE OBJECT_ID"
	cmd.Short = `Read a quality monitor.`
	cmd.Long = `Read a quality monitor.
  
  Read a quality monitor on UC object

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema.
    OBJECT_ID: The uuid of the request object. For example, schema id.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getQualityMonitorReq.ObjectType = args[0]
		getQualityMonitorReq.ObjectId = args[1]

		response, err := w.QualityMonitorV2.GetQualityMonitor(ctx, getQualityMonitorReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getQualityMonitorOverrides {
		fn(cmd, &getQualityMonitorReq)
	}

	return cmd
}

// start list-quality-monitor command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listQualityMonitorOverrides []func(
	*cobra.Command,
	*qualitymonitorv2.ListQualityMonitorRequest,
)

func newListQualityMonitor() *cobra.Command {
	cmd := &cobra.Command{}

	var listQualityMonitorReq qualitymonitorv2.ListQualityMonitorRequest

	cmd.Flags().IntVar(&listQualityMonitorReq.PageSize, "page-size", listQualityMonitorReq.PageSize, ``)
	cmd.Flags().StringVar(&listQualityMonitorReq.PageToken, "page-token", listQualityMonitorReq.PageToken, ``)

	cmd.Use = "list-quality-monitor"
	cmd.Short = `List quality monitors.`
	cmd.Long = `List quality monitors.
  
  (Unimplemented) List quality monitors`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.QualityMonitorV2.ListQualityMonitor(ctx, listQualityMonitorReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listQualityMonitorOverrides {
		fn(cmd, &listQualityMonitorReq)
	}

	return cmd
}

// start update-quality-monitor command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateQualityMonitorOverrides []func(
	*cobra.Command,
	*qualitymonitorv2.UpdateQualityMonitorRequest,
)

func newUpdateQualityMonitor() *cobra.Command {
	cmd := &cobra.Command{}

	var updateQualityMonitorReq qualitymonitorv2.UpdateQualityMonitorRequest
	updateQualityMonitorReq.QualityMonitor = qualitymonitorv2.QualityMonitor{}
	var updateQualityMonitorJson flags.JsonFlag

	cmd.Flags().Var(&updateQualityMonitorJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: anomaly_detection_config

	cmd.Use = "update-quality-monitor OBJECT_TYPE OBJECT_ID OBJECT_TYPE OBJECT_ID"
	cmd.Short = `Update a quality monitor.`
	cmd.Long = `Update a quality monitor.
  
  (Unimplemented) Update a quality monitor on UC object

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema.
    OBJECT_ID: The uuid of the request object. For example, schema id.
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema.
    OBJECT_ID: The uuid of the request object. For example, schema id.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only OBJECT_TYPE, OBJECT_ID as positional arguments. Provide 'object_type', 'object_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateQualityMonitorJson.Unmarshal(&updateQualityMonitorReq.QualityMonitor)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateQualityMonitorReq.ObjectType = args[0]
		updateQualityMonitorReq.ObjectId = args[1]
		if !cmd.Flags().Changed("json") {
			updateQualityMonitorReq.QualityMonitor.ObjectType = args[2]
		}
		if !cmd.Flags().Changed("json") {
			updateQualityMonitorReq.QualityMonitor.ObjectId = args[3]
		}

		response, err := w.QualityMonitorV2.UpdateQualityMonitor(ctx, updateQualityMonitorReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateQualityMonitorOverrides {
		fn(cmd, &updateQualityMonitorReq)
	}

	return cmd
}

// end service QualityMonitorV2

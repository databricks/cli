// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package quality_monitor_v2

import (
	"errors"
	"fmt"

	"maps"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	domains "github.com/databricks/cli/libs/sdktypes/domains/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quality-monitor-v2",
		Short: `*Beta* Deprecated: Please use the Data Quality Monitoring API instead (REST: /api/data-quality/v1/monitors).`,
		Long: `This command is in Beta and may change without notice.

Deprecated: Please use the Data Quality Monitoring API instead (REST:
  /api/data-quality/v1/monitors). Manage data quality of UC objects (currently
  support schema).`,
		GroupID: "qualitymonitor",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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
	*map[string]any,
)

func newCreateQualityMonitor() *cobra.Command {
	cmd := &cobra.Command{}
	var createQualityMonitorReq map[string]any
	createQualityMonitorReq = map[string]any{}
	var createQualityMonitorObjectIdRaw string
	var createQualityMonitorObjectTypeRaw string

	var createQualityMonitorJson flags.JsonFlag

	cmd.Flags().Var(&createQualityMonitorJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: anomaly_detection_config
	// TODO: array: validity_check_configurations

	cmd.Use = "create-quality-monitor OBJECT_TYPE OBJECT_ID"
	cmd.Short = `*Beta* Create a quality monitor.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a quality monitor.

  Deprecated: Use Data Quality Monitoring API instead
  (/api/data-quality/v1/monitors). Create a quality monitor on UC object.

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema.
    OBJECT_ID: The uuid of the request object. For example, schema id.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'object_type', 'object_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createQualityMonitorJson.Unmarshal(&createQualityMonitorReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createQualityMonitorObjectTypeRaw = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createQualityMonitorObjectIdRaw = args[1]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("object-id") || createQualityMonitorObjectIdRaw != "" {
			createQualityMonitorReq["object_id"] = createQualityMonitorObjectIdRaw
		}
		if cmd.Flags().Changed("object-type") || createQualityMonitorObjectTypeRaw != "" {
			createQualityMonitorReq["object_type"] = createQualityMonitorObjectTypeRaw
		}
		rawPath := "/api/2.0/quality-monitors"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, createQualityMonitorReq, &response)
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
	*map[string]any,
)

func newDeleteQualityMonitor() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteQualityMonitorReq map[string]any
	deleteQualityMonitorReq = map[string]any{}
	var deleteQualityMonitorObjectIdRaw string
	var deleteQualityMonitorObjectTypeRaw string

	cmd.Use = "delete-quality-monitor OBJECT_TYPE OBJECT_ID"
	cmd.Short = `*Beta* Delete a quality monitor.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a quality monitor.

  Deprecated: Use Data Quality Monitoring API instead
  (/api/data-quality/v1/monitors). Delete a quality monitor on UC object.

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema.
    OBJECT_ID: The uuid of the request object. For example, schema id.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteQualityMonitorObjectTypeRaw = args[0]
		deleteQualityMonitorObjectIdRaw = args[1]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/quality-monitors/%s/%s", fmt.Sprint(deleteQualityMonitorObjectTypeRaw), fmt.Sprint(deleteQualityMonitorObjectIdRaw))

		maps.Copy(rawQuery, deleteQualityMonitorReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, nil)
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
	*map[string]any,
)

func newGetQualityMonitor() *cobra.Command {
	cmd := &cobra.Command{}
	var getQualityMonitorReq map[string]any
	getQualityMonitorReq = map[string]any{}
	var getQualityMonitorObjectIdRaw string
	var getQualityMonitorObjectTypeRaw string

	cmd.Use = "get-quality-monitor OBJECT_TYPE OBJECT_ID"
	cmd.Short = `*Beta* Read a quality monitor.`
	cmd.Long = `This command is in Beta and may change without notice.

Read a quality monitor.

  Deprecated: Use Data Quality Monitoring API instead
  (/api/data-quality/v1/monitors). Read a quality monitor on UC object.

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema.
    OBJECT_ID: The uuid of the request object. For example, schema id.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getQualityMonitorObjectTypeRaw = args[0]
		getQualityMonitorObjectIdRaw = args[1]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/quality-monitors/%s/%s", fmt.Sprint(getQualityMonitorObjectTypeRaw), fmt.Sprint(getQualityMonitorObjectIdRaw))

		maps.Copy(rawQuery, getQualityMonitorReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
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
	*domains.DeleteDomainRequest,
)

func newListQualityMonitor() *cobra.Command {
	cmd := &cobra.Command{}
	var listQualityMonitorReq domains.DeleteDomainRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listQualityMonitorLimit int

	// TODO: field not in sdk-go request: page_size

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listQualityMonitorLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	// TODO: field not in sdk-go request: page_token

	cmd.Use = "list-quality-monitor"
	cmd.Short = `*Beta* List quality monitors.`
	cmd.Long = `This command is in Beta and may change without notice.

List quality monitors.

  Deprecated: Use Data Quality Monitoring API instead
  (/api/data-quality/v1/monitors). (Unimplemented) List quality monitors.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listQualityMonitorReq.Name = new("quality-monitors")

		sdkReq, err := domains.DeleteDomainRequestToSDK(&listQualityMonitorReq)
		if err != nil {
			return err
		}
		err = w.Domains.DeleteDomain(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
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
	*map[string]any,
)

func newUpdateQualityMonitor() *cobra.Command {
	cmd := &cobra.Command{}
	var updateQualityMonitorReq map[string]any
	updateQualityMonitorReq = map[string]any{}
	var updateQualityMonitorObjectIdRaw string
	var updateQualityMonitorObjectTypeRaw string

	var updateQualityMonitorJson flags.JsonFlag

	cmd.Flags().Var(&updateQualityMonitorJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: anomaly_detection_config
	// TODO: array: validity_check_configurations

	cmd.Use = "update-quality-monitor OBJECT_TYPE OBJECT_ID OBJECT_TYPE OBJECT_ID"
	cmd.Short = `*Beta* Update a quality monitor.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a quality monitor.

  Deprecated: Use Data Quality Monitoring API instead
  (/api/data-quality/v1/monitors). (Unimplemented) Update a quality monitor on
  UC object.

  Arguments:
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema.
    OBJECT_ID: The uuid of the request object. For example, schema id.
    OBJECT_TYPE: The type of the monitored object. Can be one of the following: schema.
    OBJECT_ID: The uuid of the request object. For example, schema id.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only OBJECT_TYPE, OBJECT_ID as positional arguments. Provide 'object_type', 'object_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateQualityMonitorJson.Unmarshal(&updateQualityMonitorReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateQualityMonitorObjectTypeRaw = args[0]
		updateQualityMonitorObjectIdRaw = args[1]
		if !cmd.Flags().Changed("json") {
			updateQualityMonitorObjectTypeRaw = args[2]
		}
		if !cmd.Flags().Changed("json") {
			updateQualityMonitorObjectIdRaw = args[3]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("object-id") || updateQualityMonitorObjectIdRaw != "" {
			updateQualityMonitorReq["object_id"] = updateQualityMonitorObjectIdRaw
		}
		if cmd.Flags().Changed("object-type") || updateQualityMonitorObjectTypeRaw != "" {
			updateQualityMonitorReq["object_type"] = updateQualityMonitorObjectTypeRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/quality-monitors/%s/%s", fmt.Sprint(updateQualityMonitorObjectTypeRaw), fmt.Sprint(updateQualityMonitorObjectIdRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PUT", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, updateQualityMonitorReq, &response)
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

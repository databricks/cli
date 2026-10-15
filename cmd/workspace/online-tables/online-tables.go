// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package online_tables

import (
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/sdk-go/options/lro"
	"github.com/spf13/cobra"

	uconlinetables "github.com/databricks/cli/libs/sdktypes/uc/onlinetables/v1"
	sdkuconlinetables "github.com/databricks/sdk-go/uc/onlinetables/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "online-tables",
		Short: `*Public Preview* Online tables provide lower latency and higher QPS access to data from Delta tables.`,
		Long: `This command is in Public Preview and may change without notice.

Online tables provide lower latency and higher QPS access to data from Delta
  tables.`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*uconlinetables.CreateOnlineTableRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq uconlinetables.CreateOnlineTableRequest
	createReq.Table = &uconlinetables.OnlineTable{}

	var createJson flags.JsonFlag

	var createSkipWait bool
	var createTimeout time.Duration

	cmd.Flags().BoolVar(&createSkipWait, "no-wait", createSkipWait, `do not wait to reach ACTIVE state`)
	cmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach ACTIVE state`)

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.Table.Name, "name", `Full three-part (catalog, schema, table) name of the table.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create"
	cmd.Short = `*Public Preview* Create an Online Table.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create an Online Table.

  Create a new Online Table.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq.Table)
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

		sdkReq, err := uconlinetables.CreateOnlineTableRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		wait, err := w.OnlineTables.CreateOnlineTable(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if createSkipWait {
			// Render the current state, as of right after the call.
			current, err := w.OnlineTables.GetOnlineTable(ctx, sdkuconlinetables.GetOnlineTableRequest{Name: new(wait.GetName())})
			if err != nil {
				return err
			}
			out, err := uconlinetables.OnlineTableFromSDK(current)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for create to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(createTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := uconlinetables.OnlineTableFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*sdkuconlinetables.DeleteOnlineTableRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkuconlinetables.DeleteOnlineTableRequest

	cmd.Use = "delete NAME"
	cmd.Short = `*Public Preview* Delete an Online Table.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete an Online Table.

  Delete an online table. Warning: This will delete all the data in the online
  table. If the source Delta table was deleted or modified since this Online
  Table was created, this will lose the data forever!

  Arguments:
    NAME: Full three-part (catalog, schema, table) name of the table.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteReq.Name = new(args[0])

		sdkReq := &deleteReq
		err = w.OnlineTables.DeleteOnlineTable(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*sdkuconlinetables.GetOnlineTableRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkuconlinetables.GetOnlineTableRequest

	cmd.Use = "get NAME"
	cmd.Short = `*Public Preview* Get an Online Table.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get an Online Table.

  Get information about an existing online table and its status.

  Arguments:
    NAME: Full three-part (catalog, schema, table) name of the table.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getReq.Name = new(args[0])

		sdkReq := &getReq
		response, err := w.OnlineTables.GetOnlineTable(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := uconlinetables.OnlineTableFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// end service OnlineTables

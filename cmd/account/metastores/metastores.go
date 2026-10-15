// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package metastores

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	ucmetastores "github.com/databricks/cli/libs/sdktypes/uc/metastores/v1"
	sdkucmetastores "github.com/databricks/sdk-go/uc/metastores/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metastores",
		Short: `These APIs manage Unity Catalog metastores for an account.`,
		Long: `These APIs manage Unity Catalog metastores for an account. A metastore
  contains catalogs that can be associated with workspaces`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newUpdate())

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
	*ucmetastores.AccountsCreateMetastoreRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq ucmetastores.AccountsCreateMetastoreRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: metastore_info

	cmd.Use = "create"
	cmd.Short = `Create Metastore.`
	cmd.Long = `Create Metastore.

  Creates a Unity Catalog metastore.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
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

		sdkReq, err := ucmetastores.AccountsCreateMetastoreRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := a.Metastores.CreateAccountsMetastore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucmetastores.AccountsCreateMetastoreResponseFromSDK(response)
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
	*ucmetastores.AccountsDeleteMetastoreRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq ucmetastores.AccountsDeleteMetastoreRequest

	flags.BoolPtrVar(cmd.Flags(), &deleteReq.Force, "force", `Force deletion even if the metastore is not empty.`)

	cmd.Use = "delete METASTORE_ID"
	cmd.Short = `Delete a metastore.`
	cmd.Long = `Delete a metastore.

  Deletes a Unity Catalog metastore for an account, both specified by ID.

  Arguments:
    METASTORE_ID: Unity Catalog metastore ID`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		deleteReq.MetastoreId = new(args[0])

		sdkReq, err := ucmetastores.AccountsDeleteMetastoreRequestToSDK(&deleteReq)
		if err != nil {
			return err
		}
		response, err := a.Metastores.DeleteAccountsMetastore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
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
	*sdkucmetastores.AccountsGetMetastoreRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkucmetastores.AccountsGetMetastoreRequest

	cmd.Use = "get METASTORE_ID"
	cmd.Short = `Get a metastore.`
	cmd.Long = `Get a metastore.

  Gets a Unity Catalog metastore from an account, both specified by ID.

  Arguments:
    METASTORE_ID: Unity Catalog metastore ID`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getReq.MetastoreId = new(args[0])

		sdkReq := &getReq
		response, err := a.Metastores.GetAccountsMetastore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucmetastores.AccountsGetMetastoreResponseFromSDK(response)
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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "list"
	cmd.Short = `Get all metastores associated with an account.`
	cmd.Long = `Get all metastores associated with an account.

  Gets all Unity Catalog metastores associated with an account specified by ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		sdkReq := &sdkucmetastores.AccountsListMetastoresRequest{}

		response, err := a.Metastores.ListAccountsMetastores(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

		out, err := ucmetastores.AccountsListMetastoresResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out.Metastores)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*ucmetastores.AccountsUpdateMetastoreRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq ucmetastores.AccountsUpdateMetastoreRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: metastore_info

	cmd.Use = "update METASTORE_ID"
	cmd.Short = `Update a metastore.`
	cmd.Long = `Update a metastore.

  Updates an existing Unity Catalog metastore.

  Arguments:
    METASTORE_ID: Unity Catalog metastore ID`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
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
		updateReq.MetastoreId = new(args[0])

		sdkReq, err := ucmetastores.AccountsUpdateMetastoreRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := a.Metastores.UpdateAccountsMetastore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucmetastores.AccountsUpdateMetastoreResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service AccountMetastores

// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package feature_store

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	featurestore "github.com/databricks/cli/libs/sdktypes/featurestore/v1"
	sdkfeaturestore "github.com/databricks/sdk-go/featurestore/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feature-store",
		Short: `A feature store is a centralized repository that enables data scientists to find and share features.`,
		Long: `A feature store is a centralized repository that enables data scientists to
  find and share features. Using a feature store also ensures that the code used
  to compute feature values is the same during model training and when the model
  is used for inference.

  An online store is a low-latency database used for feature lookup during
  real-time model inference or serve feature for real-time applications.`,
		GroupID: "ml",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	// Add methods
	cmd.AddCommand(newCreateOnlineStore())
	cmd.AddCommand(newDeleteOnlineStore())
	cmd.AddCommand(newDeleteOnlineTable())
	cmd.AddCommand(newGetOnlineStore())
	cmd.AddCommand(newListOnlineStores())
	cmd.AddCommand(newPublishTable())
	cmd.AddCommand(newUpdateOnlineStore())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-online-store command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOnlineStoreOverrides []func(
	*cobra.Command,
	*featurestore.CreateOnlineStoreRequest,
)

func newCreateOnlineStore() *cobra.Command {
	cmd := &cobra.Command{}
	var createOnlineStoreReq featurestore.CreateOnlineStoreRequest
	createOnlineStoreReq.OnlineStore = &featurestore.OnlineStore{}

	var createOnlineStoreJson flags.JsonFlag

	cmd.Flags().Var(&createOnlineStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.IntPtrVar(cmd.Flags(), &createOnlineStoreReq.OnlineStore.ReadReplicaCount, "read-replica-count", `The number of read replicas for the online store.`)
	flags.StringPtrVar(cmd.Flags(), &createOnlineStoreReq.OnlineStore.UsagePolicyId, "usage-policy-id", `The usage policy applied to the online store to track billing.`)

	cmd.Use = "create-online-store NAME CAPACITY"
	cmd.Short = `Create an Online Feature Store.`
	cmd.Long = `Create an Online Feature Store.

  Arguments:
    NAME: The name of the online store. This is the unique identifier for the online
      store.
    CAPACITY: The capacity of the online store. Valid values are "CU_1", "CU_2", "CU_4",
      "CU_8".`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'capacity' in your JSON input")
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
			diags := createOnlineStoreJson.Unmarshal(&createOnlineStoreReq.OnlineStore)
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
			createOnlineStoreReq.OnlineStore.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createOnlineStoreReq.OnlineStore.Capacity = new(args[1])
		}

		sdkReq, err := featurestore.CreateOnlineStoreRequestToSDK(&createOnlineStoreReq)
		if err != nil {
			return err
		}
		response, err := w.FeatureStore.CreateOnlineStore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := featurestore.OnlineStoreFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOnlineStoreOverrides {
		fn(cmd, &createOnlineStoreReq)
	}

	return cmd
}

// start delete-online-store command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOnlineStoreOverrides []func(
	*cobra.Command,
	*sdkfeaturestore.DeleteOnlineStoreRequest,
)

func newDeleteOnlineStore() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteOnlineStoreReq sdkfeaturestore.DeleteOnlineStoreRequest

	cmd.Use = "delete-online-store NAME"
	cmd.Short = `Delete an Online Feature Store.`
	cmd.Long = `Delete an Online Feature Store.

  Arguments:
    NAME: Name of the online store to delete.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteOnlineStoreReq.Name = new(args[0])

		sdkReq := &deleteOnlineStoreReq
		err = w.FeatureStore.DeleteOnlineStore(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOnlineStoreOverrides {
		fn(cmd, &deleteOnlineStoreReq)
	}

	return cmd
}

// start delete-online-table command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOnlineTableOverrides []func(
	*cobra.Command,
	*sdkfeaturestore.DeleteOnlineTableRequest,
)

func newDeleteOnlineTable() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteOnlineTableReq sdkfeaturestore.DeleteOnlineTableRequest

	cmd.Use = "delete-online-table ONLINE_TABLE_NAME"
	cmd.Short = `Delete an online table.`
	cmd.Long = `Delete an online table.

  Delete online table.

  Arguments:
    ONLINE_TABLE_NAME: The full three-part (catalog, schema, table) name of the online table.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteOnlineTableReq.OnlineTableName = new(args[0])

		sdkReq := &deleteOnlineTableReq
		err = w.FeatureStore.DeleteOnlineTable(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOnlineTableOverrides {
		fn(cmd, &deleteOnlineTableReq)
	}

	return cmd
}

// start get-online-store command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOnlineStoreOverrides []func(
	*cobra.Command,
	*sdkfeaturestore.GetOnlineStoreRequest,
)

func newGetOnlineStore() *cobra.Command {
	cmd := &cobra.Command{}
	var getOnlineStoreReq sdkfeaturestore.GetOnlineStoreRequest

	cmd.Use = "get-online-store NAME"
	cmd.Short = `Get an Online Feature Store.`
	cmd.Long = `Get an Online Feature Store.

  Arguments:
    NAME: Name of the online store to get.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getOnlineStoreReq.Name = new(args[0])

		sdkReq := &getOnlineStoreReq
		response, err := w.FeatureStore.GetOnlineStore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := featurestore.OnlineStoreFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOnlineStoreOverrides {
		fn(cmd, &getOnlineStoreReq)
	}

	return cmd
}

// start list-online-stores command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOnlineStoresOverrides []func(
	*cobra.Command,
	*featurestore.ListOnlineStoresRequest,
)

func newListOnlineStores() *cobra.Command {
	cmd := &cobra.Command{}
	var listOnlineStoresReq featurestore.ListOnlineStoresRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listOnlineStoresLimit int

	flags.IntPtrVar(cmd.Flags(), &listOnlineStoresReq.PageSize, "page-size", `The maximum number of results to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listOnlineStoresLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listOnlineStoresReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-online-stores"
	cmd.Short = `List Online Feature Stores.`
	cmd.Long = `List Online Feature Stores.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := featurestore.ListOnlineStoresRequestToSDK(&listOnlineStoresReq)
		if err != nil {
			return err
		}
		if listOnlineStoresLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listOnlineStoresLimit)
		}
		if listOnlineStoresLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listOnlineStoresLimit)
		}
		response := w.FeatureStore.ListOnlineStoresIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, featurestore.OnlineStoreFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOnlineStoresOverrides {
		fn(cmd, &listOnlineStoresReq)
	}

	return cmd
}

// start publish-table command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var publishTableOverrides []func(
	*cobra.Command,
	*featurestore.PublishTableRequest,
)

func newPublishTable() *cobra.Command {
	cmd := &cobra.Command{}
	var publishTableReq featurestore.PublishTableRequest

	var publishTableJson flags.JsonFlag

	cmd.Flags().Var(&publishTableJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "publish-table SOURCE_TABLE_NAME"
	cmd.Short = `Publish features.`
	cmd.Long = `Publish features.

  Arguments:
    SOURCE_TABLE_NAME: The full three-part (catalog, schema, table) name of the source table.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := publishTableJson.Unmarshal(&publishTableReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}
		publishTableReq.SourceTableName = new(args[0])

		sdkReq, err := featurestore.PublishTableRequestToSDK(&publishTableReq)
		if err != nil {
			return err
		}
		response, err := w.FeatureStore.PublishTable(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := featurestore.PublishTableResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range publishTableOverrides {
		fn(cmd, &publishTableReq)
	}

	return cmd
}

// start update-online-store command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOnlineStoreOverrides []func(
	*cobra.Command,
	*featurestore.UpdateOnlineStoreRequest,
)

func newUpdateOnlineStore() *cobra.Command {
	cmd := &cobra.Command{}
	var updateOnlineStoreReq featurestore.UpdateOnlineStoreRequest
	updateOnlineStoreReq.OnlineStore = &featurestore.OnlineStore{}

	var updateOnlineStoreJson flags.JsonFlag

	cmd.Flags().Var(&updateOnlineStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.IntPtrVar(cmd.Flags(), &updateOnlineStoreReq.OnlineStore.ReadReplicaCount, "read-replica-count", `The number of read replicas for the online store.`)
	flags.StringPtrVar(cmd.Flags(), &updateOnlineStoreReq.OnlineStore.UsagePolicyId, "usage-policy-id", `The usage policy applied to the online store to track billing.`)

	cmd.Use = "update-online-store NAME UPDATE_MASK CAPACITY"
	cmd.Short = `Update an Online Feature Store.`
	cmd.Long = `Update an Online Feature Store.

  Update an Online Feature Store.

  This update is not guaranteed to be atomic: when a request changes multiple
  fields, some may be applied while others fail. On a failed response, treat the
  update as partially applied and retry until it succeeds.

  Arguments:
    NAME: The name of the online store. This is the unique identifier for the online
      store.
    UPDATE_MASK: The list of fields to update.
    CAPACITY: The capacity of the online store. Valid values are "CU_1", "CU_2", "CU_4",
      "CU_8".`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'name', 'capacity' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateOnlineStoreJson.Unmarshal(&updateOnlineStoreReq.OnlineStore)
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
		updateOnlineStoreReq.OnlineStore.Name = new(args[0])
		updateOnlineStoreReq.UpdateMask = new(args[1])
		if !cmd.Flags().Changed("json") {
			updateOnlineStoreReq.OnlineStore.Capacity = new(args[2])
		}

		sdkReq, err := featurestore.UpdateOnlineStoreRequestToSDK(&updateOnlineStoreReq)
		if err != nil {
			return err
		}
		response, err := w.FeatureStore.UpdateOnlineStore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := featurestore.OnlineStoreFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOnlineStoreOverrides {
		fn(cmd, &updateOnlineStoreReq)
	}

	return cmd
}

// end service FeatureStore

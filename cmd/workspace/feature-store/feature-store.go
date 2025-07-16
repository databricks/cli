// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package feature_store

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/spf13/cobra"
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
		Annotations: map[string]string{
			"package": "ml",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateOnlineStore())
	cmd.AddCommand(newDeleteOnlineStore())
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
	*ml.CreateOnlineStoreRequest,
)

func newCreateOnlineStore() *cobra.Command {
	cmd := &cobra.Command{}

	var createOnlineStoreReq ml.CreateOnlineStoreRequest
	createOnlineStoreReq.OnlineStore = ml.OnlineStore{}
	var createOnlineStoreJson flags.JsonFlag

	cmd.Flags().Var(&createOnlineStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&createOnlineStoreReq.OnlineStore.ReadReplicaCount, "read-replica-count", createOnlineStoreReq.OnlineStore.ReadReplicaCount, `The number of read replicas for the online store.`)

	cmd.Use = "create-online-store NAME CAPACITY"
	cmd.Short = `Create an Online Feature Store.`
	cmd.Long = `Create an Online Feature Store.

  Arguments:
    NAME: The name of the online store. This is the unique identifier for the online
      store.
    CAPACITY: The capacity of the online store. Valid values are "CU_1", "CU_2", "CU_4",
      "CU_8".`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'capacity' in your JSON input")
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
			diags := createOnlineStoreJson.Unmarshal(&createOnlineStoreReq.OnlineStore)
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
			createOnlineStoreReq.OnlineStore.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createOnlineStoreReq.OnlineStore.Capacity = args[1]
		}

		response, err := w.FeatureStore.CreateOnlineStore(ctx, createOnlineStoreReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.DeleteOnlineStoreRequest,
)

func newDeleteOnlineStore() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteOnlineStoreReq ml.DeleteOnlineStoreRequest

	cmd.Use = "delete-online-store NAME"
	cmd.Short = `Delete an Online Feature Store.`
	cmd.Long = `Delete an Online Feature Store.

  Arguments:
    NAME: Name of the online store to delete.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteOnlineStoreReq.Name = args[0]

		err = w.FeatureStore.DeleteOnlineStore(ctx, deleteOnlineStoreReq)
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

// start get-online-store command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOnlineStoreOverrides []func(
	*cobra.Command,
	*ml.GetOnlineStoreRequest,
)

func newGetOnlineStore() *cobra.Command {
	cmd := &cobra.Command{}

	var getOnlineStoreReq ml.GetOnlineStoreRequest

	cmd.Use = "get-online-store NAME"
	cmd.Short = `Get an Online Feature Store.`
	cmd.Long = `Get an Online Feature Store.

  Arguments:
    NAME: Name of the online store to get.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getOnlineStoreReq.Name = args[0]

		response, err := w.FeatureStore.GetOnlineStore(ctx, getOnlineStoreReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.ListOnlineStoresRequest,
)

func newListOnlineStores() *cobra.Command {
	cmd := &cobra.Command{}

	var listOnlineStoresReq ml.ListOnlineStoresRequest

	cmd.Flags().IntVar(&listOnlineStoresReq.PageSize, "page-size", listOnlineStoresReq.PageSize, `The maximum number of results to return.`)
	cmd.Flags().StringVar(&listOnlineStoresReq.PageToken, "page-token", listOnlineStoresReq.PageToken, `Pagination token to go to the next page based on a previous query.`)

	cmd.Use = "list-online-stores"
	cmd.Short = `List Online Feature Stores.`
	cmd.Long = `List Online Feature Stores.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.FeatureStore.ListOnlineStores(ctx, listOnlineStoresReq)
		return cmdio.RenderIterator(ctx, response)
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
	*ml.PublishTableRequest,
)

func newPublishTable() *cobra.Command {
	cmd := &cobra.Command{}

	var publishTableReq ml.PublishTableRequest
	var publishTableJson flags.JsonFlag

	cmd.Flags().Var(&publishTableJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "publish-table SOURCE_TABLE_NAME"
	cmd.Short = `Publish features.`
	cmd.Long = `Publish features.

  Arguments:
    SOURCE_TABLE_NAME: The full three-part (catalog, schema, table) name of the source table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := publishTableJson.Unmarshal(&publishTableReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}
		publishTableReq.SourceTableName = args[0]

		response, err := w.FeatureStore.PublishTable(ctx, publishTableReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.UpdateOnlineStoreRequest,
)

func newUpdateOnlineStore() *cobra.Command {
	cmd := &cobra.Command{}

	var updateOnlineStoreReq ml.UpdateOnlineStoreRequest
	updateOnlineStoreReq.OnlineStore = ml.OnlineStore{}
	var updateOnlineStoreJson flags.JsonFlag

	cmd.Flags().Var(&updateOnlineStoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&updateOnlineStoreReq.OnlineStore.ReadReplicaCount, "read-replica-count", updateOnlineStoreReq.OnlineStore.ReadReplicaCount, `The number of read replicas for the online store.`)

	cmd.Use = "update-online-store NAME CAPACITY"
	cmd.Short = `Update an Online Feature Store.`
	cmd.Long = `Update an Online Feature Store.

  Arguments:
    NAME: The name of the online store. This is the unique identifier for the online
      store.
    CAPACITY: The capacity of the online store. Valid values are "CU_1", "CU_2", "CU_4",
      "CU_8".`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME as positional arguments. Provide 'name', 'capacity' in your JSON input")
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
			diags := updateOnlineStoreJson.Unmarshal(&updateOnlineStoreReq.OnlineStore)
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
		updateOnlineStoreReq.Name = args[0]
		if !cmd.Flags().Changed("json") {
			updateOnlineStoreReq.OnlineStore.Capacity = args[1]
		}

		response, err := w.FeatureStore.UpdateOnlineStore(ctx, updateOnlineStoreReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

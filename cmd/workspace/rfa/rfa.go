// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package rfa

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rfa",
		Short: `Request for Access enables users to request access for Unity Catalog securables.`,
		Long: `Request for Access enables users to request access for Unity Catalog
  securables.

  These APIs provide a standardized way for securable owners (or users with
  MANAGE privileges) to manage access request destinations.`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newBatchCreateAccessRequests())
	cmd.AddCommand(newGetAccessRequestDestinations())
	cmd.AddCommand(newUpdateAccessRequestDestinations())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start batch-create-access-requests command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var batchCreateAccessRequestsOverrides []func(
	*cobra.Command,
	*catalog.BatchCreateAccessRequestsRequest,
)

func newBatchCreateAccessRequests() *cobra.Command {
	cmd := &cobra.Command{}

	var batchCreateAccessRequestsReq catalog.BatchCreateAccessRequestsRequest
	var batchCreateAccessRequestsJson flags.JsonFlag

	cmd.Flags().Var(&batchCreateAccessRequestsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: requests

	cmd.Use = "batch-create-access-requests"
	cmd.Short = `Create Access Requests.`
	cmd.Long = `Create Access Requests.

  Creates access requests for Unity Catalog permissions for a specified
  principal on a securable object. This Batch API can take in multiple
  principals, securable objects, and permissions as the input and returns the
  access request destinations for each. Principals must be unique across the API
  call.

  The supported securable types are: "metastore", "catalog", "schema", "table",
  "external_location", "connection", "credential", "function",
  "registered_model", and "volume".`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := batchCreateAccessRequestsJson.Unmarshal(&batchCreateAccessRequestsReq)
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

		response, err := w.Rfa.BatchCreateAccessRequests(ctx, batchCreateAccessRequestsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range batchCreateAccessRequestsOverrides {
		fn(cmd, &batchCreateAccessRequestsReq)
	}

	return cmd
}

// start get-access-request-destinations command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getAccessRequestDestinationsOverrides []func(
	*cobra.Command,
	*catalog.GetAccessRequestDestinationsRequest,
)

func newGetAccessRequestDestinations() *cobra.Command {
	cmd := &cobra.Command{}

	var getAccessRequestDestinationsReq catalog.GetAccessRequestDestinationsRequest

	cmd.Use = "get-access-request-destinations SECURABLE_TYPE FULL_NAME"
	cmd.Short = `Get Access Request Destinations.`
	cmd.Long = `Get Access Request Destinations.

  Gets an array of access request destinations for the specified securable. Any
  caller can see URL destinations or the destinations on the metastore.
  Otherwise, only those with **BROWSE** permissions on the securable can see
  destinations.

  The supported securable types are: "metastore", "catalog", "schema", "table",
  "external_location", "connection", "credential", "function",
  "registered_model", and "volume".

  Arguments:
    SECURABLE_TYPE: The type of the securable.
    FULL_NAME: The full name of the securable.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getAccessRequestDestinationsReq.SecurableType = args[0]
		getAccessRequestDestinationsReq.FullName = args[1]

		response, err := w.Rfa.GetAccessRequestDestinations(ctx, getAccessRequestDestinationsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getAccessRequestDestinationsOverrides {
		fn(cmd, &getAccessRequestDestinationsReq)
	}

	return cmd
}

// start update-access-request-destinations command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateAccessRequestDestinationsOverrides []func(
	*cobra.Command,
	*catalog.UpdateAccessRequestDestinationsRequest,
)

func newUpdateAccessRequestDestinations() *cobra.Command {
	cmd := &cobra.Command{}

	var updateAccessRequestDestinationsReq catalog.UpdateAccessRequestDestinationsRequest
	updateAccessRequestDestinationsReq.AccessRequestDestinations = catalog.AccessRequestDestinations{}
	var updateAccessRequestDestinationsJson flags.JsonFlag

	cmd.Flags().Var(&updateAccessRequestDestinationsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: destinations

	cmd.Use = "update-access-request-destinations UPDATE_MASK SECURABLE"
	cmd.Short = `Update Access Request Destinations.`
	cmd.Long = `Update Access Request Destinations.

  Updates the access request destinations for the given securable. The caller
  must be a metastore admin, the owner of the securable, or a user that has the
  **MANAGE** privilege on the securable in order to assign destinations.
  Destinations cannot be updated for securables underneath schemas (tables,
  volumes, functions, and models). For these securable types, destinations are
  inherited from the parent securable. A maximum of 5 emails and 5 external
  notification destinations (Slack, Microsoft Teams, and Generic Webhook
  destinations) can be assigned to a securable. If a URL destination is
  assigned, no other destinations can be set.

  The supported securable types are: "metastore", "catalog", "schema", "table",
  "external_location", "connection", "credential", "function",
  "registered_model", and "volume".

  Arguments:
    UPDATE_MASK: The field mask must be a single string, with multiple fields separated by
      commas (no spaces). The field path is relative to the resource object,
      using a dot (.) to navigate sub-fields (e.g., author.given_name).
      Specification of elements in sequence or map fields is not allowed, as
      only the entire collection field can be specified. Field names must
      exactly match the resource field names.

      A field mask of * indicates full replacement. It’s recommended to
      always explicitly list the fields being updated and avoid using *
      wildcards, as it can lead to unintended results if the API changes in the
      future.
    SECURABLE: The securable for which the access request destinations are being
      retrieved.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only UPDATE_MASK as positional arguments. Provide 'securable' in your JSON input")
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
			diags := updateAccessRequestDestinationsJson.Unmarshal(&updateAccessRequestDestinationsReq.AccessRequestDestinations)
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
		updateAccessRequestDestinationsReq.UpdateMask = args[0]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &updateAccessRequestDestinationsReq.AccessRequestDestinations.Securable)
			if err != nil {
				return fmt.Errorf("invalid SECURABLE: %s", args[1])
			}

		}

		response, err := w.Rfa.UpdateAccessRequestDestinations(ctx, updateAccessRequestDestinationsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateAccessRequestDestinationsOverrides {
		fn(cmd, &updateAccessRequestDestinationsReq)
	}

	return cmd
}

// end service Rfa
